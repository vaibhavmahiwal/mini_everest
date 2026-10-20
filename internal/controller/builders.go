package controller

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	databasesv1alpha1 "github.com/vaibhavmahiwal/mini_everest/api/v1alpha1"
	"github.com/vaibhavmahiwal/mini_everest/internal/engine"
)

// buildStatefulSet returns the desired StatefulSet for a DatabaseCluster.
// Pure function: no client, no context, safe to unit test.
func buildStatefulSet(dbc *databasesv1alpha1.DatabaseCluster, profile engine.Profile) *appsv1.StatefulSet {
	labels := map[string]string{"databasecluster": dbc.Name}

	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dbc.Name,
			Namespace: dbc.Namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    &dbc.Spec.Replicas,
			ServiceName: dbc.Name,
			Selector:    &metav1.LabelSelector{MatchLabels: labels},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
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
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "database",
							Image: profile.ContainerImage(),
							Env:   profile.Env(dbc),
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: profile.ContainerPort(),
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "data",
									MountPath: profile.DataMountPath(),
								},
							},
						},
					},
				},
			},
		},
	}
}

// buildService returns the desired headless Service for a DatabaseCluster.
func buildService(dbc *databasesv1alpha1.DatabaseCluster, profile engine.Profile) *corev1.Service {
	labels := map[string]string{"databasecluster": dbc.Name}
	port := profile.ContainerPort()

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dbc.Name,
			Namespace: dbc.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector:  labels,
			ClusterIP: corev1.ClusterIPNone,
			Ports: []corev1.ServicePort{
				{
					Port:       port,
					TargetPort: intstr.FromInt32(port),
				},
			},
		},
	}
}
