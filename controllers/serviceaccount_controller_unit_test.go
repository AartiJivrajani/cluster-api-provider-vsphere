/*
Copyright 2021 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"os"
	"sigs.k8s.io/cluster-api-provider-vsphere/test/builder"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	vmwarev1 "sigs.k8s.io/cluster-api-provider-vsphere/apis/vmware/v1beta1"
	_ "sigs.k8s.io/cluster-api-provider-vsphere/pkg/context"
	"sigs.k8s.io/cluster-api-provider-vsphere/pkg/context/fake"

)

func unitTests() {
	Describe("Invoking ReconcileNormal", unitTestsReconcileNormal)
}

func unitTestsReconcileNormal() {
	var (
		ctx            *builder.UnitTestContextForController
		vsphereCluster *vmwarev1.VSphereCluster
		initObjects    []runtime.Object
	)

	JustBeforeEach(func() {
		// Note: The provider service account requires a reference to the tanzukubernetescluster hence the need to create
		// a fake tanzu kubernetes cluster in the test and pass it to during context setup.

		ctx = suite.NewUnitTestContextForControllerWithTanzuKubernetesCluster(vsphereCluster, false, initObjects...)
	})
	AfterEach(func() {
		ctx = nil
	})

	Context("When no provider service account is available", func() {
		It("Should reconcile", func() {
			By("Not creating any entities")
			assertNoEntities(ctx, ctx.Client, testNS)
			assertProviderServiceAccountsCondition(ctx.VSphereCluster, corev1.ConditionTrue, "", "", "")
		})
	})

	Describe("When the ProviderServiceAccount is created", func() {
		BeforeEach(func() {
			obj := fake.NewTanzuKubernetesCluster(3, 3)
			vsphereCluster = &obj
			vsphereCluster.Namespace = testNS
			_ = os.Setenv("SERVICE_ACCOUNTS_CM_NAMESPACE", testSystemSvcAcctNs)
			_ = os.Setenv("SERVICE_ACCOUNTS_CM_NAME", testSystemSvcAcctCM)
			initObjects = []runtime.Object{
				getSystemServiceAccountsConfigMap(testSystemSvcAcctNs, testSystemSvcAcctCM),
				getTestProviderServiceAccount(testNS, testProviderSvcAccountName, vsphereCluster),
			}
		})
		Context("When serviceaccount secret is created", func() {
			It("Should reconcile", func() {
				assertTargetNamespace(ctx, ctx.GClient, testTargetNS, false)
				updateServiceAccountSecretAndReconcileNormal(ctx)
				assertTargetNamespace(ctx, ctx.GClient, testTargetNS, true)
				By("Creating the target secret in the target namespace")
				assertTargetSecret(ctx, ctx.GClient, testTargetNS, testTargetSecret)
				assertProviderServiceAccountsCondition(ctx.VSphereCluster, corev1.ConditionTrue, "", "", "")
			})
		})
		Context("When serviceaccount secret is modified", func() {
			It("Should reconcile", func() {
				// This is to simulate an outdate token that will be replaced when the serviceaccount secret is created.
				createTargetSecretWithInvalidToken(ctx, ctx.GClient)
				updateServiceAccountSecretAndReconcileNormal(ctx)
				By("Updating the target secret in the target namespace")
				assertTargetSecret(ctx, ctx.GClient, testTargetNS, testTargetSecret)
				assertProviderServiceAccountsCondition(ctx.VSphereCluster, corev1.ConditionTrue, "", "", "")
			})
		})
		Context("When invalid role exists", func() {
			BeforeEach(func() {
				initObjects = append(initObjects, getTestRoleWithGetPod(testNS, testRoleName))
			})
			It("Should update role", func() {
				assertRoleWithGetPVC(ctx, ctx.Client, testNS, testRoleName)
				assertProviderServiceAccountsCondition(ctx.VSphereCluster, corev1.ConditionTrue, "", "", "")
			})
		})
		Context("When invalid rolebinding exists", func() {
			BeforeEach(func() {
				initObjects = append(initObjects, getTestRoleBindingWithInvalidRoleRef(testNS, testRoleBindingName))
			})
			It("Should update rolebinding", func() {
				assertRoleBinding(ctx, ctx.Client, testNS, testRoleBindingName)
				assertProviderServiceAccountsCondition(ctx.VSphereCluster, corev1.ConditionTrue, "", "", "")
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
