package fake

import (
	infrav1 "sigs.k8s.io/cluster-api-provider-vsphere/apis/vmware/v1beta1"
	"sigs.k8s.io/cluster-api-provider-vsphere/pkg/context"
	"sigs.k8s.io/cluster-api-provider-vsphere/pkg/context/vmware"
)

// NewVmwareClusterContext returns a fake ClusterContext for unit testing
// reconcilers with a fake client.
func NewVmwareClusterContext(ctx *context.ControllerContext, vsphereCluster *infrav1.VSphereCluster) *vmware.ClusterContext {

	// Create the cluster resources.
	cluster := newClusterV1()
	if vsphereCluster == nil {
		v := NewVSphereCluster()
		vsphereCluster = &v
	}

	// Add the cluster resources to the fake cluster client.
	if err := ctx.Client.Create(ctx, &cluster); err != nil {
		panic(err)
	}
	if err := ctx.Client.Create(ctx, vsphereCluster); err != nil {
		panic(err)
	}

	return &vmware.ClusterContext{
		ControllerContext: ctx,
		VSphereCluster: vsphereCluster,
		Logger:         ctx.Logger.WithName(vsphereCluster.Name),
	}
}
