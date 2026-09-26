package controller

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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
// +kubebuilder:rbac:groups=databases.mini-everest.io,resources=databaseclusters/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services;events,verbs=get;list;watch;create;update;patch;delete

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

	if err := r.reconcileStatefulSet(ctx, &dbc); err != nil {
		logger.Error(err, "failed to reconcile statefulset")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *DatabaseClusterReconciler) reconcileStatefulSet(ctx context.Context, dbc *databasesv1alpha1.DatabaseCluster) error {
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dbc.Name,
			Namespace: dbc.Namespace,
		},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, sts, func() error {
		labels := map[string]string{"databasecluster": dbc.Name}

		sts.Spec.Replicas = &dbc.Spec.Replicas
		sts.Spec.ServiceName = dbc.Name
		sts.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}
		sts.Spec.Template = corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{Labels: labels},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "database",
						Image: "postgres:16",
						Env: []corev1.EnvVar{
							{
								Name:  "POSTGRES_PASSWORD",
								Value: "postgres",
							},
						},
						Ports: []corev1.ContainerPort{
							{
								ContainerPort: 5432,
							},
						},
					},
				},
			},
		}
		return ctrl.SetControllerReference(dbc, sts, r.Scheme)
	})
	return err
}

// this function sets up the controller with the manager, specifying that it will watch for changes to DatabaseCluster resources
func (r *DatabaseClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&databasesv1alpha1.DatabaseCluster{}).
		Owns(&appsv1.StatefulSet{}).
		Complete(r)
}
