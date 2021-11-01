// Copyright (c) 2019 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package builder

import (
	gocontext "context"
	"fmt"

	"github.com/imdario/mergo"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	bootstrapv1 "sigs.k8s.io/cluster-api/bootstrap/kubeadm/api/v1alpha3"
	kcpv1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	cmv1alpha2 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha2"
	vmoprv1 "github.com/vmware-tanzu/vm-operator-api/api/v1alpha1"

	netopv1alpha1 "github.com/vmware-tanzu/net-operator-api/api/v1alpha1"
	infrav1 "gitlab.eng.vmware.com/core-build/cluster-api-provider-wcp/api/v1alpha3"

	tkgv1 "gitlab.eng.vmware.com/core-build/guest-cluster-controller/apis/run.tanzu/v1alpha2"
	"gitlab.eng.vmware.com/core-build/guest-cluster-controller/controllers/util"
	"gitlab.eng.vmware.com/core-build/guest-cluster-controller/controllers/util/version"
	"gitlab.eng.vmware.com/core-build/guest-cluster-controller/controllers/util/vlabel"
	"gitlab.eng.vmware.com/core-build/guest-cluster-controller/pkg/builder"
	"gitlab.eng.vmware.com/core-build/guest-cluster-controller/pkg/context"
	"gitlab.eng.vmware.com/core-build/guest-cluster-controller/pkg/context/fake"
)

// UnitTestContextForController is used for unit testing controllers.
type UnitTestContextForController struct {
	// GuestClusterContext is initialized with fake.NewGuestClusterContext
	// and is used for unit testing.
	context.GuestClusterContext

	// Key may be used to lookup Ctx.Cluster with Ctx.Client.Get.
	Key client.ObjectKey

	VirtualMachineImage *vmoprv1.VirtualMachineImage

	TKR *tkgv1.TanzuKubernetesRelease

	// reconciler is the builder.Reconciler being unit tested.
	Reconciler builder.Reconciler
}

// UnitTestContextForValidatingWebhook is used for unit testing validating
// webhooks.
type UnitTestContextForValidatingWebhook struct {
	// WebhookRequestContext is initialized with fake.NewWebhookRequestContext
	// and is used for unit testing.
	context.WebhookRequestContext

	// Key may be used to lookup Ctx.Cluster with Ctx.Client.Get.
	Key client.ObjectKey

	// validator is the builder.Validator being unit tested.
	builder.Validator
}

// NewUnitTestContextForController returns a new UnitTestContextForController
// with an optional prototype cluster for unit testing controllers that do not
// invoke the TanzuKubernetesCluster spec controller
func NewUnitTestContextForController(newReconcilerFn builder.NewReconcilerFunc,
	tkc *tkgv1.TanzuKubernetesCluster,
	prototypeCluster bool,
	initObjects,
	gcInitObjects []runtime.Object) *UnitTestContextForController {

	fakeClient, scheme := NewFakeClient(initObjects...)
	reconciler := newReconcilerFn()

	ctx := &UnitTestContextForController{
		GuestClusterContext: *(fake.NewGuestClusterContext(fake.NewClusterContext(fake.NewControllerContext(fake.NewControllerManagerContext(fakeClient, scheme)), tkc), prototypeCluster, gcInitObjects...)),
		Reconciler:          reconciler,
	}
	ctx.Key = client.ObjectKey{Namespace: ctx.Cluster.Namespace, Name: ctx.Cluster.Name}

	CreatePrototypePrereqs(ctx, ctx.ControllerManagerContext)

	return ctx
}

// NewUnitTestContextForValidatingWebhook returns a new
// UnitTestContextForValidatingWebhook for unit testing validating webhooks.
func NewUnitTestContextForValidatingWebhook(validator builder.Validator, initObjects ...runtime.Object) *UnitTestContextForValidatingWebhook {
	fakeClient, scheme := NewFakeClient(initObjects...)

	ctx := &UnitTestContextForValidatingWebhook{
		WebhookRequestContext: *(fake.NewWebhookRequestContext(fake.NewWebhookContext(fake.NewControllerManagerContext(fakeClient, scheme)))),
		Key:                   client.ObjectKey{Namespace: fake.Namespace, Name: fake.TanzuKubernetesClusterName},
		Validator:             validator,
	}

	CreatePrototypePrereqs(nil, ctx.ControllerManagerContext)

	return ctx
}

func CreatePrototypePrereqs(unitTestCtx *UnitTestContextForController, ctx *context.ControllerManagerContext) {
	By("Creating a prototype VirtualMachineClass", func() {
		virtualMachineClass := FakeVirtualMachineClass()
		virtualMachineClass.Name = "small"
		Expect(ctx.Client.Create(ctx, virtualMachineClass)).To(Succeed())
		virtualMachineClassKey := client.ObjectKey{Name: virtualMachineClass.Name}
		Eventually(func() error {
			return ctx.Client.Get(ctx, virtualMachineClassKey, virtualMachineClass)
		}).Should(Succeed())
	})

	By("Creating a prototype VirtualMachineImage", func() {
		virtualMachineImage := CreateFakeVirtualMachineImage()
		// virtualMachineImage.Name = FakeDistributionVersion
		Expect(ctx.Client.Create(ctx, virtualMachineImage)).To(Succeed())
		tkr, tkrAddons := CreateFakeTKR(virtualMachineImage)
		Expect(PersistAddons(ctx, ctx.Client, tkrAddons)).To(Succeed())
		Expect(PersistTKR(ctx, ctx.Client, tkr))
		if unitTestCtx != nil {
			unitTestCtx.VirtualMachineImage = virtualMachineImage
			unitTestCtx.TKR = tkr
		}
		virtualMachineImageKey := client.ObjectKey{Namespace: virtualMachineImage.Namespace, Name: virtualMachineImage.Name}
		Eventually(func() error {
			return ctx.Client.Get(ctx, virtualMachineImageKey, virtualMachineImage)
		}).Should(Succeed())

		incompatibleVirtualMachineImage := CreateFakeIncompatibleVirtualMachineImage()
		Expect(ctx.Client.Create(ctx, incompatibleVirtualMachineImage)).To(Succeed())
		incompatibleTKR, incompatibleTKRAddons := CreateFakeIncompatibleTKR(incompatibleVirtualMachineImage)
		Expect(PersistAddons(ctx, ctx.Client, incompatibleTKRAddons)).To(Succeed())
		Expect(PersistTKR(ctx, ctx.Client, incompatibleTKR)).To(Succeed())
		incompatibleVirtualMachineImageKey := client.ObjectKey{Namespace: incompatibleVirtualMachineImage.Namespace, Name: incompatibleVirtualMachineImage.Name}
		Eventually(func() error {
			return ctx.Client.Get(ctx, incompatibleVirtualMachineImageKey, incompatibleVirtualMachineImage)
		}).Should(Succeed())
	})
}

func PersistTKR(ctx gocontext.Context, c client.Client, tkr *tkgv1.TanzuKubernetesRelease) error {
	persistedTKR := &tkgv1.TanzuKubernetesRelease{ObjectMeta: metav1.ObjectMeta{Name: tkr.Name}}
	_, err := controllerutil.CreateOrUpdate(ctx, c, persistedTKR, func() error {
		tkr.ResourceVersion = persistedTKR.ResourceVersion
		tkr.DeepCopyInto(persistedTKR)
		return nil
	})
	persistedTKR.DeepCopyInto(tkr)
	return err
}

func PersistAddons(ctx gocontext.Context, c client.Client, tkas []*tkgv1.TanzuKubernetesAddon) error {
	for _, addon := range tkas {
		addon := addon // capture the var for closure (func())
		persistedAddon := &tkgv1.TanzuKubernetesAddon{ObjectMeta: metav1.ObjectMeta{Name: addon.Name}}
		res, err := controllerutil.CreateOrUpdate(ctx, c, persistedAddon, func() error {
			err := mergo.Merge(persistedAddon, addon, mergo.WithOverride)
			return err
		})
		if err != nil {
			return errors.Wrapf(err, "res: %s", res)
		}
	}
	return nil
}

// UnitTestContextForMutatingWebhook is used for unit testing mutating webhooks.
type UnitTestContextForMutatingWebhook struct {
	// WebhookRequestContext is initialized with fake.NewWebhookRequestContext
	// and is used for unit testing.
	context.WebhookRequestContext

	// Key may be used to lookup Ctx.Cluster with Ctx.Client.Get.
	Key client.ObjectKey

	// mutator is the builder.Mutator being unit tested.
	builder.Mutator
}

// NewUnitTestContextForMutatingWebhook returns a new UnitTestContextForMutatingWebhook for unit testing mutating webhooks.
func NewUnitTestContextForMutatingWebhook(mutator builder.Mutator, virtualMachineImageVersions ...string) *UnitTestContextForMutatingWebhook {
	fakeClient, scheme := NewFakeClient()

	ctx := &UnitTestContextForMutatingWebhook{
		WebhookRequestContext: *(fake.NewWebhookRequestContext(fake.NewWebhookContext(fake.NewControllerManagerContext(fakeClient, scheme)))),
		Key:                   client.ObjectKey{Namespace: fake.Namespace, Name: fake.TanzuKubernetesClusterName},
		Mutator:               mutator,
	}

	CreateTKRs(ctx.ControllerManagerContext, ctx.Client, true, virtualMachineImageVersions)

	return ctx
}

type latestMarker map[string]string

func newLatestMarker(compatible bool, versions []string) latestMarker {
	if !compatible {
		return latestMarker{}
	}
	m := latestMarker{}
	for _, v := range versions {
		v, _ := util.NormalizeSemver(v)
		sv, _ := version.ParseSemantic(v)
		if sv == nil {
			continue
		}
		for _, vLabel := range vlabel.Prefixes(util.TKRName(v)) {
			latest, filled := m[vLabel]
			if !filled {
				m[vLabel] = v
				continue
			}
			latestSV, _ := version.ParseSemantic(latest)
			if latestSV.LessThan(sv) {
				m[vLabel] = v
			}
		}
	}
	return m
}

func (m latestMarker) Label(tkr *tkgv1.TanzuKubernetesRelease) {
	for vLabel := range tkr.Labels {
		if m[vLabel] == tkr.Spec.Version {
			tkr.Labels[vLabel] = vlabel.ValueLatest
		}
	}
}

func CreateTKRs(ctx gocontext.Context, c client.Client, compatible bool, distributionVersions []string) {
	labeler := newLatestMarker(compatible, distributionVersions)

	for _, distributionVersion := range distributionVersions {
		By(fmt.Sprintf("Creating a TanzuKubernetesRelease with version %s", distributionVersion))
		tkr, tkrAddons := NewTKR(nil, distributionVersion, FakeKubernetesVersion, FakeEtcdVersion, FakeDNSVersion, FakeCNIVersion, FakeCSIVersion, FakeCPIVersion, FakeAuthServiceVersion, FakeMetricsServerVersion, compatible, true)
		labeler.Label(tkr)
		Expect(PersistAddons(ctx, c, tkrAddons)).To(Succeed())
		Expect(PersistTKR(ctx, c, tkr)).To(Succeed())
	}
}

func SetTKRRefForVersion(tkc *tkgv1.TanzuKubernetesCluster, version string) {
	tkc.Spec.Distribution.Version = version
	SetTKRRef(tkc, tkgv1.TKRRef(version))
}

func SetTKRRef(tkc *tkgv1.TanzuKubernetesCluster, tkrReference *corev1.ObjectReference) {
	tkc.Spec.Topology.ControlPlane.TKR.Reference = tkrReference
	for i := range tkc.Spec.Topology.NodePools {
		tkc.Spec.Topology.NodePools[i].TKR.Reference = tkrReference
	}
}

func NewFakeClient(initObjects ...runtime.Object) (client.Client, *runtime.Scheme) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = clusterv1.AddToScheme(scheme)
	_ = bootstrapv1.AddToScheme(scheme)
	_ = kcpv1.AddToScheme(scheme)
	_ = infrav1.AddToScheme(scheme)
	_ = vmoprv1.AddToScheme(scheme)
	_ = tkgv1.AddToScheme(scheme)
	_ = apiextv1beta1.AddToScheme(scheme)
	_ = cmv1alpha2.AddToScheme(scheme)
	_ = netopv1alpha1.AddToScheme(scheme)
	fakeClient := clientfake.NewFakeClientWithScheme(scheme, initObjects...)

	return fakeClient, scheme
}

// ReconcileNormal manually invokes the ReconcileNormal method on the controller
func (ctx UnitTestContextForController) ReconcileNormal() error {
	var err error
	switch t := ctx.Reconciler.(type) {
	default:
		panic("Unexpected type")
	case builder.GuestClusterReconciler:
		_, err = t.ReconcileNormal(&ctx.GuestClusterContext)
	case builder.TanzuKubernetesClusterReconciler:
		_, err = t.ReconcileNormal(ctx.GuestClusterContext.ClusterContext)
	}

	return err
}
