package v1alpha1

import (
	operatorv1alpha1 "github.com/openshift/api/operator/v1alpha1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ManilaProvisionerSpec defines the desired state of ManilaProvisioner
type ManilaProvisionerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	operatorv1alpha1.OperatorSpec `json:",inline"`

	// Number of replicas to deploy for a provisioner deployment.
	// Optional, defaults to 1.
	Replicas int32 `json:"replicas"`

	// Name of storage class to create. If the storage class already exists, it will not be updated.
	//// This allows users to create their storage classes in advance.
	// Required, no default.
	StorageClassName string `json:"storageClassName"`

	// The reclaim policy of the storage class
	// Optional, defaults to OpenShift default "Delete."
	ReclaimPolicy *v1.PersistentVolumeReclaimPolicy `json:"reclaimPolicy,omitempty"`

	// TODO [aos-devel] Attn: All OpenShift 4.0 Components Needing AWS credentials
	OpenStackSecrets v1.SecretReference

	// Optional, defaults to "default".
	Type *string `json:"type,omitempty"`

	// Optional, defaults to "nova".
	Zones []string `json:"zones,omitempty"`

	// Required, no default.
	Protocol Protocol `json:"protocol"`

	// Required, no default.
	Backend Backend `json:"backend"`

	// Required, no default.
	OsSecretName *string `json:"osSecretName"`

	// Optional, defaults to "default".
	OsSecretNamespace *string `json:"osSecretNamespace,omitempty"`

	// Optional, defaults to value of osSecretNamespace.
	ShareSecretNamespace *string `json:"shareSecretNamespace,omitempty"`

	// Optional, no default.
	OsShareID *string `json:"osShareID,omitempty"`

	// Optional, no default.
	OsShareName *string `json:"osShareID,omitempty"`

	// Optional, no default.
	OsShareAccessID *string `json:"osShareAccessID,omitempty"`

	// Required for backend csi-cephfs and protocol CEPHFS, no default.
	CSIDriver *string `json:"csiDriver,omitempty"`

	// Optional for backend nfs and protocol NFS, defaults to "0.0.0.0".
	NFSShareClient *string `json:"nfsShareClient,omitempty"`
}

type Protocol string

const (
	ProtocolCephFS = "CEPHFS"
	ProtocolNFS    = "NFS"
)

type Backend string

const (
	BackendCephFS    = "cephfs"
	BackendCSICephFS = "csi-cephfs"
	BackendNFS       = "nfs"
)

// ManilaProvisionerStatus defines the observed state of ManilaProvisioner
type ManilaProvisionerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	operatorv1alpha1.OperatorStatus `json:",inline"`
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
