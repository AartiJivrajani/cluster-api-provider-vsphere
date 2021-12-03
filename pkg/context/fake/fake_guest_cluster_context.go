package fake

import (
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/cluster-api-provider-vsphere/pkg/context/vmware"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	//apiregv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
)

// NewGuestClusterContext returns a fake GuestClusterContext for unit testing
// guest cluster controllers with a fake client.
func NewGuestClusterContext(ctx *vmware.ClusterContext, prototypeCluster bool, gcInitObjects ...runtime.Object) *vmware.GuestClusterContext {

	if prototypeCluster {
		cluster := newCluster(ctx.VSphereCluster)

		if err := ctx.Client.Create(ctx, cluster); err != nil {
			panic(err)
		}
	}

	// Add prototype addons that kubeadm would have generated for us
	//gcInitObjects = append(gcInitObjects, GetPrototypeKubeadmConfigMap())
	//gcInitObjects = append(gcInitObjects, GetPrototypeCoreDNSDeployment())
	//gcInitObjects = append(gcInitObjects, GetPrototypeKubeProxyDaemonSet())

	return &vmware.GuestClusterContext{
		ClusterContext: ctx,
		GuestClient:    NewFakeGuestClusterClient(gcInitObjects...),
	}
}

func NewFakeGuestClusterClient(initObjects ...runtime.Object) client.Client {
	scheme := scheme.Scheme
	_ = apiextv1.AddToScheme(scheme)
	//_ = apiregv1.AddToScheme(scheme)

	return fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(initObjects...).Build()
}
