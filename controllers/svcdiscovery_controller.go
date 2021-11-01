// Copyright (c) 2019 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	goctx "context"
	"net"
	"net/url"
	"reflect"
	"sigs.k8s.io/cluster-api-provider-vsphere/pkg/context"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	bootstrapapi "k8s.io/cluster-bootstrap/token/api"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	capv1 "sigs.k8s.io/cluster-api-provider-vsphere/apis/vmware/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	//tkgv1 "gitlab.eng.vmware.com/core-build/guest-cluster-controller/apis/run.tanzu/v1alpha2"
	// "gitlab.eng.vmware.com/core-build/guest-cluster-controller/controllers/common"
	// "gitlab.eng.vmware.com/core-build/guest-cluster-controller/pkg/builder"
	// "gitlab.eng.vmware.com/core-build/guest-cluster-controller/pkg/context"
)

const (
	controllerName = "svcdiscovery-controller"
	SupervisorLoadBalancerSvcNamespace = "kube-system"
	SupervisorLoadBalancerSvcName      = "kube-apiserver-lb-svc"
	SupervisorAPIServerPort            = 6443

	SupervisorHeadlessSvcNamespace = "default"
	SupervisorHeadlessSvcName      = "supervisor"
	SupervisorHeadlessServiceSetupFailedReason = "SupervisorHeadlessServiceSetupFailed"
)

// AddToManager adds this package's controller to the provided manager.
func AddToManager(ctx *context.ControllerManagerContext, mgr manager.Manager) error {
	controller, err := builder.NewController(ctx, mgr, controllerName, reconciler{})
	if err != nil {
		return err
	}

	// Watch for updates to the LB-type Service for the supervisor apiserver VIP.
	if err := controller.Watch(
		&source.Kind{Type: &corev1.Service{}},
		&handler.EnqueueRequestsFromMapFunc{ToRequests: svcMapper{ctx}}); err != nil {
		return err
	}

	cache, err := cache.New(mgr.GetConfig(),
		cache.Options{
			Scheme: mgr.GetScheme(),
			Mapper: mgr.GetRESTMapper(),
			Resync: ctx.SyncPeriod,
			// configMapMapper is hardwired to the public namespace
			Namespace: metav1.NamespacePublic,
		},
	)
	if err != nil {
		return errors.Wrapf(err, "failed to create configmap cache")
	}

	if err := mgr.Add(cache); err != nil {
		return errors.Wrapf(err, "failed to start configmap cache")
	}

	src := source.NewKindWithCache(&corev1.ConfigMap{}, cache)

	// Watch for updates to the cluster-info configmap for the supervisor apiserver FIP.
	err = controller.Watch(
		src,
		&handler.EnqueueRequestsFromMapFunc{ToRequests: configMapMapper{ctx}})
	if err != nil {
		return errors.Wrapf(err, "failed to watch configmaps")
	}

	return nil
}

type svcMapper struct {
	ctx *context.ControllerManagerContext
}

func (d svcMapper) Map(o handler.MapObject) []reconcile.Request {
	// We are only interested in the LB-type Service for the supervisor apiserver.
	if o.Meta.GetNamespace() != SupervisorLoadBalancerSvcNamespace || o.Meta.GetName() != SupervisorLoadBalancerSvcName {
		return nil
	}
	return allClustersRequests(d.ctx)
}

type configMapMapper struct {
	ctx *context.ControllerManagerContext
}

func (d configMapMapper) Map(o handler.MapObject) []reconcile.Request {
	// We are only interested in the cluster-info configmap for the supervisor apiserver.
	if o.Meta.GetNamespace() != metav1.NamespacePublic || o.Meta.GetName() != bootstrapapi.ConfigMapClusterInfo {
		return nil
	}
	return allClustersRequests(d.ctx)
}

func allClustersRequests(ctx *context.ControllerManagerContext) []reconcile.Request {
	clustersList := &capv1.VSphereClusterList{}
	if err := ctx.Client.List(ctx, clustersList, &client.ListOptions{}); err != nil {
		return nil
	}

	requests := make([]reconcile.Request, 0, len(clustersList.Items))
	for _, cluster := range clustersList.Items {
		key := client.ObjectKey{Namespace: cluster.Namespace, Name: cluster.Name}
		requests = append(requests, reconcile.Request{NamespacedName: key})
	}
	return requests
}

// NewReconciler returns the package's GuestClusterReconciler.
func NewReconciler() builder.Reconciler {
	return reconciler{}
}

type reconciler struct{}

func (r reconciler) RequiredComponents() []builder.ComponentType {
	return []builder.ComponentType{}
}

// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=services/status,verbs=get
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=configmaps/status,verbs=get

func (r reconciler) ReconcileNormal(ctx *context.GuestClusterContext) (reconcile.Result, error) {
	ctx.Logger.V(4).Info("Reconciling Service Discovery", "cluster", ctx.ClusterObjectKey)

	if err := r.reconcileSupervisorHeadlessService(ctx); err != nil {
		conditions.MarkFalse(ctx.Cluster, capv1.ServiceDiscoveryReadyCondition, capv1.SupervisorHeadlessServiceSetupFailedReason,
			clusterv1.ConditionSeverityWarning, err.Error())
		return reconcile.Result{}, errors.Wrapf(err, "failed to configure supervisor headless service for %s", ctx)
	}

	return reconcile.Result{}, nil
}

// Setup a local k8s service in the target cluster that proxies to the Supervisor Cluster API Server. The add-ons are
// dependent on this local service to connect to the Supervisor Cluster.
func (r reconciler) reconcileSupervisorHeadlessService(ctx *context.GuestClusterContext) error {
	// Create the headless service to the supervisor api server on the target cluster.
	supervisorPort := SupervisorAPIServerPort
	svc := NewSupervisorHeadlessService(common.SupervisorHeadlessSvcPort, supervisorPort)
	if err := ctx.GuestClient.Create(ctx, svc); err != nil && !apierrors.IsAlreadyExists(err) {
		return errors.Wrapf(err, "cannot create k8s service %s/%s in ", svc.Namespace, svc.Name)
	}

	supervisorHost, err := GetSupervisorAPIServerAddress(ctx.ClusterContext)
	if err != nil {
		// Note: We have watches on the LB Svc (VIP) & the cluster-info configmap (FIP). There is no need to return an error to keep
		// re-trying.
		conditions.MarkFalse(ctx.Cluster, capv1.ServiceDiscoveryReadyCondition, capv1.SupervisorHeadlessServiceSetupFailedReason,
			clusterv1.ConditionSeverityWarning, err.Error())
		return nil
	}

	ctx.Logger.Info("Discovered supervisor apiserver address", "host", supervisorHost, "port", supervisorPort)
	// CreateOrUpdate the newEndpoints with the discovered supervisor api server address
	newEndpoints := NewSupervisorHeadlessServiceEndpoints(supervisorHost, supervisorPort)
	endpointsKey := types.NamespacedName{Name: SupervisorHeadlessSvcName, Namespace: SupervisorHeadlessSvcNamespace}
	if createErr := ctx.GuestClient.Create(ctx, newEndpoints); createErr != nil {

		if apierrors.IsAlreadyExists(createErr) {
			var endpoints corev1.Endpoints
			if getErr := ctx.GuestClient.Get(ctx, endpointsKey, &endpoints); getErr != nil {
				return errors.Wrapf(getErr, "cannot get k8s service endpoints %s", endpointsKey)
			}
			// Update only if modified
			if !reflect.DeepEqual(endpoints.Subsets, newEndpoints.Subsets) {
				endpoints.Subsets = newEndpoints.Subsets
				// Update the newEndpoints if it already exists
				if updateErr := ctx.GuestClient.Update(ctx, &endpoints); updateErr != nil {
					return errors.Wrapf(updateErr, "cannot update k8s service endpoints %s", endpointsKey)
				}
			}
		} else {
			return errors.Wrapf(createErr, "cannot create k8s service endpoints %s", endpointsKey)
		}
	}

	conditions.MarkTrue(ctx.Cluster, capv1.ServiceDiscoveryReadyCondition)
	return nil
}

func GetSupervisorAPIServerAddress(ctx *context.ClusterContext) (string, error) {
	// Discover the supervisor api server address
	// 1. Check if a k8s service "kube-system/kube-apiserver-lb-svc" is available, if so, fetch the loadbalancer IP.
	// 2. If not, get the Supervisor Cluster Management Network Floating IP (FIP) from the cluster-info configmap. This is
	// to support non-NSX-T development usecases only. If we are unable to find the cluster-info configmap for some reason,
	// we log the error.
	supervisorHost, err := GetSupervisorAPIServerVIP(ctx.Client)
	if err != nil {
		ctx.Logger.Info("Unable to discover supervisor apiserver virtual ip, fallback to floating ip", "reason", err.Error())
		supervisorHost, err = GetSupervisorAPIServerFIP(ctx.Client)
		if err != nil {
			ctx.Logger.Error(err, "Unable to discover supervisor apiserver address")
			return "", errors.Wrapf(err, "Unable to discover supervisor apiserver address")
		}
	}
	return supervisorHost, nil
}

func NewSupervisorHeadlessService(port, targetPort int) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      SupervisorHeadlessSvcName,
			Namespace: SupervisorHeadlessSvcNamespace,
		},
		Spec: corev1.ServiceSpec{
			// Note: This will be a headless service with no selectors. The endpoints will be manually created.
			ClusterIP: corev1.ClusterIPNone,
			Ports: []corev1.ServicePort{
				{
					Port:       int32(port),
					TargetPort: intstr.FromInt(targetPort),
				},
			},
		},
	}
}

func NewSupervisorHeadlessServiceEndpoints(targetHost string, targetPort int) *corev1.Endpoints {
	var endpointAddr corev1.EndpointAddress
	if ip := net.ParseIP(targetHost); ip != nil {
		endpointAddr.IP = ip.String()
	} else {
		endpointAddr.Hostname = targetHost
	}
	return &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      SupervisorHeadlessSvcName,
			Namespace: SupervisorHeadlessSvcNamespace,
		},
		Subsets: []corev1.EndpointSubset{
			{
				Addresses: []corev1.EndpointAddress{
					endpointAddr,
				},
				Ports: []corev1.EndpointPort{
					{
						Port: int32(targetPort),
					},
				},
			},
		},
	}
}

func GetSupervisorAPIServerVIP(client client.Client) (string, error) {
	svc := &corev1.Service{}
	svcKey := types.NamespacedName{Name: SupervisorLoadBalancerSvcName, Namespace: SupervisorLoadBalancerSvcNamespace}
	if err := client.Get(goctx.Background(), svcKey, svc); err != nil {
		return "", errors.Wrapf(err, "unable to get supervisor loadbalancer svc %s", svcKey)
	}
	if len(svc.Status.LoadBalancer.Ingress) > 0 {
		ingress := svc.Status.LoadBalancer.Ingress[0]
		if ipAddr := ingress.IP; ipAddr != "" {
			return ipAddr, nil
		}
		return ingress.Hostname, nil
	}
	return "", errors.Errorf("no VIP found in the supervisor loadbalancer svc %s", svcKey)
}

func GetSupervisorAPIServerFIP(client client.Client) (string, error) {
	urlString, err := getSupervisorAPIServerURLWithFIP(client)
	if err != nil {
		return "", errors.Wrap(err, "unable to get supervisor url")
	}
	urlVal, err := url.Parse(urlString)
	if err != nil {
		return "", errors.Wrapf(err, "unable to parse supervisor url from %s", urlString)
	}
	host := urlVal.Hostname()
	if host == "" {
		return "", errors.Errorf("unable to get supervisor host from url %s", urlVal)
	}
	return host, nil
}

func getSupervisorAPIServerURLWithFIP(client client.Client) (string, error) {
	cm := &corev1.ConfigMap{}
	cmKey := types.NamespacedName{Name: bootstrapapi.ConfigMapClusterInfo, Namespace: metav1.NamespacePublic}
	if err := client.Get(goctx.Background(), cmKey, cm); err != nil {
		return "", err
	}
	kubeconfig, err := tryParseClusterInfoFromConfigMap(cm)
	if err != nil {
		return "", err
	}
	clusterConfig := getClusterFromKubeConfig(kubeconfig)
	if clusterConfig != nil {
		return clusterConfig.Server, nil
	}
	return "", errors.Errorf("unable to get cluster from kubeconfig in ConfigMap %s/%s", cm.Namespace, cm.Name)

}

// tryParseClusterInfoFromConfigMap tries to parse a kubeconfig file from a ConfigMap key
func tryParseClusterInfoFromConfigMap(cm *corev1.ConfigMap) (*clientcmdapi.Config, error) {
	kubeConfigString, ok := cm.Data[bootstrapapi.KubeConfigKey]
	if !ok || len(kubeConfigString) == 0 {
		return nil, errors.Errorf("no %s key in ConfigMap %s/%s", bootstrapapi.KubeConfigKey, cm.Namespace, cm.Name)
	}
	parsedKubeConfig, err := clientcmd.Load([]byte(kubeConfigString))
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't parse the kubeconfig file in the ConfigMap %s/%s", cm.Namespace, cm.Name)
	}
	return parsedKubeConfig, nil
}

// GetClusterFromKubeConfig returns the default Cluster of the specified KubeConfig
func getClusterFromKubeConfig(config *clientcmdapi.Config) *clientcmdapi.Cluster {
	// If there is an unnamed cluster object, use it
	if config.Clusters[""] != nil {
		return config.Clusters[""]
	}
	if config.Contexts[config.CurrentContext] != nil {
		return config.Clusters[config.Contexts[config.CurrentContext].Cluster]
	}
	return nil
}
