// Copyright (c) 2019 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package serviceaccount_test

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	tkgv1 "gitlab.eng.vmware.com/core-build/guest-cluster-controller/apis/run.tanzu/v1alpha2"
	//"gitlab.eng.vmware.com/core-build/guest-cluster-controller/pkg/context/fake"
	"sigs.k8s.io/cluster-api-provider-vsphere/pkg/context/fake"
	"gitlab.eng.vmware.com/core-build/guest-cluster-controller/test/builder"
	vmware_capv "sigs.k8s.io/cluster-api-provider-vsphere/apis/vmware/v1beta1"
	"sigs.k8s.io/cluster-api-provider-vsphere/pkg/context"
)

func unitTests() {
	Describe("Invoking ReconcileNormal", unitTestsReconcileNormal)
}

func unitTestsReconcileNormal() {
	var (
		//ctx         *builder.UnitTestContextForController
		//tkc         *tkgv1.TanzuKubernetesCluster
		controllerCtx *context.ControllerContext
		clusterCtx    *context.ClusterContext
		vSphereCluster vmware_capv.VSphereCluster
		initObjects []runtime.Object
	)

	JustBeforeEach(func() {
		// Note: The provider service account requires a reference to the tanzukubernetescluster hence the need to create
		// a fake tanzu kubernetes cluster in the test and pass it to during context setup.
		controllerCtx = fake.NewControllerContext(fake.NewControllerManagerContext(initObjects...))
		clusterCtx = fake.NewClusterContext(controllerCtx)
		//ctx = suite.NewUnitTestContextForControllerWithTanzuKubernetesCluster(tkc, false, initObjects...)
		//suite.NewUnitTestContextForControllerWithPrototypeCluster(initObjects...)
	})
	AfterEach(func() {
		clusterCtx = nil
	})

	Context("When no provider service account is available", func() {
		It("Should reconcile", func() {
			By("Not creating any entities")
			assertNoEntities(clusterCtx, clusterCtx.Client, testNS)
			assertProviderServiceAccountsCondition(clusterCtx.Cluster, corev1.ConditionTrue, "", "", "")
		})
	})

	Describe("When the ProviderServiceAccount is created", func() {
		BeforeEach(func() {
			obj := fake.NewTanzuKubernetesCluster(3, 3)
			tkc = &obj
			tkc.Namespace = testNS
			_ = os.Setenv("SERVICE_ACCOUNTS_CM_NAMESPACE", testSystemSvcAcctNs)
			_ = os.Setenv("SERVICE_ACCOUNTS_CM_NAME", testSystemSvcAcctCM)
			initObjects = []runtime.Object{
				getSystemServiceAccountsConfigMap(testSystemSvcAcctNs, testSystemSvcAcctCM),
				getTestProviderServiceAccount(testNS, testProviderSvcAccountName, tkc),
			}
		})
		Context("When serviceaccount secret is created", func() {
			It("Should reconcile", func() {
				assertTargetNamespace(ctx, ctx.GuestClient, testTargetNS, false)
				updateServiceAccountSecretAndReconcileNormal(ctx)
				assertTargetNamespace(ctx, ctx.GuestClient, testTargetNS, true)
				By("Creating the target secret in the target namespace")
				assertTargetSecret(ctx, ctx.GuestClient, testTargetNS, testTargetSecret)
				assertProviderServiceAccountsCondition(ctx.Cluster, corev1.ConditionTrue, "", "", "")
			})
		})
		Context("When serviceaccount secret is modified", func() {
			It("Should reconcile", func() {
				// This is to simulate an outdate token that will be replaced when the serviceaccount secret is created.
				createTargetSecretWithInvalidToken(ctx, ctx.GuestClient)
				updateServiceAccountSecretAndReconcileNormal(ctx)
				By("Updating the target secret in the target namespace")
				assertTargetSecret(ctx, ctx.GuestClient, testTargetNS, testTargetSecret)
				assertProviderServiceAccountsCondition(ctx.Cluster, corev1.ConditionTrue, "", "", "")
			})
		})
		Context("When invalid role exists", func() {
			BeforeEach(func() {
				initObjects = append(initObjects, getTestRoleWithGetPod(testNS, testRoleName))
			})
			It("Should update role", func() {
				assertRoleWithGetPVC(ctx, ctx.Client, testNS, testRoleName)
				assertProviderServiceAccountsCondition(ctx.Cluster, corev1.ConditionTrue, "", "", "")
			})
		})
		Context("When invalid rolebinding exists", func() {
			BeforeEach(func() {
				initObjects = append(initObjects, getTestRoleBindingWithInvalidRoleRef(testNS, testRoleBindingName))
			})
			It("Should update rolebinding", func() {
				assertRoleBinding(ctx, ctx.Client, testNS, testRoleBindingName)
				assertProviderServiceAccountsCondition(ctx.Cluster, corev1.ConditionTrue, "", "", "")
			})
		})
	})
}

// Updates the service account secret similar to how a token controller would act upon a service account
// and then re-invokes reconcileNormal
func updateServiceAccountSecretAndReconcileNormal(ctx *builder.UnitTestContextForController) {
	assertServiceAccountAndUpdateSecret(ctx, ctx.Client, testNS, testSvcAccountName)
	Expect(ctx.ReconcileNormal()).Should(Succeed())
}
