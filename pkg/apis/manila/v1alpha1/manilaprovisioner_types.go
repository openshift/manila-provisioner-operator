package v1alpha1

import (
	operatorv1alpha1 "github.com/openshift/api/operator/v1alpha1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

const (
	defaultImagePullSpec = "openshift/origin-manila-provisioner:latest"
	defaultVersion       = "4.0.0"
)

// SetDefaults sets the default vaules for the external provisioner spec and returns true if the spec was changed
func (p *ManilaProvisioner) SetDefaults() bool {
	changed := false
	ps := &p.Spec
	if len(ps.ManagementState) == 0 {
		ps.ManagementState = "Managed"
		changed = true
	}
	if len(ps.ImagePullSpec) == 0 {
		ps.ImagePullSpec = defaultImagePullSpec
		changed = true
	}
	if len(ps.Version) == 0 {
		ps.Version = defaultVersion
		changed = true
	}
	if ps.ReclaimPolicy == nil {
		reclaimPolicy := v1.PersistentVolumeReclaimDelete
		ps.ReclaimPolicy = &reclaimPolicy
		changed = true
	}
	if ps.Replicas == 0 {
		ps.Replicas = 1
		changed = true
	}
	if ps.Type == nil {
		sType := "default"
		ps.Type = &sType
		changed = true
	}
	if ps.Zones == nil {
		zones := []string{"nova"}
		ps.Zones = zones
		changed = true
	}
	if ps.OsSecretNamespace == nil {
		osSecretNamespace := "default"
		ps.OsSecretNamespace = &osSecretNamespace
		changed = true
	}
	if ps.ShareSecretNamespace == nil {
		shareSecretNamespace := *ps.OsSecretNamespace
		ps.ShareSecretNamespace = &shareSecretNamespace
		changed = true
	}
	if ps.Backend == BackendNFS && ps.Protocol == ProtocolNFS && ps.NFSShareClient == nil {
		nfsShareClient := "0.0.0.0"
		ps.NFSShareClient = &nfsShareClient
		changed = true
	}
	return changed
}

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
	ProtocolCephFS Protocol = "CEPHFS"
	ProtocolNFS    Protocol = "NFS"
)

type Backend string

const (
	BackendCephFS    Backend = "cephfs"
	BackendCSICephFS Backend = "csi-cephfs"
	BackendNFS       Backend = "nfs"
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
