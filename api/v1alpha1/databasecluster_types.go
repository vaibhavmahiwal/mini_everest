package main

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DataBaseClusterSpec defines the desired state of DataBaseCluster
type DataBaseClusterSpec struct {
	Replicas int32 `json:"replicas"`
}

// DataBaseClusterStatus defines the observed state of DataBaseCluster
type DataBaseClusterStatus struct {
	ReadyReplicas int32 `json:"readyreplicas"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// DataBaseCluster is the Schema for the databaseclusters API
type DataBaseCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DataBaseClusterSpec   `json:"spec,omitempty"`
	Status DataBaseClusterStatus `json:"status,omitempty"`
}

//kubebuilder:object:root=true

// DataBaseClusterList contains a list of DataBaseCluster
type DataBaseClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []DataBaseCluster `json:"items"`
}
