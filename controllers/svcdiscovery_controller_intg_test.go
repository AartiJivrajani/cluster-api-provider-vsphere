package controllers

import (
	goctx "context"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"log"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"sigs.k8s.io/cluster-api-provider-vsphere/pkg/builder"
)

var _ = FDescribe("Service Discovery controller integration tests", func() {
	var (
		intCtx         *builder.IntegrationTestContext
		initObjects []client.Object
	)
	BeforeEach(func() {
		serviceDiscoveryTestSuite.SetIntegrationTestClient(testEnv.Manager.GetClient())
		intCtx = serviceDiscoveryTestSuite.NewIntegrationTestContextWithClusters(goctx.Background(), testEnv.Manager.GetClient(), true)
	})
	AfterEach(func() {
		intCtx.AfterEach()
		intCtx = nil
	})

	doAssertions := func() {
		// TODO: change the key of this annotation.
		intCtx.VSphereCluster.Annotations = map[string]string{"run.tanzu.test.update": "true"}
		Expect(intCtx.PatchHelper.Patch(intCtx, intCtx.VSphereCluster)).To(Succeed())

		initObjects = []client.Object{
			newTestSupervisorLBServiceWithIPStatus(),
		}
		createObjects(intCtx, intCtx.Client, initObjects)
		Expect(intCtx.Client.Status().Update(ctx, newTestSupervisorLBServiceWithIPStatus())).To(Succeed())
		//updateObjectsStatus(intCtx, intCtx.Client, initObjects)

		headlessSvc := &corev1.Service{}
		assertEventuallyExistsInNamespace(intCtx, intCtx.Client, "kube-system",  "kube-apiserver-lb-svc", headlessSvc)
		log.Printf("~~~~~~~~~~~~~ IP(do assertions) %v", headlessSvc.Status.LoadBalancer.Ingress[0].IP)

		//By("creating a service and no endpoint in the guest cluster")
		assertHeadlessSvcWithNoEndpoints(intCtx, intCtx.GuestClient, supervisorHeadlessSvcNamespace, supervisorHeadlessSvcName)
	}

	Context("When the TanzuKubernetesCluster is updated", func() {
		//BeforeEach(func() {
		//	// TODO: change the key of this annotation.
		//	intCtx.VSphereCluster.Annotations = map[string]string{"run.tanzu.test.update": "true"}
		//	Expect(intCtx.PatchHelper.Patch(intCtx, intCtx.VSphereCluster)).To(Succeed())
		//
		//	initObjects = []client.Object{
		//		newTestSupervisorLBServiceWithIPStatus(),
		//	}
		//	createObjects(intCtx, intCtx.Client, initObjects)
		//	updateObjectsStatus(intCtx, intCtx.Client, initObjects)
		//})
		FIt("Should reconcile headless svc", doAssertions)
	})

	//Context("When the CAPI Cluster is updated", func() {
	//	BeforeEach(func() {
	//		intCtx.Cluster.Annotations = map[string]string{"run.tanzu.test.update": "true"}
	//		Expect(intCtx.Client.Update(intCtx, intCtx.VSphereCluster)).To(Succeed())
	//	})
	//	It("Should reconcile headless svc", doAssertions)
	//})
	Context("When VIP is available", func() {
		BeforeEach(func() {
			initObjects = []client.Object{
				newTestSupervisorLBServiceWithIPStatus(),
			}
			//createObjects(intCtx, intCtx.Client, initObjects)
			//updateObjectsStatus(intCtx, intCtx.Client, initObjects)
		})
		AfterEach(func() {
			//deleteObjects(intCtx, intCtx.Client, initObjects)
		})
		It("Should reconcile headless svc", func() {
			By("creating a service and endpoints using the VIP in the guest cluster")
			// kube-system/kube-apiserver-lb-svc
			headlessSvc := &corev1.Service{}
			assertEventuallyExistsInNamespace(intCtx, intCtx.Client, "kube-system",  "kube-apiserver-lb-svc", headlessSvc)
			log.Printf("~~~~~~~~~~~~~ IP %v", headlessSvc.Status.LoadBalancer.Ingress[0].IP)
			Expect(headlessSvc.Status.LoadBalancer.Ingress[0].IP).ToNot(BeEmpty())
			//assertHeadlessSvcWithVIPEndpoints(intCtx, intCtx.GuestClient, supervisorHeadlessSvcNamespace, supervisorHeadlessSvcName)
		})
	})

	Context("When FIP is available", func() {
		BeforeEach(func() {
			//initObjects = []client.Object{
			//	newTestConfigMapWithHost(testSupervisorAPIServerFIP)}
			//createObjects(intCtx, intCtx.Client, initObjects)
		})
		AfterEach(func() {
			deleteObjects(intCtx, intCtx.Client, initObjects)
		})
		It("Should reconcile headless svc", func() {
			By("creating a service and endpoints using the FIP in the guest cluster")
			assertHeadlessSvcWithFIPEndpoints(intCtx, intCtx.GuestClient, supervisorHeadlessSvcNamespace, supervisorHeadlessSvcName)
		})
	})
	Context("When headless svc and endpoints already exists", func() {
		BeforeEach(func() {
			// Create the svc & endpoint objects in guest cluster
			createObjects(intCtx, intCtx.GuestClient, newTestHeadlessSvcEndpoints())
			// Init objects in the supervisor cluster
			initObjects = []client.Object{
				newTestSupervisorLBServiceWithIPStatus()}
			createObjects(intCtx, intCtx.Client, initObjects)
			updateObjectsStatus(intCtx, intCtx.Client, initObjects)
		})
		AfterEach(func() {
			deleteObjects(intCtx, intCtx.Client, initObjects)
			// Note: No need to delete guest cluster objects as a new guest cluster testenv endpoint is created for each test.
		})
		It("Should reconcile headless svc", func() {
			By("updating the service and endpoints using the VIP in the guest cluster")
			assertHeadlessSvcWithUpdatedVIPEndpoints(intCtx, intCtx.GuestClient, supervisorHeadlessSvcNamespace, supervisorHeadlessSvcName)
		})
	})

	})
