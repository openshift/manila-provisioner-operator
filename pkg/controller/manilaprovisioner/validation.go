package manilaprovisioner

import (
	"github.com/openshift/manila-provisioner-operator/pkg/apis/manila/v1alpha1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func (r *ReconcileManilaProvisioner) validateManilaProvisioner(instance *v1alpha1.ManilaProvisioner) field.ErrorList {
	var errs field.ErrorList

	errs = append(errs, r.validateCSIDriver(instance.Spec, instance.Spec.CSIDriver, field.NewPath("spec").Child("csiDriver"))...)
	return errs
}

func (r *ReconcileManilaProvisioner) validateCSIDriver(spec v1alpha1.ManilaProvisionerSpec, csiDriver *string, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}

	if csiDriver == nil && spec.Backend == v1alpha1.BackendCSICephFS && spec.Protocol == v1alpha1.ProtocolCephFS {
		errs = append(errs, field.Required(fldPath, "name of the CSI driver is required when backend is csi-cephfs and protocol is CEPHFS"))
	}

	return errs
}
