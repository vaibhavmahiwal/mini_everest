package controller

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
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

	if err := r.reconcileService(ctx, &dbc); err != nil {
		logger.Error(err, "failed to reconcile service")
		return ctrl.Result{}, err
	}

	if err := r.updateStatus(ctx, &dbc); err != nil {
		logger.Error(err, "failed to update status")
		return ctrl.Result{}, err
	}

	if err := r.reconcileBackupCronJob(ctx, &dbc); err != nil {
		logger.Error(err, "failed to reconcile backup cronjob")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

//adding StorageSize in

func (r *DatabaseClusterReconciler) reconcileStatefulSet(ctx context.Context, dbc *databasesv1alpha1.DatabaseCluster) error {
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dbc.Name,      //name:what aresource is called
			Namespace: dbc.Namespace, //namespace is where it belongs
		},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, sts, func() error {
		labels := map[string]string{"databasecluster": dbc.Name}

		sts.Spec.Replicas = &dbc.Spec.Replicas
		//adding storage size to the statefulset spec
		sts.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{
			{
				//how to do this in a better way
				ObjectMeta: metav1.ObjectMeta{
					Name: "data",
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse(dbc.Spec.StorageSize),
						},
					},
				},
			},
		}
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
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "data",
								MountPath: "/var/lib/postgresql/data",
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

// adding the headless service second owned resource
func (r *DatabaseClusterReconciler) reconcileService(ctx context.Context, dbc *databasesv1alpha1.DatabaseCluster) error {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dbc.Name,
			Namespace: dbc.Namespace,
		},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, svc, func() error {
		labels := map[string]string{"databasecluster": dbc.Name}
		svc.Spec.Selector = labels
		svc.Spec.ClusterIP = corev1.ClusterIPNone
		svc.Spec.Ports = []corev1.ServicePort{
			{
				Port:       5432,
				TargetPort: intstr.FromInt(5432),
			},
		}
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
	if dbc.Status.ReadyReplicas != sts.Status.ReadyReplicas {
		return nil //already up to date,no need to update
	}
	dbc.Status.ReadyReplicas = sts.Status.ReadyReplicas
	return r.Status().Update(ctx, dbc)
}

func (r *DatabaseClusterReconciler) reconcileBackupCronJob(ctx context.Context, dbc *databasesv1alpha1.DatabaseCluster) error {
	cj := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dbc.Name,
			Namespace: dbc.Namespace,
		},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, cj, func() error {
		backupLabels := map[string]string{"backup-for": dbc.Name}
		//this is the schedule for the cronjob to run, which is taken from the DatabaseCluster spec
		cj.Spec.Schedule = dbc.Spec.BackupSchedule
		//a job template whose container run a backup command against a database
		cj.Spec.ConcurrencyPolicy = batchv1.ForbidConcurrent

		cj.Spec.JobTemplate = batchv1.JobTemplateSpec{
			Spec: batchv1.JobSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: backupLabels},
					Spec: corev1.PodSpec{
						RestartPolicy: corev1.RestartPolicyOnFailure,
						Containers:    []corev1.Container{},
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
