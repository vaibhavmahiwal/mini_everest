package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DatabaseClusterSpec defines the desired state of DatabaseCluster
type DatabaseClusterSpec struct {
	// Engine defines the database engine (e.g. postgresql, mysql)
	Engine string `json:"engine"`

	// Replicas defines the desired number of pods
	Replicas int32 `json:"replicas"`

	// StorageSize defines the volume claim storage size (e.g. 1Gi)
	StorageSize string `json:"storageSize"`
}

// DatabaseClusterStatus defines the observed state of DatabaseCluster
type DatabaseClusterStatus struct {
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// DatabaseCluster is the Schema for the databaseclusters API
type DatabaseCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DatabaseClusterSpec   `json:"spec,omitempty"`
	Status DatabaseClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DatabaseClusterList contains a list of DatabaseCluster
type DatabaseClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []DatabaseCluster `json:"items"`
}
