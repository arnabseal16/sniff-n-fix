package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	centurionv1 "ccs.sniff-n-fix.com/centurion-operator/api/v1"
)

// EventListenerReconciler reconciles a EventListener object
type EventListenerReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=centurion.ccs.sniff-n-fix.com,resources=eventlisteners,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=centurion.ccs.sniff-n-fix.com,resources=eventlisteners/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=centurion.ccs.sniff-n-fix.com,resources=eventlisteners/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=pods,verbs=delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the EventListener object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
func (r *EventListenerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("eventlistener", req.NamespacedName)

	// Fetch EventListener
	var listener centurionv1.EventListener
	err := client.IgnoreNotFound(r.Get(ctx, req.NamespacedName, &listener))
	if err != nil {
		log.Error(err, "unable to fetch EventListeners")
		return ctrl.Result{}, err
	}
	r.Log.Info(fmt.Sprintf("Reconciling for %s/%s:", listener.Namespace, listener.Name))

	if listener.IsBeingDeleted() {
		return ctrl.Result{}, nil
	}

	// Process Actions
	r.processActions(&listener)

	// Update Status
	err = client.IgnoreNotFound(r.Status().Update(ctx, &listener))
	if err != nil {
		log.Error(err, "unable to update eventlistener")
		return ctrl.Result{}, err
	}

	r.Log.Info(fmt.Sprintf("Reconciled: %s/%s", listener.Namespace, listener.Name))
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EventListenerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&centurionv1.EventListener{}).
		Complete(r)
}
