package controller

import (
	"github.com/openshift/manila-provisioner-operator/pkg/controller/manilaprovisioner"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, manilaprovisioner.Add)
}
