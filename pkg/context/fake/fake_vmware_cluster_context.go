package fake

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"

	infrav1 "sigs.k8s.io/cluster-api-provider-vsphere/apis/vmware/v1beta1"
	"sigs.k8s.io/cluster-api-provider-vsphere/pkg/context"
	"sigs.k8s.io/cluster-api-provider-vsphere/pkg/context/vmware"
)

// NewVmwareClusterContext returns a fake ClusterContext for unit testing
// reconcilers with a fake client.
func NewVmwareClusterContext(ctx *context.ControllerContext) *vmware.ClusterContext {

	// Create the cluster resources.
	cluster := newClusterV1()
	vsphereCluster := newVmwareVSphereCluster(cluster)

	// Add the cluster resources to the fake cluster client.
	if err := ctx.Client.Create(ctx, &cluster); err != nil {
		panic(err)
	}
	if err := ctx.Client.Create(ctx, &vsphereCluster); err != nil {
		panic(err)
	}

	return &vmware.ClusterContext{
		ControllerContext: ctx,
		Cluster:           &cluster,
		VSphereCluster:    &vsphereCluster,
		Logger:            ctx.Logger.WithName(vsphereCluster.Name),
	}
}

func newVmwareVSphereCluster(owner clusterv1.Cluster) infrav1.VSphereCluster {
	return infrav1.VSphereCluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: owner.Namespace,
			Name:      owner.Name,
			UID:       VSphereClusterUUID,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         owner.APIVersion,
					Kind:               owner.Kind,
					Name:               owner.Name,
					UID:                owner.UID,
					BlockOwnerDeletion: &boolTrue,
					Controller:         &boolTrue,
				},
			},
		},
		Spec: infrav1.VSphereClusterSpec{},
	}
}
