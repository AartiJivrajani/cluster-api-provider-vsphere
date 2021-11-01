// Copyright (c) 2019 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	//tkgv1 "gitlab.eng.vmware.com/core-build/guest-cluster-controller/apis/run.tanzu/v1alpha2"
	//"gitlab.eng.vmware.com/core-build/guest-cluster-controller/controllers/svcdiscovery"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	bootstrapapi "k8s.io/cluster-bootstrap/token/api"
	_ "sigs.k8s.io/cluster-api-provider-vsphere/apis/vmware/v1beta1"
	"sigs.k8s.io/cluster-api-provider-vsphere/pkg/context"
	"sigs.k8s.io/cluster-api-provider-vsphere/pkg/context/fake"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

func unitTests() {
	Describe("Invoking ReconcileNormal", unitTestsReconcileNormal)
}

func unitTestsReconcileNormal() {
	var (
		//ctx           *builder.UnitTestContextForController
		controllerCtx *context.ControllerContext
		clusterCtx    *context.ClusterContext
		initObjects   []runtime.Object
	)
	BeforeEach(func() {
		controllerCtx = fake.NewControllerContext(fake.NewControllerManagerContext(initObjects...))
		clusterCtx = fake.NewClusterContext(controllerCtx)
	})

	// JustBeforeEach(func() {
	// 	ctx = suite.NewUnitTestContextForController(initObjects...)

	// })
	JustAfterEach(func() {
		controllerCtx = nil
	})
	Context("When no VIP or FIP is available ", func() {
		It("Should reconcile headless svc", func() {
			By("creating a service and no endpoint in the guest cluster")
			assertHeadlessSvcWithNoEndpoints(ctx, clusterCtx.Client, SupervisorHeadlessSvcNamespace, SupervisorHeadlessSvcName)
			assertServiceDiscoveryCondition(clusterCtx.VSphereCluster, corev1.ConditionFalse, "Unable to discover supervisor apiserver address",
				SupervisorHeadlessServiceSetupFailedReason, clusterv1.ConditionSeverityWarning)
		})
	})
	Context("When VIP is available", func() {
		BeforeEach(func() {
			initObjects = []runtime.Object{
				newTestSupervisorLBServiceWithIPStatus()}
		})
		It("Should reconcile headless svc", func() {
			By("creating a service and endpoints using the VIP in the guest cluster")
			assertHeadlessSvcWithVIPEndpoints(ctx, clusterCtx.Client, SupervisorHeadlessSvcNamespace, SupervisorHeadlessSvcName)
			assertServiceDiscoveryCondition(clusterCtx.VSphereCluster, corev1.ConditionTrue, "", "", "")
		})
		It("Should get supervisor master endpoint IP", func() {
			supervisorEndpointIP, err := GetSupervisorAPIServerAddress(clusterCtx)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(supervisorEndpointIP).To(Equal(testSupervisorAPIServerVIP))
		})
	})
	Context("When FIP is available", func() {
		BeforeEach(func() {
			initObjects = []runtime.Object{
				newTestConfigMapWithHost(testSupervisorAPIServerFIP)}
		})
		It("Should reconcile headless svc", func() {
			By("creating a service and endpoints using the FIP in the guest cluster")
			assertHeadlessSvcWithFIPEndpoints(ctx, clusterCtx.Client, SupervisorHeadlessSvcNamespace, SupervisorHeadlessSvcName)
			assertServiceDiscoveryCondition(clusterCtx.VSphereCluster, corev1.ConditionTrue, "", "", "")
		})
	})
	Context("When VIP and FIP are available", func() {
		BeforeEach(func() {
			initObjects = []runtime.Object{
				newTestSupervisorLBServiceWithIPStatus(),
				newTestConfigMapWithHost(testSupervisorAPIServerFIP)}
		})
		It("Should reconcile headless svc", func() {
			By("creating a service and endpoints using the VIP in the guest cluster")
			assertHeadlessSvcWithVIPEndpoints(ctx, clusterCtx.Client, SupervisorHeadlessSvcNamespace, SupervisorHeadlessSvcName)
			assertServiceDiscoveryCondition(clusterCtx.VSphereCluster, corev1.ConditionTrue, "", "", "")
		})
	})
	Context("When VIP is an hostname", func() {
		BeforeEach(func() {
			initObjects = []runtime.Object{
				newTestSupervisorLBServiceWithHostnameStatus()}
		})
		It("Should reconcile headless svc", func() {
			By("creating a service and endpoints using the VIP in the guest cluster")
			assertHeadlessSvcWithVIPHostnameEndpoints(ctx, clusterCtx.Client, SupervisorHeadlessSvcNamespace, SupervisorHeadlessSvcName)
			assertServiceDiscoveryCondition(clusterCtx.VSphereCluster, corev1.ConditionTrue, "", "", "")
		})
	})
	Context("When FIP is an hostname", func() {
		BeforeEach(func() {
			initObjects = []runtime.Object{
				newTestConfigMapWithHost(testSupervisorAPIServerFIPHostName),
			}
		})
		It("Should reconcile headless svc", func() {
			By("creating a service and endpoints using the FIP in the guest cluster")
			assertHeadlessSvcWithFIPHostNameEndpoints(ctx, clusterCtx.Client, SupervisorHeadlessSvcNamespace, SupervisorHeadlessSvcName)
			assertServiceDiscoveryCondition(clusterCtx.VSphereCluster, corev1.ConditionTrue, "", "", "")
		})
	})
	Context("When FIP is an empty hostname", func() {
		BeforeEach(func() {
			initObjects = []runtime.Object{
				newTestConfigMapWithHost(""),
			}
		})
		It("Should reconcile headless svc", func() {
			By("creating a service and no endpoint in the guest cluster")
			assertHeadlessSvcWithNoEndpoints(ctx, clusterCtx.Client, SupervisorHeadlessSvcNamespace, SupervisorHeadlessSvcName)
			assertServiceDiscoveryCondition(clusterCtx.VSphereCluster, corev1.ConditionFalse, "Unable to discover supervisor apiserver address",
				SupervisorHeadlessServiceSetupFailedReason, clusterv1.ConditionSeverityWarning)
		})
	})
	Context("When FIP is an invalid host", func() {
		BeforeEach(func() {
			initObjects = []runtime.Object{
				newTestConfigMapWithHost("host^name"),
			}
		})
		It("Should reconcile headless svc", func() {
			By("creating a service and no endpoint in the guest cluster")
			assertHeadlessSvcWithNoEndpoints(ctx, clusterCtx.Client, SupervisorHeadlessSvcNamespace, SupervisorHeadlessSvcName)
			assertServiceDiscoveryCondition(clusterCtx.VSphereCluster, corev1.ConditionFalse, "Unable to discover supervisor apiserver address",
				SupervisorHeadlessServiceSetupFailedReason, clusterv1.ConditionSeverityWarning)
		})
	})
	Context("When FIP config map has invalid kubeconfig data", func() {
		BeforeEach(func() {
			initObjects = []runtime.Object{
				newTestConfigMapWithData(
					map[string]string{
						bootstrapapi.KubeConfigKey: "invalid-kubeconfig-data",
					}),
			}
		})
		It("Should reconcile headless svc", func() {
			By("creating a service and no endpoint in the guest cluster")
			assertHeadlessSvcWithNoEndpoints(ctx, clusterCtx.Client, SupervisorHeadlessSvcNamespace, SupervisorHeadlessSvcName)
			assertServiceDiscoveryCondition(clusterCtx.VSphereCluster, corev1.ConditionFalse, "Unable to discover supervisor apiserver address",
				SupervisorHeadlessServiceSetupFailedReason, clusterv1.ConditionSeverityWarning)
		})
	})
	Context("When FIP config map has invalid kubeconfig key", func() {
		BeforeEach(func() {
			initObjects = []runtime.Object{
				newTestConfigMapWithData(
					map[string]string{
						"invalid-key": "invalid-kubeconfig-data",
					}),
			}
		})
		It("Should reconcile headless svc", func() {
			By("creating a service and no endpoint in the guest cluster")
			assertHeadlessSvcWithNoEndpoints(ctx, clusterCtx.Client, SupervisorHeadlessSvcNamespace, SupervisorHeadlessSvcName)
			assertServiceDiscoveryCondition(clusterCtx.VSphereCluster, corev1.ConditionFalse, "Unable to discover supervisor apiserver address",
				SupervisorHeadlessServiceSetupFailedReason, clusterv1.ConditionSeverityWarning)
		})
	})
}
