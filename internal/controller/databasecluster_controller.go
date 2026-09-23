package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	databasesv1alpha1 "github.com/vaibhavmahiwal/mini_everest/api/v1alpha1"
)

// DatabaseClusterReconciler reconciles a DatabaseCluster object
type DatabaseClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=databases.mini-everest.io,resources=databaseclusters,verbs=get;list;watch
// +kubebuilder:rbac:groups=databases.mini-everest.io,resources=databaseclusters/status,verbs=get;update;patch

//this function is called when a database cluster is created ,updated or deleted
//it is responsible for reconciling the state of the cluster with the desired state defined
//in the DatabaseCluster resource

func (r *DatabaseClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var dbc databasesv1alpha1.DatabaseCluster
	if err := r.Get(ctx, req.NamespacedName, &dbc); err != nil {
		if errors.IsNotFound(err) {
			// object was deleted; nothing to do
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	logger.Info("reconciling DatabaseCluster", "name", dbc.Name, "replicas", dbc.Spec.Replicas)

	return ctrl.Result{}, nil
}

// this function sets up the controller with the manager, specifying that it will watch for changes to DatabaseCluster resources
func (r *DatabaseClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&databasesv1alpha1.DatabaseCluster{}).
		Complete(r)
}
