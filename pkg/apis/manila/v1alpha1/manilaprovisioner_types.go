package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ManilaProvisionerSpec defines the desired state of ManilaProvisioner
type ManilaProvisionerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
}

// ManilaProvisionerStatus defines the observed state of ManilaProvisioner
type ManilaProvisionerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ManilaProvisioner is the Schema for the manilaprovisioners API
// +k8s:openapi-gen=true
type ManilaProvisioner struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ManilaProvisionerSpec   `json:"spec,omitempty"`
	Status ManilaProvisionerStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ManilaProvisionerList contains a list of ManilaProvisioner
type ManilaProvisionerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ManilaProvisioner `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ManilaProvisioner{}, &ManilaProvisionerList{})
}
