package builder

import (
	"sigs.k8s.io/cluster-api-provider-vsphere/pkg/context"
	"sigs.k8s.io/cluster-api-provider-vsphere/pkg/services"
)

type ClusterReconciler struct {
	*context.ControllerContext
	NetworkProvider       services.NetworkProvider
	ControlPlaneService   services.ControlPlaneEndpointService
	ResourcePolicyService services.ResourcePolicyService
}

type ServiceAccountReconciler struct {
	*context.ControllerContext
}
