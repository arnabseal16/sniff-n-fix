package sqs

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	centurionv1 "ccs.sniff-n-fix.com/centurion-operator/api/v1"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SQSMessageAPI defines the interface for the GetQueueUrl function.
// We use this interface to test the function using a mocked service.
type SQSMessageAPI interface {
	GetQueueUrl(ctx context.Context,
		params *sqs.GetQueueUrlInput,
		optFns ...func(*sqs.Options)) (*sqs.GetQueueUrlOutput, error)

	ReceiveMessage(ctx context.Context,
		params *sqs.ReceiveMessageInput,
		optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error)

	DeleteMessage(ctx context.Context,
		params *sqs.DeleteMessageInput,
		optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error)
}

func GetClient() *sqs.Client {
	var role string
	role = os.Getenv("AWS_ROLE_ARN")

	cfg, err := config.LoadDefaultConfig(context.TODO())
	cfg.Credentials = stscreds.NewAssumeRoleProvider(sts.NewFromConfig(cfg), role)

	if err != nil {
		panic("configuration error, " + err.Error())
	}
	client := sqs.NewFromConfig(cfg)
	return client
}

func getQueueUrl(queue *string, client *sqs.Client) *sqs.GetQueueUrlOutput {
	gQInput := &sqs.GetQueueUrlInput{
		QueueName: queue,
	}

	// Get URL of queue
	urlResult, err := GetQueueURL(context.TODO(), client, gQInput)
	if err != nil {
		fmt.Println("Got an error getting the queue URL:")
		fmt.Println(err)
		return nil
	}

	return urlResult
}

// GetQueueURL gets the URL of an Amazon SQS queue.
// Inputs:
//     c is the context of the method call, which includes the AWS Region.
//     api is the interface that defines the method call.
//     input defines the input arguments to the service call.
// Output:
//     If success, a GetQueueUrlOutput object containing the result of the service call and nil.
//     Otherwise, nil and an error from the call to GetQueueUrl.
func GetQueueURL(c context.Context, api SQSMessageAPI, input *sqs.GetQueueUrlInput) (*sqs.GetQueueUrlOutput, error) {
	return api.GetQueueUrl(c, input)
}

// GetMessages gets the most recent message from an Amazon SQS queue.
// Inputs:
//     c is the context of the method call, which includes the AWS Region.
//     api is the interface that defines the method call.
//     input defines the input arguments to the service call.
// Output:
//     If success, a ReceiveMessageOutput object containing the result of the service call and nil.
//     Otherwise, nil and an error from the call to ReceiveMessage.
func GetMessages(c context.Context, api SQSMessageAPI, input *sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error) {
	return api.ReceiveMessage(c, input)
}

// RemoveMessage deletes a message from an Amazon SQS queue.
// Inputs:
//     c is the context of the method call, which includes the AWS Region.
//     api is the interface that defines the method call.
//     input defines the input arguments to the service call.
// Output:
//     If success, a DeleteMessageOutput object containing the result of the service call and nil.
//     Otherwise, nil and an error from the call to DeleteMessage.
func RemoveMessage(c context.Context, api SQSMessageAPI, input *sqs.DeleteMessageInput) (*sqs.DeleteMessageOutput, error) {
	return api.DeleteMessage(c, input)
}

func SqsListener(chn chan<- *sqstypes.Message, queue *string, log logr.Logger) {
	var timeout int32

	if *queue == "" {
		fmt.Println("You must supply the name of a queue (-q QUEUE)")
		return
	}

	if timeout < 0 {
		timeout = 0
	}

	if timeout > 12*60*60 {
		timeout = 12 * 60 * 60
	}

	client := GetClient()

	queueURL := getQueueUrl(queue, client).QueueUrl

	for {

		output, err := GetMessages(context.TODO(), client, &sqs.ReceiveMessageInput{
			QueueUrl:            queueURL,
			MaxNumberOfMessages: int32(1),
			WaitTimeSeconds:     int32(10),
		})

		if err != nil {
			log.Error(err, "failed to fetch sqs message")
		}

		for _, message := range output.Messages {
			chn <- &message
		}
	}
}

type EventMessage struct {
	Type          string  `json:"type"`
	Target        string  `json:"target"`
	Action        string  `json:"action"`
	Namespace     string  `json:"namespace"`
	ReceiptHandle *string `json:"receipthandle"`
}

// message format expected from datadog:
// {"action": "delete","target": "ingress-kong-746bc7b64d-f4kqr","namespace": "atmos-system","type": "pod"}===

func HandleMessage(msgResult *sqstypes.Message, eventListenerClient client.Client) {
	fmt.Println("Message Body: " + *msgResult.Body)

	var eventMessage EventMessage
	msgBody := strings.Split(*msgResult.Body, "===")[0]
	err := json.Unmarshal([]byte(msgBody), &eventMessage)
	eventMessage.ReceiptHandle = msgResult.ReceiptHandle

	// if strings.ToLower(eventMessage.Type) == "pod" {
	// 	podClient := clientset.CoreV1().Pods(eventMessage.Namespace)
	// 	fmt.Println(eventMessage.Namespace)
	// 	podName := strings.ToLower(eventMessage.Target)
	// 	if strings.ToLower(eventMessage.Action) == "restart" {
	// 		err := podClient.Delete(context.Background(), podName, metav1.DeleteOptions{})
	// 		if err != nil {
	// 			fmt.Println("There was an error: " + err.Error())
	// 		}
	// 	}
	// 	// podObj, _ := podClient.Get(context.Background(), podName, metav1.GetOptions{})
	// }

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	eventListener, err := CreateEventListener(eventMessage)

	if err != nil {
		fmt.Println(err.Error())
	}

	result, err := UpsertEventListener(eventListenerClient, eventListener)

	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println(&result)

}

func DeleteMessage(rHandle *string, queueName *string) error {
	client := GetClient()

	queueURL := getQueueUrl(queueName, client).QueueUrl

	dMInput := &sqs.DeleteMessageInput{
		QueueUrl:      queueURL,
		ReceiptHandle: rHandle,
	}

	_, err := RemoveMessage(context.TODO(), client, dMInput)
	return err
}

func CreateEventListener(eventMessage EventMessage) (*centurionv1.EventListener, error) {
	labels, err := getLabels()
	if err != nil {
		return nil, err
	}

	event := &centurionv1.EventListener{}
	event.ObjectMeta.Namespace = eventMessage.Namespace
	event.ObjectMeta.Name = eventMessage.Target
	event.ObjectMeta.Labels = labels
	event.Spec.Actions = []centurionv1.EventListenerAction{
		{
			ActionType:    centurionv1.ActionType(eventMessage.Action),
			ResourceType:  centurionv1.ResourceType(eventMessage.Type),
			Target:        eventMessage.Target,
			ReceiptHandle: eventMessage.ReceiptHandle,
		},
	}

	return event, nil
}

func UpsertEventListener(c client.Client, eventListener *centurionv1.EventListener) (*centurionv1.EventListener, error) {
	ctx := context.Background()
	key := client.ObjectKeyFromObject(eventListener)
	existing := &centurionv1.EventListener{}

	if err := c.Get(ctx, key, existing); err != nil {
		if !errors.IsNotFound(err) {
			return nil, err
		}
		if err := c.Create(ctx, eventListener); err != nil {
			return nil, err
		}
		return eventListener, nil
	}

	if equality.Semantic.DeepDerivative(existing, eventListener) {
		return existing, nil
	}

	existing.Spec = eventListener.DeepCopy().Spec
	if err := c.Update(ctx, existing); err != nil {
		return nil, err
	}

	return existing, nil
}

func getLabels() (map[string]string, error) {
	labels := os.Getenv("EVENT_LISTENER_LABELS")
	if len(labels) == 0 {
		return map[string]string{}, nil
	}
	var labelJson map[string]string
	err := json.Unmarshal([]byte(labels), &labelJson)
	if err != nil {
		return nil, err
	}

	return labelJson, nil
}
