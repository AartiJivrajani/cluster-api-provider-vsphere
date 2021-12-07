package controllers

/*import (
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/meta"
	ctrlmgr "sigs.k8s.io/controller-runtime/pkg/manager"

	infrav1b1 "sigs.k8s.io/cluster-api-provider-vsphere/apis/v1beta1"
	vmwarev1b1 "sigs.k8s.io/cluster-api-provider-vsphere/apis/vmware/v1beta1"
	"sigs.k8s.io/cluster-api-provider-vsphere/controllers"
	"sigs.k8s.io/cluster-api-provider-vsphere/pkg/context"
	"sigs.k8s.io/cluster-api-provider-vsphere/pkg/manager"
)

func AddManagerFuncWithLogger(logger logr.Logger) manager.AddToManagerFunc {
	return func(ctx *context.ControllerManagerContext, mgr ctrlmgr.Manager) error {
		cluster := &infrav1b1.VSphereCluster{}
		gvr := infrav1b1.GroupVersion.WithResource(reflect.TypeOf(cluster).Elem().Name())
		_, err := mgr.GetRESTMapper().KindFor(gvr)
		if err != nil {
			if meta.IsNoMatchError(err) {
				logger.Info(fmt.Sprintf("CRD for %s not loaded, skipping.", gvr.String()))
			} else {
				return err
			}
		} else {
			if err := setupVAPIControllers(ctx, mgr); err != nil {
				return err
			}
		}

		supervisorCluster := &vmwarev1b1.VSphereCluster{}
		gvr = vmwarev1b1.GroupVersion.WithResource(reflect.TypeOf(supervisorCluster).Elem().Name())
		_, err = mgr.GetRESTMapper().KindFor(gvr)
		if err != nil {
			if meta.IsNoMatchError(err) {
				logger.Info(fmt.Sprintf("CRD for %s not loaded, skipping.", gvr.String()))
			} else {
				return err
			}
		} else {
			if err := setupSupervisorControllers(ctx, mgr); err != nil {
				return err
			}
		}

		return nil
	}
}

func setupVAPIControllers(ctx *context.ControllerManagerContext, mgr ctrlmgr.Manager) error {
	if err := (&infrav1b1.VSphereClusterTemplate{}).SetupWebhookWithManager(mgr); err != nil {
		return err
	}

	if err := (&infrav1b1.VSphereMachine{}).SetupWebhookWithManager(mgr); err != nil {
		return err
	}
	if err := (&infrav1b1.VSphereMachineList{}).SetupWebhookWithManager(mgr); err != nil {
		return err
	}

	if err := (&infrav1b1.VSphereMachineTemplate{}).SetupWebhookWithManager(mgr); err != nil {
		return err
	}
	if err := (&infrav1b1.VSphereMachineTemplateList{}).SetupWebhookWithManager(mgr); err != nil {
		return err
	}

	if err := (&infrav1b1.VSphereVM{}).SetupWebhookWithManager(mgr); err != nil {
		return err
	}
	if err := (&infrav1b1.VSphereVMList{}).SetupWebhookWithManager(mgr); err != nil {
		return err
	}

	if err := (&infrav1b1.VSphereDeploymentZone{}).SetupWebhookWithManager(mgr); err != nil {
		return err
	}
	if err := (&infrav1b1.VSphereDeploymentZoneList{}).SetupWebhookWithManager(mgr); err != nil {
		return err
	}

	if err := (&infrav1b1.VSphereFailureDomain{}).SetupWebhookWithManager(mgr); err != nil {
		return err
	}
	if err := (&infrav1b1.VSphereFailureDomainList{}).SetupWebhookWithManager(mgr); err != nil {
		return err
	}

	if err := controllers.AddClusterControllerToManager(ctx, mgr, &infrav1b1.VSphereCluster{}); err != nil {
		return err
	}
	if err := controllers.AddMachineControllerToManager(ctx, mgr, &infrav1b1.VSphereMachine{}); err != nil {
		return err
	}
	if err := controllers.AddVMControllerToManager(ctx, mgr); err != nil {
		return err
	}
	if err := controllers.AddVsphereClusterIdentityControllerToManager(ctx, mgr); err != nil {
		return err
	}
	if err := controllers.AddVSphereDeploymentZoneControllerToManager(ctx, mgr); err != nil {
		return err
	}
	return nil
}

func setupSupervisorControllers(ctx *context.ControllerManagerContext, mgr ctrlmgr.Manager) error {
	if err := controllers.AddClusterControllerToManager(ctx, mgr, &vmwarev1b1.VSphereCluster{}); err != nil {
		return err
	}

	if err := controllers.AddMachineControllerToManager(ctx, mgr, &vmwarev1b1.VSphereMachine{}); err != nil {
		return err
	}

	if err := controllers.AddServiceAccountProviderControllerToManager(ctx, mgr); err != nil {
		return err
	}

	if err := controllers.AddServiceDiscoveryControllerToManager(ctx, mgr); err != nil {
		return err
	}

	return nil
}*/
