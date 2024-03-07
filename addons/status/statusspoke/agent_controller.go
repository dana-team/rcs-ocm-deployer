package statusspoke

import (
	"context"

	rcsv1alpha1 "github.com/dana-team/container-app-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CappSyncReconciler struct {
	spokeClient client.Client
	hubClient   client.Client
	Scheme      *runtime.Scheme
}

func (r *CappSyncReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithName("status-controller").WithValues("Capp", req.NamespacedName)
	logger.Info("Starting Reconcile")

	// get instance of spoke capp
	spokeCapp := &rcsv1alpha1.Capp{}
	logger.Info("Trying to fetch Capp from spoke")
	if err := r.spokeClient.Get(ctx, req.NamespacedName, spokeCapp); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("failed to find Capp")
			return ctrl.Result{}, nil
		} else {
			logger.Error(err, "failed to get Capp from spoke")
			return ctrl.Result{}, err
		}
	}
	logger.Info("fetched Capp from spoke successfully")

	// get instance of hub client cap
	logger.Info("Trying to fetch Capp from hub")
	hubCapp := &rcsv1alpha1.Capp{}
	if err := r.hubClient.Get(ctx, req.NamespacedName, hubCapp); err != nil {
		logger.Error(err, "failed to get Capp")
		return ctrl.Result{}, err

	}
	logger.Info("Fetched Capp from hub successfully")
	syncCappStatus(&spokeCapp.Status, &hubCapp.Status)
	if err := r.hubClient.Status().Update(ctx, hubCapp); err != nil {
		logger.Error(err, "failed to update status on hub cluster")
		return ctrl.Result{}, err
	}
	logger.Info("Updated Capp status successfully on hub cluster")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CappSyncReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&rcsv1alpha1.Capp{}).
		Complete(r)
}
