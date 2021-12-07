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

	//nolint
	. "github.com/onsi/ginkgo"

	//nolint
	. "github.com/onsi/gomega"

	vmwarev1 "sigs.k8s.io/cluster-api-provider-vsphere/apis/vmware/v1beta1"
	"sigs.k8s.io/cluster-api-provider-vsphere/pkg/builder"
)

var _ = Describe("ServiceAccount controller integration tests", func() {
	var (
		ctx *builder.IntegrationTestContext
	)

	BeforeEach(func() {
		serviceAccountProviderTestsuite.SetIntegrationTestClient(testEnv.Client)
		ctx = serviceAccountProviderTestsuite.NewIntegrationTestContextWithClusters(testEnv.Client, true, false)
		testSystemSvcAcctCM := "test-system-svc-acct-cm"
		cfgMap := getSystemServiceAccountsConfigMap(ctx.TanzuKubernetesCluster.Namespace, testSystemSvcAcctCM)
		Expect(ctx.Client.Create(ctx, cfgMap)).To(Succeed())
		_ = os.Setenv("SERVICE_ACCOUNTS_CM_NAMESPACE", ctx.TanzuKubernetesCluster.Namespace)
		_ = os.Setenv("SERVICE_ACCOUNTS_CM_NAME", testSystemSvcAcctCM)
	})

	AfterEach(func() {
		ctx.AfterEach()
		ctx = nil
	})

	FDescribe("When the ProviderServiceAccount is created", func() {
		var (
			pSvcAccount *vmwarev1.ProviderServiceAccount
		)
		BeforeEach(func() {
			pSvcAccount = getTestProviderServiceAccount(ctx.Namespace, testProviderSvcAccountName, ctx.TanzuKubernetesCluster)
			createTestResource(ctx, ctx.Client, pSvcAccount)
		})
		AfterEach(func() {
			// Deleting the provider service account is not strictly required as the context itself gets teared down but
			// keeping it for clarity.
			deleteTestResource(ctx, ctx.Client, pSvcAccount)
		})

		Context("When serviceaccount secret is created", func() {
			BeforeEach(func() {
				// Note: Envtest doesn't run controller-manager, hence, the token controller. The token controller is required
				// to create a secret containing the bearer token, cert etc for a service account. We need to
				// simulate the job of the token controller by waiting for the service account creation and then updating it
				// with a prototype secret.
				assertServiceAccountAndUpdateSecret(ctx, ctx.Client, ctx.Namespace, testSvcAccountName)
			})
			It("Should reconcile", func() {
				By("Creating the target secret in the target namespace")
				assertTargetSecret(ctx, ctx.GuestClient, testTargetNS, testTargetSecret)
			})
		})

		Context("When serviceaccount secret is rotated", func() {
			BeforeEach(func() {
				createTargetSecretWithInvalidToken(ctx, ctx.GuestClient)
				assertServiceAccountAndUpdateSecret(ctx, ctx.Client, ctx.Namespace, testSvcAccountName)
			})
			It("Should reconcile", func() {
				By("Updating the target secret in the target namespace")
				assertTargetSecret(ctx, ctx.GuestClient, testTargetNS, testTargetSecret)
			})
		})
	})

	Context("When provider service account does not exist", func() {
		It("Should not reconcile", func() {
			By("Not creating any entities")
			assertNoEntities(ctx, ctx.Client, ctx.Namespace)
		})
	})

	Context("When the ProviderServiceAccount has a non-existing cluster ref", func() {
		BeforeEach(func() {
			nonExistingTanzuKubernetesCluster := ctx.TanzuKubernetesCluster
			nonExistingTanzuKubernetesCluster.Name = "non-existing-managed-ckuster"
			pSvcAccount := getTestProviderServiceAccount(ctx.Namespace, testProviderSvcAccountName, nonExistingTanzuKubernetesCluster)
			createTestResource(ctx, ctx.Client, pSvcAccount)
		})
		It("Should not reconcile", func() {
			By("Not creating any entities")
			assertNoEntities(ctx, ctx.Client, ctx.Namespace)
		})
	})

	Context("When the ProviderServiceAccount has invalid cluster ref", func() {
		BeforeEach(func() {
			createTestProviderSvcAccountWithInvalidRef(ctx, ctx.Client, ctx.Namespace, ctx.TanzuKubernetesCluster)
		})
		It("Should not reconcile", func() {
			By("Not creating any entities")
			assertNoEntities(ctx, ctx.Client, ctx.Namespace)
		})
	})
})
