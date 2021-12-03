// Copyright (c) 2019 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package controllers

//
//func intgTests() {
//	var (
//		ctx         *builder.IntegrationTestContext
//		initObjects []runtime.Object
//	)
//	BeforeEach(func() {
//		ctx = serviceAccountProviderTestsuite.NewIntegrationTestContextWithClusters(true, false)
//	})
//	AfterEach(func() {
//		ctx.AfterEach()
//		ctx = nil
//	})
//
//	doAssertions := func() {
//		By("creating a service and no endpoint in the guest cluster")
//		assertHeadlessSvcWithNoEndpoints(ctx, ctx.GuestClient, svcdiscovery.SupervisorHeadlessSvcNamespace, svcdiscovery.SupervisorHeadlessSvcName)
//	}
//
//	Context("When the TanzuKubernetesCluster is updated", func() {
//		BeforeEach(func() {
//			ctx.TanzuKubernetesCluster.Annotations = map[string]string{"run.tanzu.test.update": "true"}
//			Expect(ctx.PatchHelper.Patch(ctx, ctx.TanzuKubernetesCluster)).To(Succeed())
//		})
//		It("Should reconcile headless svc", doAssertions)
//	})
//
//	Context("When the CAPI Cluster is updated", func() {
//		BeforeEach(func() {
//			ctx.Cluster.Annotations = map[string]string{"run.tanzu.test.update": "true"}
//			Expect(ctx.Client.Update(ctx, ctx.Cluster)).To(Succeed())
//		})
//		It("Should reconcile headless svc", doAssertions)
//	})
//	Context("When VIP is available", func() {
//		BeforeEach(func() {
//			initObjects = []runtime.Object{
//				newTestSupervisorLBServiceWithIPStatus()}
//			createObjects(ctx, ctx.Client, initObjects)
//			updateObjectsStatus(ctx, ctx.Client, initObjects)
//		})
//		AfterEach(func() {
//			deleteObjects(ctx, ctx.Client, initObjects)
//		})
//		It("Should reconcile headless svc", func() {
//			By("creating a service and endpoints using the VIP in the guest cluster")
//			assertHeadlessSvcWithVIPEndpoints(ctx, ctx.GuestClient, svcdiscovery.SupervisorHeadlessSvcNamespace, svcdiscovery.SupervisorHeadlessSvcName)
//		})
//	})
//	Context("When FIP is available", func() {
//		BeforeEach(func() {
//			initObjects = []runtime.Object{
//				newTestConfigMapWithHost(testSupervisorAPIServerFIP)}
//			createObjects(ctx, ctx.Client, initObjects)
//		})
//		AfterEach(func() {
//			deleteObjects(ctx, ctx.Client, initObjects)
//		})
//		It("Should reconcile headless svc", func() {
//			By("creating a service and endpoints using the FIP in the guest cluster")
//			assertHeadlessSvcWithFIPEndpoints(ctx, ctx.GuestClient, svcdiscovery.SupervisorHeadlessSvcNamespace, svcdiscovery.SupervisorHeadlessSvcName)
//		})
//	})
//	Context("When headless svc and endpoints already exists", func() {
//		BeforeEach(func() {
//			// Create the svc & endpoint objects in guest cluster
//			createObjects(ctx, ctx.GuestClient, newTestHeadlessSvcEndpoints())
//			// Init objects in the supervisor cluster
//			initObjects = []runtime.Object{
//				newTestSupervisorLBServiceWithIPStatus()}
//			createObjects(ctx, ctx.Client, initObjects)
//			updateObjectsStatus(ctx, ctx.Client, initObjects)
//		})
//		AfterEach(func() {
//			deleteObjects(ctx, ctx.Client, initObjects)
//			// Note: No need to delete guest cluster objects as a new guest cluster testenv endpoint is created for each test.
//		})
//		It("Should reconcile headless svc", func() {
//			By("updating the service and endpoints using the VIP in the guest cluster")
//			assertHeadlessSvcWithUpdatedVIPEndpoints(ctx, ctx.GuestClient, svcdiscovery.SupervisorHeadlessSvcNamespace, svcdiscovery.SupervisorHeadlessSvcName)
//		})
//	})
//}
