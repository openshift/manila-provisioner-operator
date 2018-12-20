# manilla-provisioner-operator
Operator for the manilla provisioner

## Usage

### OLM
> TODO

### YAML
1. Create CRD, namespace, RBAC rules, service account, & operator
```
$ oc apply -f ./deploy/
```
2. Create a CR
```
$ oc create -f ./deploy/crds/manila_v1alpha1_manilaprovisioner_cr.yaml -n=openshift-manila-provisioner-operator
```
