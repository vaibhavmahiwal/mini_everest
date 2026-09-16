// package v1alpha1 contains the API schemadefinitions for the mini_everest v1alpha1 API group
// +kubebuilder:object:generate=true
// +groupName=mini_everest.io
package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	//groupversion is a schema Groupversion for the mini_everest API
	GroupVersion = schema.GroupVersion{
		Group:   "databases.mini-everest.io",
		Version: "v1alpha1",
	}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}
	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)
