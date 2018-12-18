package manilaprovisioner

import (
	"context"
	"fmt"
	"log"
	"strings"

	operatorv1alpha1 "github.com/openshift/api/operator/v1alpha1"
	"github.com/openshift/library-go/pkg/operator/resource/resourceapply"
	"github.com/openshift/library-go/pkg/operator/resource/resourcemerge"
	"github.com/openshift/library-go/pkg/operator/resource/resourceread"
	"github.com/openshift/library-go/pkg/operator/v1alpha1helpers"
	manilav1alpha1 "github.com/openshift/manila-provisioner-operator/pkg/apis/manila/v1alpha1"
	"github.com/openshift/manila-provisioner-operator/pkg/generated"
	appsv1 "k8s.io/api/apps/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	//corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/kubernetes"
	storageclientv1 "k8s.io/client-go/kubernetes/typed/storage/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	provisionerName = "openshift.io/openstack-manila"
	leaseName       = "openshift.io-openstack-manila" // provisionerName slashes replaced with dashes
	finalizerName   = "manila.storage.openshift.io"
)

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new ManilaProvisioner Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	clientset, err := kubernetes.NewForConfig(mgr.GetConfig())
	if err != nil {
		log.Fatal(err)
	}
	return &ReconcileManilaProvisioner{client: mgr.GetClient(), clientset: clientset, scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("manilaprovisioner-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource ManilaProvisioner
	err = c.Watch(&source.Kind{Type: &manilav1alpha1.ManilaProvisioner{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource Deployments and StorageClasses and requeue the owner ManilaProvisioner
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &manilav1alpha1.ManilaProvisioner{},
	})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &storagev1.StorageClass{}}, &handler.EnqueueRequestsFromMapFunc{
		ToRequests: handler.ToRequestsFunc(func(a handler.MapObject) []reconcile.Request {
			return []reconcile.Request{
				{NamespacedName: types.NamespacedName{
					Namespace: a.Meta.GetLabels()[OwnerLabelNamespace],
					Name:      a.Meta.GetLabels()[OwnerLabelName],
				}},
			}
		}),
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileManilaProvisioner{}

// ReconcileManilaProvisioner reconciles a ManilaProvisioner object
type ReconcileManilaProvisioner struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client    client.Client
	clientset *kubernetes.Clientset
	scheme    *runtime.Scheme
}

// Reconcile reads that state of the cluster for a ManilaProvisioner object and makes changes based on the state read
// and what is in the ManilaProvisioner.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileManilaProvisioner) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log.Printf("Reconciling ManilaProvisioner %s/%s\n", request.Namespace, request.Name)

	// Fetch the ManilaProvisioner instance
	instance := &manilav1alpha1.ManilaProvisioner{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	pr := instance
	switch pr.Spec.ManagementState {
	case operatorv1alpha1.Unmanaged:
		return reconcile.Result{}, nil

	case operatorv1alpha1.Removed:
		err := r.cleanup(pr)
		if err != nil {
			log.Printf("error cleaning up: %v", err)
			return reconcile.Result{}, err
		}
		pr.Status.TaskSummary = "Remove"
		pr.Status.TargetAvailability = nil
		pr.Status.CurrentAvailability = nil
		pr.Status.Conditions = []operatorv1alpha1.OperatorCondition{
			{
				Type:   operatorv1alpha1.OperatorStatusTypeAvailable,
				Status: operatorv1alpha1.ConditionFalse,
			},
		}
		return reconcile.Result{}, r.client.Update(context.TODO(), pr)
	}

	if pr.DeletionTimestamp != nil {
		err := r.cleanup(pr)
		if err != nil {
			log.Printf("error cleaning up: %v", err)
			return reconcile.Result{}, err
		}
	}

	// Simulate initializer.
	if pr.SetDefaults() {
		err := r.client.Update(context.TODO(), pr)
		if err != nil {
			log.Printf("error setting defaults: %v", err)
		}
		return reconcile.Result{}, err
	}

	// Validate and don't sync anything if there are errors
	validationErrors := r.validateManilaProvisioner(pr)
	if len(validationErrors) > 0 {
		log.Printf("validation errors: %v", validationErrors)
		errors := []error{}
		for _, err := range validationErrors {
			errors = append(errors, err)
		}
		return reconcile.Result{}, utilerrors.NewAggregate(errors)
	}

	err = r.syncFinalizer(pr)
	if err != nil {
		return reconcile.Result{}, err
	}

	errors := r.syncRBAC(pr)

	err = r.syncStorageClass(pr)
	if err != nil {
		errors = append(errors, fmt.Errorf("error syncing storageClass: %v", err))
	}

	previousAvailability := pr.Status.CurrentAvailability
	forceDeployment := pr.ObjectMeta.Generation != pr.Status.ObservedGeneration
	deployment, err := r.syncDeployment(pr, previousAvailability, forceDeployment)
	if err != nil {
		errors = append(errors, fmt.Errorf("error syncing deployment: %v", err))
	}

	err = r.syncStatus(pr, deployment, errors)
	if err != nil {
		errors = append(errors, fmt.Errorf("error syncing status: %v", err))
	}

	if len(errors) > 0 {
		log.Printf("errors: %v", errors)
	}
	return reconcile.Result{}, utilerrors.NewAggregate(errors)
}

func (r *ReconcileManilaProvisioner) syncFinalizer(pr *manilav1alpha1.ManilaProvisioner) error {
	if hasFinalizer(pr.Finalizers, finalizerName) {
		return nil
	}

	if pr.Finalizers == nil {
		pr.Finalizers = []string{}
	}
	pr.Finalizers = append(pr.Finalizers, finalizerName)

	return r.client.Update(context.TODO(), pr)
}

func hasFinalizer(finalizers []string, finalizerName string) bool {
	for _, f := range finalizers {
		if f == finalizerName {
			return true
		}
	}
	return false
}

func (r *ReconcileManilaProvisioner) syncRBAC(pr *manilav1alpha1.ManilaProvisioner) []error {
	selector := labelsForProvisioner(pr)

	errors := []error{}

	serviceAccount := resourceread.ReadServiceAccountV1OrDie(generated.MustAsset("assets/serviceaccount.yaml"))
	serviceAccount.SetNamespace(pr.GetNamespace())
	serviceAccount.SetLabels(selector)
	if err := controllerutil.SetControllerReference(pr, serviceAccount, r.scheme); err != nil {
		errors = append(errors, err)
	}
	_, _, err := resourceapply.ApplyServiceAccount(r.clientset.CoreV1(), serviceAccount)
	if err != nil {
		errors = append(errors, fmt.Errorf("error applying serviceAccount: %v", err))
	}

	// TODO roles
	//
	clusterRole := resourceread.ReadClusterRoleV1OrDie(generated.MustAsset("assets/clusterrole.yaml"))
	clusterRole.SetLabels(selector)
	_, _, err = resourceapply.ApplyClusterRole(r.clientset.RbacV1(), clusterRole)
	if err != nil {
		errors = append(errors, fmt.Errorf("error applying clusterRole: %v", err))
	}

	clusterRoleBinding := resourceread.ReadClusterRoleBindingV1OrDie(generated.MustAsset("assets/clusterrolebinding.yaml"))
	clusterRoleBinding.Subjects[0].Namespace = pr.GetNamespace()
	clusterRoleBinding.SetLabels(selector)
	_, _, err = resourceapply.ApplyClusterRoleBinding(r.clientset.RbacV1(), clusterRoleBinding)
	if err != nil {
		errors = append(errors, fmt.Errorf("error applying clusterRoleBinding: %v", err))
	}

	return errors
}

func (r *ReconcileManilaProvisioner) syncStorageClass(pr *manilav1alpha1.ManilaProvisioner) error {
	selector := labelsForProvisioner(pr)

	parameters := map[string]string{}
	if pr.Spec.Type != nil {
		parameters["type"] = *pr.Spec.Type
	}
	if pr.Spec.Zones != nil {
		parameters["zones"] = strings.Join(pr.Spec.Zones, ",")
	}
	parameters["protocol"] = string(pr.Spec.Protocol)
	parameters["backend"] = string(pr.Spec.Backend)
	// TODO use secretReference?
	parameters["osSecretName"] = pr.Spec.OsSecretName
	if pr.Spec.OsSecretNamespace != nil {
		parameters["osSecretNamespace"] = *pr.Spec.OsSecretNamespace
	}
	if pr.Spec.ShareSecretNamespace != nil {
		parameters["shareSecretNamespace"] = *pr.Spec.ShareSecretNamespace
	}
	if pr.Spec.OsShareID != nil {
		parameters["osShareID"] = *pr.Spec.OsShareID
	}
	if pr.Spec.OsShareName != nil {
		parameters["osShareName"] = *pr.Spec.OsShareName
	}
	if pr.Spec.OsShareAccessID != nil {
		parameters["osShareAccessID"] = *pr.Spec.OsShareAccessID
	}
	if pr.Spec.Backend == manilav1alpha1.BackendCSICephFS && pr.Spec.Protocol == manilav1alpha1.ProtocolCephFS {
		// TODO validate this in code because it's Required if, and can't do that via CRD validation
		parameters["csi-driver"] = *pr.Spec.CSIDriver
	}
	if pr.Spec.Backend == manilav1alpha1.BackendNFS && pr.Spec.Protocol == manilav1alpha1.ProtocolNFS {
		parameters["nfs-share-client"] = *pr.Spec.NFSShareClient
	}

	sc := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: pr.Spec.StorageClassName,
		},
		Provisioner:   provisionerName,
		Parameters:    parameters,
		ReclaimPolicy: pr.Spec.ReclaimPolicy,
	}
	sc.SetLabels(selector)

	_, _, err := ApplyStorageClass(r.clientset.StorageV1(), sc)
	return err
}

// ApplyStorageClass merges objectmeta, tries to write everything else
func ApplyStorageClass(client storageclientv1.StorageClassesGetter, required *storagev1.StorageClass) (*storagev1.StorageClass, bool, error) {
	existing, err := client.StorageClasses().Get(required.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		actual, err := client.StorageClasses().Create(required)
		return actual, true, err
	}
	if err != nil {
		return nil, false, err
	}

	modified := resourcemerge.BoolPtr(false)
	resourcemerge.EnsureObjectMeta(modified, &existing.ObjectMeta, required.ObjectMeta)
	contentSame := equality.Semantic.DeepEqual(existing, required)
	if contentSame && !*modified {
		return existing, false, nil
	}

	// Provisioner, Parameters, ReclaimPolicy, and VolumeBindingMode are immutable
	recreate := resourcemerge.BoolPtr(false)
	resourcemerge.SetStringIfSet(recreate, &existing.Provisioner, required.Provisioner)
	resourcemerge.SetMapStringStringIfSet(recreate, &existing.Parameters, required.Parameters)
	if required.ReclaimPolicy != nil && !equality.Semantic.DeepEqual(existing.ReclaimPolicy, required.ReclaimPolicy) {
		existing.ReclaimPolicy = required.ReclaimPolicy
		*recreate = true
	}
	resourcemerge.SetStringSliceIfSet(modified, &existing.MountOptions, required.MountOptions)
	if required.AllowVolumeExpansion != nil && !equality.Semantic.DeepEqual(existing.AllowVolumeExpansion, required.AllowVolumeExpansion) {
		existing.AllowVolumeExpansion = required.AllowVolumeExpansion
	}
	if required.VolumeBindingMode != nil && !equality.Semantic.DeepEqual(existing.VolumeBindingMode, required.VolumeBindingMode) {
		existing.VolumeBindingMode = required.VolumeBindingMode
		*recreate = true
	}
	if required.AllowedTopologies != nil && !equality.Semantic.DeepEqual(existing.AllowedTopologies, required.AllowedTopologies) {
		existing.AllowedTopologies = required.AllowedTopologies
	}

	if *recreate {
		err := client.StorageClasses().Delete(existing.Name, nil)
		if err != nil && !apierrors.IsNotFound(err) {
			return nil, false, err
		}
		actual, err := client.StorageClasses().Create(existing)
		return actual, true, err
	}
	actual, err := client.StorageClasses().Update(existing)
	return actual, true, err
}

func (r *ReconcileManilaProvisioner) syncDeployment(pr *manilav1alpha1.ManilaProvisioner, previousAvailability *operatorv1alpha1.VersionAvailability, forceDeployment bool) (*appsv1.Deployment, error) {
	selector := labelsForProvisioner(pr)

	deployment := resourceread.ReadDeploymentV1OrDie(generated.MustAsset("assets/deployment.yaml"))

	deployment.SetName(pr.GetName())
	deployment.SetNamespace(pr.GetNamespace())
	deployment.SetLabels(selector)

	deployment.Spec.Replicas = &pr.Spec.Replicas
	deployment.Spec.Selector = &metav1.LabelSelector{MatchLabels: selector}

	template := &deployment.Spec.Template

	template.SetLabels(selector)

	template.Spec.Containers[0].Image = pr.Spec.ImagePullSpec

	if err := controllerutil.SetControllerReference(pr, deployment, r.scheme); err != nil {
		return nil, err
	}
	actualDeployment, _, err := resourceapply.ApplyDeployment(r.clientset.AppsV1(), deployment, resourcemerge.ExpectedDeploymentGeneration(deployment, previousAvailability), forceDeployment)
	if err != nil {
		return nil, err
	}

	return actualDeployment, nil
}

func (r *ReconcileManilaProvisioner) cleanup(pr *manilav1alpha1.ManilaProvisioner) error {
	err := r.cleanupStorageClass(pr)
	if err != nil {
		return err
	}
	err = r.cleanupRBAC(pr)
	if err != nil {
		return err
	}
	err = r.cleanupFinalizer(pr)
	if err != nil {
		return err
	}
	return nil
}

func (r *ReconcileManilaProvisioner) cleanupStorageClass(pr *manilav1alpha1.ManilaProvisioner) error {
	scList := &storagev1.StorageClassList{}
	opts := &client.ListOptions{
		LabelSelector: labels.Set(labelsForProvisioner(pr)).AsSelector(),
	}
	err := r.client.List(context.TODO(), opts, scList)
	if err != nil {
		return err
	}
	for _, sc := range scList.Items {
		err = r.client.Delete(context.TODO(), &sc)
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func (r *ReconcileManilaProvisioner) cleanupRBAC(pr *manilav1alpha1.ManilaProvisioner) error {
	crList := &rbacv1.ClusterRoleList{}
	opts := &client.ListOptions{
		LabelSelector: labels.Set(labelsForProvisioner(pr)).AsSelector(),
	}
	err := r.client.List(context.TODO(), opts, crList)
	if err != nil {
		return err
	}
	for _, cr := range crList.Items {
		err := r.client.Delete(context.TODO(), &cr)
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}
	crbList := &rbacv1.ClusterRoleBindingList{}
	err = r.client.List(context.TODO(), opts, crbList)
	if err != nil {
		return err
	}
	for _, crb := range crbList.Items {
		err := r.client.Delete(context.TODO(), &crb)
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func (r *ReconcileManilaProvisioner) cleanupFinalizer(pr *manilav1alpha1.ManilaProvisioner) error {
	finalizers := []string{}
	for _, f := range pr.Finalizers {
		if f == finalizerName {
			continue
		}
		finalizers = append(finalizers, f)
	}
	pr.Finalizers = finalizers
	err := r.client.Update(context.TODO(), pr)
	if err != nil {
		return err
	}
	return nil
}

const (
	OwnerLabelNamespace = "manila.storage.openshift.io/owner-namespace"
	OwnerLabelName      = "manila.storage.openshift.io/owner-name"
)

// labelsForProvisioner returns the labels for selecting the resources
// belonging to the given provisioner name.
func labelsForProvisioner(pr *manilav1alpha1.ManilaProvisioner) map[string]string {
	return map[string]string{
		OwnerLabelNamespace: pr.Namespace,
		OwnerLabelName:      pr.Name,
	}
}

// Identical to efs syncStatus
func (r *ReconcileManilaProvisioner) syncStatus(operatorConfig *manilav1alpha1.ManilaProvisioner, deployment *appsv1.Deployment, errors []error) error {
	versionAvailability := operatorv1alpha1.VersionAvailability{}
	versionAvailability = resourcemerge.ApplyDeploymentGenerationAvailability(versionAvailability, deployment, errors...)
	operatorConfig.Status.CurrentAvailability = &versionAvailability

	// given the VersionAvailability and the status.Version, we can compute availability
	availableCondition := operatorv1alpha1.OperatorCondition{
		Type:   operatorv1alpha1.OperatorStatusTypeAvailable,
		Status: operatorv1alpha1.ConditionUnknown,
	}
	if operatorConfig.Status.CurrentAvailability != nil && operatorConfig.Status.CurrentAvailability.ReadyReplicas > 0 {
		availableCondition.Status = operatorv1alpha1.ConditionTrue
	} else {
		availableCondition.Status = operatorv1alpha1.ConditionFalse
	}
	v1alpha1helpers.SetOperatorCondition(&operatorConfig.Status.Conditions, availableCondition)

	syncSuccessfulCondition := operatorv1alpha1.OperatorCondition{
		Type:   operatorv1alpha1.OperatorStatusTypeSyncSuccessful,
		Status: operatorv1alpha1.ConditionTrue,
	}
	if operatorConfig.Status.CurrentAvailability != nil && len(operatorConfig.Status.CurrentAvailability.Errors) > 0 {
		syncSuccessfulCondition.Status = operatorv1alpha1.ConditionFalse
		syncSuccessfulCondition.Message = strings.Join(operatorConfig.Status.CurrentAvailability.Errors, "\n")
	}
	if operatorConfig.Status.TargetAvailability != nil && len(operatorConfig.Status.TargetAvailability.Errors) > 0 {
		syncSuccessfulCondition.Status = operatorv1alpha1.ConditionFalse
		if len(syncSuccessfulCondition.Message) == 0 {
			syncSuccessfulCondition.Message = strings.Join(operatorConfig.Status.TargetAvailability.Errors, "\n")
		} else {
			syncSuccessfulCondition.Message = availableCondition.Message + "\n" + strings.Join(operatorConfig.Status.TargetAvailability.Errors, "\n")
		}
	}
	v1alpha1helpers.SetOperatorCondition(&operatorConfig.Status.Conditions, syncSuccessfulCondition)
	if syncSuccessfulCondition.Status == operatorv1alpha1.ConditionTrue {
		operatorConfig.Status.ObservedGeneration = operatorConfig.ObjectMeta.Generation
	}

	return r.client.Status().Update(context.TODO(), operatorConfig)
}
