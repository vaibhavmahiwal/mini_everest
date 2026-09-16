package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DataBaseClusterSpec defines the desired state of DataBaseCluster
type DatabaseClusterSpec struct {
	Replicas int32 `json:"replicas"`
}

// DataBaseClusterStatus defines the observed state of DataBaseCluster
type DatabaseClusterStatus struct {
	ReadyReplicas int32 `json:"readyreplicas"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// DataBaseCluster is the Schema for the databaseclusters API
type DatabaseCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DatabaseClusterSpec   `json:"spec,omitempty"`
	Status DatabaseClusterStatus `json:"status,omitempty"`
}

//kubebuilder:object:root=true

// DataBaseClusterList contains a list of DataBaseCluster
type DataBaseClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []DatabaseCluster `json:"items"`
}
