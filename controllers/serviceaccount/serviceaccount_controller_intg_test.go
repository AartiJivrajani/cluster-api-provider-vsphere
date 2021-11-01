// Copyright (c) 2019 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package serviceaccount_test

import (
	"os"

	//nolint
	. "github.com/onsi/ginkgo"

	//nolint
	. "github.com/onsi/gomega"

	tkgv1 "gitlab.eng.vmware.com/core-build/guest-cluster-controller/apis/run.tanzu/v1alpha2"
	"gitlab.eng.vmware.com/core-build/guest-cluster-controller/test/builder"
)

func intgTests() {
	var (
		ctx *builder.IntegrationTestContext
	)

	BeforeEach(func() {
		ctx = suite.NewIntegrationTestContextWithClusters(true, false)
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

	Describe("When the ProviderServiceAccount is created", func() {
		var (
			pSvcAccount *tkgv1.ProviderServiceAccount
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
}
