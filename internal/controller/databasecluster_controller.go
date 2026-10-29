package controller

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	databasesv1alpha1 "github.com/vaibhavmahiwal/mini_everest/api/v1alpha1"
	"github.com/vaibhavmahiwal/mini_everest/internal/engine"
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
// +kubebuilder:rbac:groups=batch,resources=cronjobs,verbs=get;list;watch;create;update;patch;delete

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

	profile, err := engine.For(dbc.Spec.Engine)
	if err != nil {
		logger.Error(err, "unsupported engine", "engine", dbc.Spec.Engine)
		return ctrl.Result{}, err
	}

	if err := r.reconcileStatefulSet(ctx, &dbc, profile); err != nil {
		logger.Error(err, "failed to reconcile statefulset")
		return ctrl.Result{}, err
	}

	if err := r.reconcileService(ctx, &dbc, profile); err != nil {
		logger.Error(err, "failed to reconcile service")
		return ctrl.Result{}, err
	}

	if err := r.reconcileBackupCronJob(ctx, &dbc, profile); err != nil {
		logger.Error(err, "failed to reconcile backup cronjob")
		return ctrl.Result{}, err
	}

	if err := r.updateStatus(ctx, &dbc); err != nil {
		logger.Error(err, "failed to update status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

//adding StorageSize in

func (r *DatabaseClusterReconciler) reconcileStatefulSet(ctx context.Context, dbc *databasesv1alpha1.DatabaseCluster, profile engine.Profile) error {
	sts := &appsv1.StatefulSet{}
	sts.Name = dbc.Name
	sts.Namespace = dbc.Namespace
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, sts, func() error {
		desired, err := buildStatefulSet(dbc, profile)
		if err != nil {
			return err
		}
		sts.Spec = desired.Spec
		sts.Labels = desired.Labels
		return ctrl.SetControllerReference(dbc, sts, r.Scheme)
	})
	return err
}

// adding the headless service second owned resource
func (r *DatabaseClusterReconciler) reconcileService(ctx context.Context, dbc *databasesv1alpha1.DatabaseCluster, profile engine.Profile) error {
	svc := &corev1.Service{}
	svc.Name = dbc.Name
	svc.Namespace = dbc.Namespace
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, svc, func() error {
		desired := buildService(dbc, profile)
		svc.Spec = desired.Spec
		svc.Labels = desired.Labels
		return ctrl.SetControllerReference(dbc, svc, r.Scheme)
	})
	return err
}

// creating this seperately so that could be unit tested on its own later
// this func read compare update if different
func (r *DatabaseClusterReconciler) updateStatus(ctx context.Context, dbc *databasesv1alpha1.DatabaseCluster) error {
	var sts appsv1.StatefulSet
	if err := r.Get(ctx, client.ObjectKey{Name: dbc.Name, Namespace: dbc.Namespace}, &sts); err != nil {
		return err
	}
	if dbc.Status.ReadyReplicas == sts.Status.ReadyReplicas {
		return nil
	}
	dbc.Status.ReadyReplicas = sts.Status.ReadyReplicas
	return r.Status().Update(ctx, dbc)
}

// this function reconciles the backup cronjob for the database cluster
func (r *DatabaseClusterReconciler) reconcileBackupCronJob(ctx context.Context, dbc *databasesv1alpha1.DatabaseCluster, profile engine.Profile) error {
	cj := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dbc.Name + "-backup",
			Namespace: dbc.Namespace,
		},
	}
	if dbc.Spec.BackupSchedule == "" {
		return client.IgnoreNotFound(r.Delete(ctx, cj))
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, cj, func() error {
		backupLabels := map[string]string{"backup-for": dbc.Name}
		cj.Spec.Schedule = dbc.Spec.BackupSchedule
		cj.Spec.ConcurrencyPolicy = batchv1.ForbidConcurrent
		successfulJobsHistoryLimit := int32(2)
		failedJobsHistoryLimit := int32(1)
		cj.Spec.SuccessfulJobsHistoryLimit = &successfulJobsHistoryLimit
		cj.Spec.FailedJobsHistoryLimit = &failedJobsHistoryLimit

		cj.Spec.JobTemplate = batchv1.JobTemplateSpec{
			Spec: batchv1.JobSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: backupLabels},
					Spec: corev1.PodSpec{
						RestartPolicy: corev1.RestartPolicyOnFailure,
						Containers: []corev1.Container{
							{
								Name:    "backup",
								Image:   profile.ContainerImage(),
								Command: profile.BackupCommand(dbc),
								Env:     profile.BackupEnv(dbc),
							},
						},
					},
				},
			},
		}
		return ctrl.SetControllerReference(dbc, cj, r.Scheme)
	})
	return err
}

// this function sets up the controller with the manager, specifying that it will watch for changes to DatabaseCluster resources
func (r *DatabaseClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&databasesv1alpha1.DatabaseCluster{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&corev1.Service{}).
		Owns(&batchv1.CronJob{}).
		Complete(r)
}
