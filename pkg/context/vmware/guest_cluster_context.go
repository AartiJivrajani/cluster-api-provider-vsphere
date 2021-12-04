package vmware

import (
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GuestClusterContext is the context used for GuestClusterControllers.
type GuestClusterContext struct {
	*ClusterContext

	// GuestClient can be used to access the guest cluster.
	GuestClient client.Client
}

// String returns ClusterGroupVersionKind ClusterNamespace/ClusterName.
func (c *GuestClusterContext) String() string {
	return fmt.Sprintf("%s %s/%s", c.Cluster.GroupVersionKind(), c.Cluster.Namespace, c.Cluster.Name)
}
