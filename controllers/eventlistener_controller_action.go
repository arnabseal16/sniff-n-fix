package controllers

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	snfv1 "ccs.sniff-n-fix.com/snf-operator/api/v1"

	//"ccs.sniff-n-fix.com/snf-operator/pkg/sqs"

	snf_sqs "ccs.sniff-n-fix.com/snf-operator/pkg/sqs"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *EventListenerReconciler) processActions(listener *snfv1.EventListener) {
	for _, action := range listener.Spec.Actions {
		wg := new(sync.WaitGroup)
		conditionType := snfv1.GetConditionType(action)
		err := r.processAction(listener, action, wg)
		if err != nil {
			reason := fmt.Sprintf("Failed to execute '%s'", getActionLabel(listener, action))
			listener.Status.SetCondition(conditionType, snfv1.ConditionFalse, reason, err.Error())
			r.Log.Error(err, reason)
		} else {
			reason := fmt.Sprintf("Exected '%s'", getActionLabel(listener, action))
			listener.Status.SetCondition(conditionType, snfv1.ConditionTrue, reason, "Action Complete")
			r.Log.Info(reason)
		}
	}
}

func (r *EventListenerReconciler) processAction(listener *snfv1.EventListener, action snfv1.EventListenerAction, wg *sync.WaitGroup) error {
	resource, err := r.newResource(action.ResourceType, listener.Name, listener.Namespace)
	if err != nil {
		return err
	}

	// Gets existing queue
	queue_client := snf_sqs.GetClient()
	queue_ptr, err := queue_client.GetQueueUrl(context.Background(), nil)
	queue := queue_ptr.QueueUrl

	if err != nil {
		return err
	}

	r.Log.Info(fmt.Sprintf("Executing %s", getActionLabel(listener, action)))

	//Action: Delete if detected
	if action.ActionType == snfv1.ActionDelete {
		err = client.IgnoreNotFound(r.Client.Delete(context.Background(), resource))
		if err != nil {
			return err
		}
	} else if action.ActionType == snfv1.ActionScaleUp { //Action: Scale Up if detected
		dep := &appsv1.Deployment{}
		_ = r.Client.Get(context.Background(), client.ObjectKey{
			Namespace: resource.GetNamespace(),
			Name:      resource.GetName(),
		}, dep)

		runnning_replicas := dep.Spec.Replicas
		updated_replicas := int(math.Ceil(float64(*runnning_replicas) * 1.4))

		//get_labels := dep.Labels

		patch_replica := []byte(fmt.Sprintf(`{"spec":{"replicas":%d}}`, updated_replicas))

		_ = r.Client.Patch(context.Background(), &appsv1.Deployment{ //patch for updating replica
			ObjectMeta: metav1.ObjectMeta{
				Namespace: resource.GetNamespace(),
				Name:      resource.GetName(),
			},
		}, client.RawPatch(types.StrategicMergePatchType, patch_replica))

		//Create Label String to track Scale
		timestamp := string(strconv.FormatInt(time.Now().UTC().UnixNano(), 10))
		label := fmt.Sprintf("%s:true:%d", timestamp, runnning_replicas) //Label syntax: <time-stamp>:<bool>:<replica number> eg: 1257894000000000000:true:10
		patch_label := []byte(fmt.Sprintf(`{"metadata":{"labels":{"snfKey":%s}}}`, label))

		_ = r.Client.Patch(context.Background(), &appsv1.Deployment{ //patch for updatng label
			ObjectMeta: metav1.ObjectMeta{
				Namespace: resource.GetNamespace(),
				Name:      resource.GetName(),
			},
		}, client.RawPatch(types.StrategicMergePatchType, patch_label))

		var actionrecurse snfv1.EventListenerAction

		//Create Action for Scale Down and start a sync recursion
		actionrecurse.ActionType = snfv1.ActionScaleDown
		actionrecurse.ReceiptHandle = action.ReceiptHandle
		actionrecurse.ResourceType = action.ResourceType
		actionrecurse.Target = action.Target

		wg.Add(1)                                          //Add Counter for WaitGroup
		err = r.processAction(listener, actionrecurse, wg) //Create the recursion

	} else if action.ActionType == snfv1.ActionScaleDown { //Action: Scale Down if detected
		dep := &appsv1.Deployment{}
		_ = r.Client.Get(context.Background(), client.ObjectKey{
			Namespace: resource.GetNamespace(),
			Name:      resource.GetName(),
		}, dep)

		get_labels := dep.Labels

		timeDiff := getTimeDiff((strings.Split(get_labels["snfKey"], ":"))[0], string(strconv.FormatInt(time.Now().UTC().UnixNano(), 10)))

		if timeDiff < 10 {
			time.Sleep(5 * time.Minute)
			go r.processAction(listener, action, wg)
		}

		original_relica := (strings.Split(get_labels["snfKey"], ":"))[2] //fetch original replica value

		patch_replica := []byte(fmt.Sprintf(`{"spec":{"replicas":%s}}`, original_relica)) //create patch with original replica value (To scale down)

		_ = r.Client.Patch(context.Background(), &appsv1.Deployment{ //patch for updating replica
			ObjectMeta: metav1.ObjectMeta{
				Namespace: resource.GetNamespace(),
				Name:      resource.GetName(),
			},
		}, client.RawPatch(types.StrategicMergePatchType, patch_replica))

		//UpdateLabel to nill
		patch_label := []byte(`{"metadata":{"labels":{"snfKey":":::"}}}`)

		_ = r.Client.Patch(context.Background(), &appsv1.Deployment{ //patch for updating label
			ObjectMeta: metav1.ObjectMeta{
				Namespace: resource.GetNamespace(),
				Name:      resource.GetName(),
			},
		}, client.RawPatch(types.StrategicMergePatchType, patch_label))

	}

	if err != nil {
		var rHandle *string = action.ReceiptHandle //Fetch receipt Handle of message

		if action.ActionType == snfv1.ActionDelete {
			r.Log.Info(fmt.Sprintf("Removing '%s' from queue '%s'", getActionLabel(listener, action), queue))
			err = snf_sqs.DeleteMessage(rHandle, queue)
		} else if action.ActionType == snfv1.ActionScaleUp {
			r.Log.Info(fmt.Sprintf("Scaling up '%s' by a factor of 40 percentage from queue '%s'", getActionLabel(listener, action), queue))
		} else if action.ActionType == snfv1.ActionScaleDown {
			r.Log.Info(fmt.Sprintf("Scaling down '%s' to original replica after cool down from queue '%s'", getActionLabel(listener, action), queue))
			err = snf_sqs.DeleteMessage(rHandle, queue)
			defer wg.Done() //Decrement Counter for WaitGroup
		} else {
			r.Log.Info(fmt.Sprintf("Action Performed:  '%s' from queue '%s'", getActionLabel(listener, action), queue))
			err = snf_sqs.DeleteMessage(rHandle, queue)
		}
	}
	return err
}

func getActionLabel(listener *snfv1.EventListener, action snfv1.EventListenerAction) string {
	return fmt.Sprintf("%s/%s - %s", listener.Namespace, listener.Name, action.ActionType)
}

func getTimeDiff(labelTime string, presTime string) int {
	tdt1, err1 := strconv.Atoi(labelTime)
	tdt2, err2 := strconv.Atoi(presTime)

	if err1 != nil || err2 != nil {
		fmt.Println(err1, err2)
	}

	return (tdt2 - tdt1) / 60000000000
}

// func toEventMessage(listener *snfv1.EventListener, action snfv1.EventListenerAction) string {

// 	// msg := snf_sqs.EventMessage {
// 	// 	Type: string(action.ResourceType),
// 	// 	Target: listener.Name,
// 	// 	Action: string(action.ActionType),
// 	// 	Namespace: listener.Namespace,
// 	// }
// 	// return msg

// 	// msg := *sqstypes.Message{

// 	// }

// 	return action.ReceiptHandle
// }
