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
	goctx "context"
	"os"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	//nolint
	. "github.com/onsi/ginkgo"

	//nolint
	. "github.com/onsi/gomega"

	vmwarev1 "sigs.k8s.io/cluster-api-provider-vsphere/apis/vmware/v1beta1"
	"sigs.k8s.io/cluster-api-provider-vsphere/pkg/builder"
)

var _ = Describe("ServiceAccount controller integration tests", func() {
	var (
		intCtx *builder.IntegrationTestContext
	)

	BeforeEach(func() {
		ServiceAccountProviderTestsuite.SetIntegrationTestClient(testEnv.Manager.GetClient())
		intCtx = ServiceAccountProviderTestsuite.NewIntegrationTestContextWithClusters(goctx.Background(), testEnv.Manager.GetClient(), true)
		testSystemSvcAcctCM := "test-system-svc-acct-cm"
		cfgMap := getSystemServiceAccountsConfigMap(intCtx.VSphereCluster.Namespace, testSystemSvcAcctCM)
		Expect(intCtx.Client.Create(intCtx, cfgMap)).To(Succeed())
		_ = os.Setenv("SERVICE_ACCOUNTS_CM_NAMESPACE", intCtx.VSphereCluster.Namespace)
		_ = os.Setenv("SERVICE_ACCOUNTS_CM_NAME", testSystemSvcAcctCM)
	})

	AfterEach(func() {
		intCtx.AfterEach()
		intCtx = nil
	})

	Describe("When the ProviderServiceAccount is created", func() {
		var (
			pSvcAccount *vmwarev1.ProviderServiceAccount
			targetNSObj *corev1.Namespace
		)
		BeforeEach(func() {
			pSvcAccount = getTestProviderServiceAccount(intCtx.Namespace, testProviderSvcAccountName, intCtx.VSphereCluster)
			createTestResource(intCtx, intCtx.Client, pSvcAccount)
			assertEventuallyExistsInNamespace(intCtx, intCtx.Client, intCtx.Namespace, testProviderSvcAccountName, pSvcAccount)
		})
		AfterEach(func() {
			// Deleting the provider service account is not strictly required as the context itself gets teared down but
			// keeping it for clarity.
			deleteTestResource(intCtx, intCtx.Client, pSvcAccount)
		})

		Context("When serviceaccount secret is created", func() {
			BeforeEach(func() {
				// Note: Envtest doesn't run controller-manager, hence, the token controller. The token controller is required
				// to create a secret containing the bearer token, cert etc for a service account. We need to
				// simulate the job of the token controller by waiting for the service account creation and then updating it
				// with a prototype secret.
				//log.Printf("ctx.Client to get ************** %v", intCtx.Client)
				assertServiceAccountAndUpdateSecret(intCtx, intCtx.Client, intCtx.Namespace, testProviderSvcAccountName)
			})
			It("Should reconcile", func() {
				By("Creating the target secret in the target namespace")
				assertTargetSecret(intCtx, intCtx.GuestClient, testTargetNS, testTargetSecret)
			})
		})

		Context("When serviceaccount secret is rotated", func() {
			BeforeEach(func() {
				targetNSObj = &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: testTargetNS,
					},
				}
				Expect(intCtx.GuestClient.Create(intCtx, targetNSObj)).To(Succeed())
				createTargetSecretWithInvalidToken(intCtx, intCtx.GuestClient, testTargetNS)
				assertServiceAccountAndUpdateSecret(intCtx, intCtx.Client, intCtx.Namespace, testSvcAccountName)
			})
			AfterEach(func() {
				deleteTestResource(intCtx, intCtx.GuestClient, targetNSObj)
			})
			It("Should reconcile", func() {
				By("Updating the target secret in the target namespace")
				assertTargetSecret(intCtx, intCtx.GuestClient, testTargetNS, testTargetSecret)
			})
		})

		Context("When provider service account does not exist", func() {
			It("Should not reconcile", func() {
				By("Not creating any entities")
				assertNoEntities(intCtx, intCtx.Client, intCtx.Namespace)
			})
		})

		Context("When the ProviderServiceAccount has a non-existing cluster ref", func() {
			BeforeEach(func() {
				nonExistingTanzuKubernetesCluster := intCtx.VSphereCluster
				nonExistingTanzuKubernetesCluster.Name = "non-existing-managed-ckuster"
				pSvcAccountNew := getTestProviderServiceAccount(intCtx.Namespace, testProviderSvcAccountName, nonExistingTanzuKubernetesCluster)
				deleteTestResource(intCtx, intCtx.Client, pSvcAccount)
				createTestResource(intCtx, intCtx.Client, pSvcAccountNew)
			})
			It("Should not reconcile", func() {
				By("Not creating any entities")
				assertNoEntities(intCtx, intCtx.Client, intCtx.Namespace)
			})
		})
		//
		//Context("When the ProviderServiceAccount has invalid cluster ref", func() {
		//	BeforeEach(func() {
		//		createTestProviderSvcAccountWithInvalidRef(ctx, ctx.Client, ctx.Namespace, ctx.VSphereCluster)
		//	})
		//	It("Should not reconcile", func() {
		//		By("Not creating any entities")
		//		assertNoEntities(ctx, ctx.Client, ctx.Namespace)
		//	})
		//})
	})
})
