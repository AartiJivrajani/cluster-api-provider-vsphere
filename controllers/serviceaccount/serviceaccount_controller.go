// Copyright (c) 2019 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package serviceaccount

import (
	"fmt"
	"os"
	"strconv"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	tkgv1 "gitlab.eng.vmware.com/core-build/guest-cluster-controller/apis/run.tanzu/v1alpha2"
	"gitlab.eng.vmware.com/core-build/guest-cluster-controller/pkg/builder"
	//"gitlab.eng.vmware.com/core-build/guest-cluster-controller/pkg/context"
	"sigs.k8s.io/cluster-api-provider-vsphere/pkg/context"
)

const (
	controllerName             = "provider-serviceaccount-controller"
	kindProviderServiceAccount = "ProviderServiceAccount"
	systemServiceAccountPrefix = "system.serviceaccount"
)

// AddToManager adds this package's controller to the provided manager.
func AddToManager(ctx *context.ControllerManagerContext, mgr manager.Manager) error {
	c, err := builder.NewController(ctx, mgr, controllerName, reconciler{})
	if err != nil {
		return err
	}

	// Watch a ProviderServiceAccount
	if err := c.Watch(&source.Kind{Type: &tkgv1.ProviderServiceAccount{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &tkgv1.TanzuKubernetesCluster{},
	}); err != nil {
		return err
	}

	// Watch a ServiceAccount created by this controller
	if err := c.Watch(&source.Kind{Type: &corev1.ServiceAccount{}},
		&handler.EnqueueRequestsFromMapFunc{ToRequests: requestMapper{ctx}}); err != nil {
		return err
	}

	// Watch a Role created by this controller
	if err := c.Watch(&source.Kind{Type: &rbacv1.Role{}},
		&handler.EnqueueRequestsFromMapFunc{ToRequests: requestMapper{ctx}}); err != nil {
		return err
	}

	// Watch a RoleBinding created by this controller
	if err := c.Watch(&source.Kind{Type: &rbacv1.RoleBinding{}},
		&handler.EnqueueRequestsFromMapFunc{ToRequests: requestMapper{ctx}}); err != nil {
		return err
	}

	return nil
}

type requestMapper struct {
	ctx *context.ControllerManagerContext
}

func (d requestMapper) Map(o handler.MapObject) []reconcile.Request {
	// If the watched object [role|rolebinding|serviceaccount] is owned by this providerserviceaccount controller, then
	// lookup the tanzu kubernetes cluster that owns the providerserviceaccount object that needs to be queued. We do this because
	// this controller is effectively a tanzukubernetescluster controller which reconciles it's dependent providerserviceaccounts.
	ownerRef := metav1.GetControllerOf(o.Meta)
	if ownerRef != nil && ownerRef.Kind == kindProviderServiceAccount {
		key := types.NamespacedName{Namespace: o.Meta.GetNamespace(), Name: ownerRef.Name}
		return getTanzuKubernetesCluster(d.ctx, key)
	}
	return nil
}

func getTanzuKubernetesCluster(ctx *context.ControllerManagerContext, pSvcAccountKey types.NamespacedName) []reconcile.Request {
	pSvcAccount := &tkgv1.ProviderServiceAccount{}
	if err := ctx.Client.Get(ctx, pSvcAccountKey, pSvcAccount); err != nil {
		return nil
	}

	tkcRef := pSvcAccount.Spec.Ref
	if tkcRef == nil || tkcRef.Name == "" {
		return nil
	}
	key := client.ObjectKey{Namespace: pSvcAccount.Namespace, Name: tkcRef.Name}
	return []reconcile.Request{{NamespacedName: key}}
}

// NewReconciler returns the package's GuestClusterReconciler.
func NewReconciler() builder.Reconciler {
	return reconciler{}
}

type reconciler struct{}

func (r reconciler) RequiredComponents() []builder.ComponentType {
	return []builder.ComponentType{}
}

func (r reconciler) ReconcileDelete(ctx *context.ClusterContext) (reconcile.Result, error) {
	ctx.Logger.V(4).Info("Reconciling deleting Provider ServiceAccounts", "cluster", ctx.ClusterObjectKey)

	pSvcAccounts, err := getProviderServiceAccounts(ctx)
	if err != nil {
		ctx.Logger.Error(err, "Error fetching provider serviceaccounts")
		return reconcile.Result{}, err
	}

	for _, pSvcAccount := range pSvcAccounts {
		// Delete entries for configmap with serviceaccount
		if err := r.deleteServiceAccountConfigMap(ctx, pSvcAccount); err != nil {
			return reconcile.Result{}, errors.Wrapf(err, "unable to delete configmap entry for provider serviceaccount %s", pSvcAccount.Name)
		}
	}

	return reconcile.Result{}, nil
}

// +kubebuilder:rbac:groups=run.tanzu.vmware.com,resources=providerserviceaccounts,verbs=get;list;watch;
// +kubebuilder:rbac:groups=run.tanzu.vmware.com,resources=providerserviceaccounts/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles;rolebindings,verbs=get;list;watch;create;update;patch;delete

func (r reconciler) ReconcileNormal(ctx *context.GuestClusterContext) (_ reconcile.Result, reterr error) {
	ctx.Logger.V(4).Info("Reconciling Provider ServiceAccount", "cluster", ctx.ClusterObjectKey)

	defer func() {
		if reterr != nil {
			conditions.MarkFalse(ctx.Cluster, tkgv1.ProviderServiceAccountsReadyCondition, tkgv1.ProviderServiceAccountsReconciliationFailedReason,
				clusterv1.ConditionSeverityWarning, reterr.Error())
		} else {
			conditions.MarkTrue(ctx.Cluster, tkgv1.ProviderServiceAccountsReadyCondition)
		}
	}()

	pSvcAccounts, err := getProviderServiceAccounts(ctx.ClusterContext)
	if err != nil {
		ctx.Logger.Error(err, "Error fetching provider serviceaccounts")
		return reconcile.Result{}, err
	}
	err = r.ensureProviderServiceAccounts(ctx, pSvcAccounts)
	if err != nil {
		ctx.Logger.Error(err, "Error ensuring provider serviceaccounts")
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

// Ensure service accounts from provider spec is created
func (r reconciler) ensureProviderServiceAccounts(ctx *context.GuestClusterContext, pSvcAccounts []tkgv1.ProviderServiceAccount) error {

	for _, pSvcAccount := range pSvcAccounts {
		// 1. Create service accounts by the name specified in Provider Spec
		if err := r.ensureServiceAccount(ctx, pSvcAccount); err != nil {
			return errors.Wrapf(err, "unable to create provider serviceaccount %s", pSvcAccount.Name)
		}

		// 2. Update configmap with serviceaccount
		if err := r.ensureServiceAccountConfigMap(ctx.ClusterContext, pSvcAccount); err != nil {
			return errors.Wrapf(err, "unable to sync configmap for provider serviceaccount %s", pSvcAccount.Name)
		}

		// 3. Create the associated role for the service account
		if err := r.ensureRole(ctx, pSvcAccount); err != nil {
			return errors.Wrapf(err, "unable to create role for provider serviceaccount %s", pSvcAccount.Name)
		}

		// 4. Create the associated roleBinding for the service account
		if err := r.ensureRoleBinding(ctx, pSvcAccount); err != nil {
			return errors.Wrapf(err, "unable to create rolebinding for provider serviceaccount %s", pSvcAccount.Name)
		}

		// 5. Sync the service account with the target
		if err := r.syncServiceAccountSecret(ctx, pSvcAccount); err != nil {
			return errors.Wrapf(err, "unable to sync secret for provider serviceaccount %s", pSvcAccount.Name)
		}
	}
	return nil
}

func (r reconciler) ensureServiceAccount(ctx *context.GuestClusterContext, pSvcAccount tkgv1.ProviderServiceAccount) error {
	svcAccount := corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getServiceAccountName(pSvcAccount),
			Namespace: pSvcAccount.Namespace,
		},
	}
	logger := ctx.Logger.WithValues("providerserviceaccount", pSvcAccount.Name, "serviceaccount", svcAccount.Name)
	err := controllerutil.SetControllerReference(&pSvcAccount, &svcAccount, ctx.Scheme)
	if err != nil {
		return err
	}
	logger.V(4).Info("Creating service account")
	err = ctx.Client.Create(ctx, &svcAccount)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		// Note: We skip updating the service account because the token controller updates the service account with a
		// secret and we don't want to overwrite it with an empty secret.
		return err
	}
	return nil
}

func (r reconciler) ensureRole(ctx *context.GuestClusterContext, pSvcAccount tkgv1.ProviderServiceAccount) error {
	role := rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getRoleName(pSvcAccount),
			Namespace: pSvcAccount.Namespace,
		},
	}
	logger := ctx.Logger.WithValues("providerserviceaccount", pSvcAccount.Name, "role", role.Name)
	logger.V(4).Info("Creating or updating role")
	_, err := controllerutil.CreateOrUpdate(ctx, ctx.Client, &role, func() error {
		if err := controllerutil.SetControllerReference(&pSvcAccount, &role, ctx.Scheme); err != nil {
			return err
		}
		// TODO(GCM-1234): Limit the rules that can be specified here
		role.Rules = pSvcAccount.Spec.Rules
		return nil
	})
	return err
}

func (r reconciler) ensureRoleBinding(ctx *context.GuestClusterContext, pSvcAccount tkgv1.ProviderServiceAccount) error {
	roleName := getRoleName(pSvcAccount)
	svcAccountName := getServiceAccountName(pSvcAccount)
	roleBinding := rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getRoleBindingName(pSvcAccount),
			Namespace: pSvcAccount.Namespace,
		},
	}
	logger := ctx.Logger.WithValues("providerserviceaccount", pSvcAccount.Name, "rolebinding", roleBinding.Name)
	logger.V(4).Info("Creating or updating rolebinding")
	_, err := controllerutil.CreateOrUpdate(ctx, ctx.Client, &roleBinding, func() error {
		if err := controllerutil.SetControllerReference(&pSvcAccount, &roleBinding, ctx.Scheme); err != nil {
			return err
		}
		roleBinding.RoleRef = rbacv1.RoleRef{
			Name:     roleName,
			Kind:     "Role",
			APIGroup: rbacv1.GroupName,
		}
		roleBinding.Subjects = []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				APIGroup:  "",
				Name:      svcAccountName,
				Namespace: pSvcAccount.Namespace},
		}
		return nil
	})
	return err
}

func (r reconciler) syncServiceAccountSecret(ctx *context.GuestClusterContext, pSvcAccount tkgv1.ProviderServiceAccount) error {
	logger := ctx.Logger.WithValues("providerserviceaccount", pSvcAccount.Name)
	logger.V(4).Info("Attempting to sync secret for provider service account")
	var svcAccount corev1.ServiceAccount
	err := ctx.Client.Get(ctx, types.NamespacedName{Name: getServiceAccountName(pSvcAccount), Namespace: pSvcAccount.Namespace}, &svcAccount)
	if err != nil {
		return err
	}
	// Check if token secret exists
	if len(svcAccount.Secrets) == 0 {
		// Note: We don't have to requeue here because we have a watch on the service account and the cluster should be reconciled
		// when a secret is added to the service account by the token controller.
		logger.Info("Skipping sync secret for provider service account: serviceaccount has no secrets", "serviceaccount", svcAccount.Name)
		return nil
	}

	// Choose the default secret
	secretRef := svcAccount.Secrets[0]
	logger.V(4).Info("Fetching secret for provider service account", "secret", secretRef.Name)
	var sourceSecret corev1.Secret
	err = ctx.Client.Get(ctx, types.NamespacedName{Name: secretRef.Name, Namespace: svcAccount.Namespace}, &sourceSecret)
	if err != nil {
		return err
	}

	// Create the target namespace if it is not existing
	targetNamespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: pSvcAccount.Spec.TargetNamespace,
		},
	}

	if err = ctx.GuestClient.Get(ctx, client.ObjectKey{Name: pSvcAccount.Spec.TargetNamespace}, targetNamespace); err != nil {
		if apierrors.IsNotFound(err) {
			err = ctx.GuestClient.Create(ctx, targetNamespace)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	// Note: We ignore the Secret type & annotations because they are created by the token controller and will not be valid in the target cluster.
	targetSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pSvcAccount.Spec.TargetSecretName,
			Namespace: pSvcAccount.Spec.TargetNamespace,
		},
	}
	logger.V(4).Info("Creating or updating secret in cluster", "namespace", targetSecret.Namespace, "name", targetSecret.Name)
	_, err = controllerutil.CreateOrUpdate(ctx, ctx.GuestClient, targetSecret, func() error {
		targetSecret.Data = sourceSecret.Data
		return nil
	})
	return err
}

func (r reconciler) getConfigMapAndBuffer(ctx *context.ClusterContext) (*corev1.ConfigMap, *corev1.ConfigMap, error) {
	configMap := &corev1.ConfigMap{}

	if err := ctx.Client.Get(ctx, GetCMNamespaceName(), configMap); err != nil {
		return nil, nil, err
	}

	configMapBuffer := &corev1.ConfigMap{}
	configMapBuffer.Name = configMap.Name
	configMapBuffer.Namespace = configMap.Namespace
	return configMapBuffer, configMap, nil
}

func (r reconciler) deleteServiceAccountConfigMap(ctx *context.ClusterContext, svcAccount tkgv1.ProviderServiceAccount) error {
	logger := ctx.Logger.WithValues("providerserviceaccount", svcAccount.Name)

	svcAccountName := getSystemServiceAccountFullName(svcAccount)
	configMapBuffer, configMap, err := r.getConfigMapAndBuffer(ctx)
	if err != nil {
		return err
	}
	if valid, exist := configMap.Data[svcAccountName]; !exist || valid != strconv.FormatBool(true) {
		// Service account name is not in the config map
		return nil
	}
	logger.Info("Deleting config map entry for provider service account")
	_, err = controllerutil.CreateOrUpdate(ctx, ctx.Client, configMapBuffer, func() error {
		configMapBuffer.Data = configMap.Data
		delete(configMapBuffer.Data, svcAccountName)
		return nil
	})
	return err
}

func (r reconciler) ensureServiceAccountConfigMap(ctx *context.ClusterContext, svcAccount tkgv1.ProviderServiceAccount) error {
	logger := ctx.Logger.WithValues("providerserviceaccount", svcAccount.Name)

	svcAccountName := getSystemServiceAccountFullName(svcAccount)
	configMapBuffer, configMap, err := r.getConfigMapAndBuffer(ctx)
	if err != nil {
		return err
	}
	if valid, exist := configMap.Data[svcAccountName]; exist && valid == strconv.FormatBool(true) {
		// Service account name is already in the config map
		return nil
	}
	logger.Info("Updating config map for provider service account")
	_, err = controllerutil.CreateOrUpdate(ctx, ctx.Client, configMapBuffer, func() error {
		configMapBuffer.Data = configMap.Data
		configMapBuffer.Data[svcAccountName] = "true"
		return nil
	})
	return err
}

func getProviderServiceAccounts(ctx *context.ClusterContext) ([]tkgv1.ProviderServiceAccount, error) {
	var pSvcAccounts []tkgv1.ProviderServiceAccount

	pSvcAccountList := tkgv1.ProviderServiceAccountList{}
	if err := ctx.Client.List(ctx, &pSvcAccountList, client.InNamespace(ctx.Cluster.Namespace)); err != nil {
		return nil, err
	}

	for _, pSvcAccount := range pSvcAccountList.Items {
		// TODO (GCM-1233): For now, skip provider service accounts that are in the process of being deleted.
		// step to clean up the target secret in the guest cluster. Note: when the provider service account is deleted
		// all the associated [serviceaccount|role|rolebindings] are deleted. Hence, the bearer token in the target
		// secret will be rendered invalid. Still, it's a good practice to delete the secret that we created.
		if pSvcAccount.DeletionTimestamp != nil {
			continue
		}
		ref := pSvcAccount.Spec.Ref
		if ref != nil && ref.Name == ctx.Cluster.Name {
			pSvcAccounts = append(pSvcAccounts, pSvcAccount)
		}
	}
	return pSvcAccounts, nil
}

func getRoleName(pSvcAccount tkgv1.ProviderServiceAccount) string {
	return pSvcAccount.Name
}

func getRoleBindingName(pSvcAccount tkgv1.ProviderServiceAccount) string {
	return pSvcAccount.Name
}

func getServiceAccountName(pSvcAccount tkgv1.ProviderServiceAccount) string {
	return pSvcAccount.Name
}

func getSystemServiceAccountFullName(pSvcAccount tkgv1.ProviderServiceAccount) string {
	return fmt.Sprintf("%s.%s.%s", systemServiceAccountPrefix, getServiceAccountNamespace(pSvcAccount), getServiceAccountName(pSvcAccount))
}

func getServiceAccountNamespace(pSvcAccount tkgv1.ProviderServiceAccount) string {
	return pSvcAccount.Namespace
}

// GetCMNamespaceName gets capi valid modifier configmap metadata
func GetCMNamespaceName() types.NamespacedName {
	return types.NamespacedName{
		Namespace: os.Getenv("SERVICE_ACCOUNTS_CM_NAMESPACE"),
		Name:      os.Getenv("SERVICE_ACCOUNTS_CM_NAME"),
	}
}
