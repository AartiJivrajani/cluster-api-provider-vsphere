// Copyright (c) 2019 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package builder

import (
	"context"
	"fmt"
	"path/filepath"

	uuid "github.com/satori/go.uuid"

	// nolint
	. "github.com/onsi/ginkgo"
	// nolint
	. "github.com/onsi/gomega"

	"github.com/go-logr/logr"
	"github.com/go-logr/logr/testing"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	vmwarev1 "sigs.k8s.io/cluster-api-provider-vsphere/apis/vmware/v1beta1"
	/*"gitlab.eng.vmware.com/core-build/guest-cluster-controller/controllers/util/status"
	"gitlab.eng.vmware.com/core-build/guest-cluster-controller/pkg/context/fake"
	"gitlab.eng.vmware.com/core-build/guest-cluster-controller/test/testutil"*/)

// IntegrationTestContext is used for integration testing
// GuestClusterControllers.
type IntegrationTestContext struct {
	context.Context
	Client                    client.Client
	GuestClient               client.Client
	Namespace                 string
	Cluster                   *clusterv1.Cluster
	ClusterKey                client.ObjectKey
	TanzuKubernetesCluster    *vmwarev1.VSphereCluster
	TanzuKubernetesClusterKey client.ObjectKey
	envTest                   *envtest.Environment
	suite                     *TestSuite
	PatchHelper               *patch.Helper
}

func (*IntegrationTestContext) GetLogger() logr.Logger {
	return testing.NullLogger{}
}

const (
	ControlPlaneStorageClassName  = "prototype-cp-storage-class"
	WorkerMachineStorageClassName = "prototype-worker-storage-class"
)

var boolTrue = true

// valid public certificate
/*var validTLSCertificate = []tkgv1.TLSCertificate{
	{
		Name: "test-cert-valid",
		Data: "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUVMekNDQXhlZ0F3SUJBZ0lVYWV2Q1Jha0hpbi96UVRNNVU3RGpjcDlRNzJRd0RRWUpLb1pJaHZjTkFRRUwKQlFBd2dhWXhDekFKQmdOVkJBWVRBbFZUTVJNd0VRWURWUVFJREFwWFlYTm9hVzVuZEc5dU1SQXdEZ1lEVlFRSApEQWRUWldGMGRHeGxNUTh3RFFZRFZRUUtEQVpXVFhkaGNtVXhEakFNQmdOVkJBc01CVTFCVUVKVk1Td3dLZ1lEClZRUUREQ05uWTIwek9EWTNMWEJ5YVhaaGRHVnlaV2RwYzNSeWVTNXRZMmhoYTJWeUxtTnZiVEVoTUI4R0NTcUcKU0liM0RRRUpBUllTYldOb1lXdGxja0IyYlhkaGNtVXVZMjl0TUI0WERUSXdNVEF5T1RJeE1USXlNbG9YRFRJeApNVEF5T1RJeE1USXlNbG93Z2FZeEN6QUpCZ05WQkFZVEFsVlRNUk13RVFZRFZRUUlEQXBYWVhOb2FXNW5kRzl1Ck1SQXdEZ1lEVlFRSERBZFRaV0YwZEd4bE1ROHdEUVlEVlFRS0RBWldUWGRoY21VeERqQU1CZ05WQkFzTUJVMUIKVUVKVk1Td3dLZ1lEVlFRRERDTm5ZMjB6T0RZM0xYQnlhWFpoZEdWeVpXZHBjM1J5ZVM1dFkyaGhhMlZ5TG1OdgpiVEVoTUI4R0NTcUdTSWIzRFFFSkFSWVNiV05vWVd0bGNrQjJiWGRoY21VdVkyOXRNSUlCSWpBTkJna3Foa2lHCjl3MEJBUUVGQUFPQ0FROEFNSUlCQ2dLQ0FRRUF2bDhIZEtab3o2eW5ZYVVkQzN5WklFR2UrQzZtZ2FxUmZvdVEKWFJtUVdYZllpeTJwNWtocExtc094TTdQNk02UEJkZDFJbytMMzRRZ0Z2Sk5ySW94bGlkOVFJNVJiMXBWMWhPUQphbHo4cUxGZmdSWHFLaXNRV3JRTmtpeldhMEVQT1Z3M3lubXo5VEd6QTdsQ3VtcGJiNnVnSXc4dlhQTkJZWElDCnZwemQ0dnliaktXQUxIMlVwYUFBV3NsUlVQWmZqMnFPbnJXMnJxdm9WT2I5R01CQUxKaEFXZys3andMZzhLU2QKR1BBMU1nalRRMXd2YnozNkxxUWJwRlNKa0ozazdnTXZoV09Sc0RMeHJRREZWU3plN2lZM0NvcVNzektacTFMQQpFeS94MEtHcTdWb0Q1QXlHU2NRSkZXR1IwUzlJaTRTV041ci9abUJYdGE0aWpRa2lad0lEQVFBQm8xTXdVVEFkCkJnTlZIUTRFRmdRVUFoVW1sWmhLSmNNbnplelNiTGVkNWc3dm9wSXdId1lEVlIwakJCZ3dGb0FVQWhVbWxaaEsKSmNNbnplelNiTGVkNWc3dm9wSXdEd1lEVlIwVEFRSC9CQVV3QXdFQi96QU5CZ2txaGtpRzl3MEJBUXNGQUFPQwpBUUVBSFNjVGxXc3MwUGNSOERVNVdPY1FINStuNmJOV3Q0WTg4OEpLMEFOMkpScUxObGdTZTBPNDlmbTN6bmlRCjVmSVk0TmFrYUhQUHdYeWZXNE9FdWhZYWhSdkxPME9GV1EvK1BsbXY5amQ1ZFEzdlNDMzExWDl1Z213NHRVYncKODRVTFFnaTEvRkpvS2tOblBvVFEzbDYwbkdPU2F5YVZvbld5QjRvTzdxK3R4NkhiK2o5UStVN2dTK2RIQXNsegpqUjh5WWhkd2kxT0ZuVHNhR0o5MW14Y1VsRGJuNkFIU1E3WXBvZlo1NjlGU2duTlhKb3hpYkljcnF2K1R4Z2F1CmJQcmluY0hxU1Jhc2dGQUhReE93WlRSQlRYRE1GZ213ZVBGeXp5MTZsa1dORTRjZzE1ai92TWZrZkxteUZMWHEKRFFjWnpmUDdoUStMYW9EeDNyeUE3S0prdkE9PQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==",
	},
}

var trustWithValidCertificate = &tkgv1.TrustConfiguration{
	AdditionalTrustedCAs: validTLSCertificate,
}*/

// AfterEach should be invoked by ginko.AfterEach to stop the guest cluster's
// API server.
func (ctx *IntegrationTestContext) AfterEach() {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: ctx.Namespace,
		},
	}
	By("Destroying integration test namespace")
	Expect(ctx.Client.Delete(ctx, namespace)).To(Succeed())

	if ctx.envTest != nil {
		By("Shutting down guest cluster control plane")
		Expect(ctx.envTest.Stop()).To(Succeed())
	}
}

// NewIntegrationTestContext should be invoked by ginkgo.BeforeEach
//
// This function creates a TanzuKubernetesCluster with a generated name, but stops
// short of generating a CAPI cluster so that it will work when the Tanzu Kubernetes
// Cluster controller is also deployed.
//
// This function returns a TestSuite context
// The resources created by this function may be cleaned up by calling AfterEach
// with the IntegrationTestContext returned by this function
func (s *TestSuite) NewIntegrationTestContext(integrationTestClient client.Client) *IntegrationTestContext {
	ctx := &IntegrationTestContext{
		Context: context.Background(),
		Client:  integrationTestClient,
		suite:   s,
		//StopManager:     s.stopManager,
		//StartNewManager: s.startNewManager,
	}

	By("Creating a temporary namespace", func() {
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: uuid.NewV4().String(),
			},
		}
		Expect(ctx.Client.Create(s, namespace)).To(Succeed())

		ctx.Namespace = namespace.Name
	})

	/*var virtualMachineClassName string
	By("Creating a prototype VirtualMachineClass", func() {
		virtualMachineClass := FakeVirtualMachineClass()
		Expect(ctx.Client.Create(s, virtualMachineClass)).To(Succeed())
		virtualMachineClassKey := client.ObjectKey{Name: virtualMachineClass.Name}
		Eventually(func() error {
			return ctx.Client.Get(s, virtualMachineClassKey, virtualMachineClass)
		}).Should(Succeed())

		virtualMachineClassName = virtualMachineClass.Name
	})*/

	/*By("Creating a prototype ResourceQuota", func() {
		rq := &corev1.ResourceQuota{
			TypeMeta: metav1.TypeMeta{
				Kind: "ResourceQuota",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "simple-rq-it",
				Namespace: ctx.Namespace,
			},
			Spec: corev1.ResourceQuotaSpec{
				Hard: corev1.ResourceList{
					"limits.cpu":    resource.MustParse("2"),
					"limits.memory": resource.MustParse("2Gi"),
				},
			},
		}
		rq.Spec.Hard[corev1.ResourceName(ControlPlaneStorageClassName+".storageclass.storage.k8s.io/requests.storage")] = resource.MustParse("1Gi")
		rq.Spec.Hard[corev1.ResourceName(WorkerMachineStorageClassName+".storageclass.storage.k8s.io/requests.storage")] = resource.MustParse("1Gi")
		Expect(ctx.Client.Create(s, rq)).To(Succeed())
		rqKey := client.ObjectKey{Namespace: rq.Namespace, Name: rq.Name}
		Eventually(func() error {
			return ctx.Client.Get(s, rqKey, rq)
		}).Should(Succeed())
	})

	By("Creating a prototype VirtualMachineImage", func() {
		virtualMachineImage := CreateFakeVirtualMachineImage()
		Expect(ctx.Client.Create(s, virtualMachineImage)).To(Succeed())
		tkr, tkrAddons := CreateFakeTKR(virtualMachineImage)
		Expect(PersistAddons(s, ctx.Client, tkrAddons)).To(Succeed())
		Expect(PersistTKR(s, ctx.Client, tkr)).To(Succeed())

		incompatibleVirtualMachineImage := CreateFakeIncompatibleVirtualMachineImage()
		Expect(ctx.Client.Create(s, incompatibleVirtualMachineImage)).To(Succeed())
		incompatibleTKR, incompatibleTKRAddons := CreateFakeIncompatibleTKR(incompatibleVirtualMachineImage)
		Expect(PersistAddons(s, ctx.Client, incompatibleTKRAddons)).To(Succeed())
		Expect(PersistTKR(s, ctx.Client, incompatibleTKR)).To(Succeed())
	})

	By("Creating a prototype vmoperator-network-config map", func() {
		cm := NewNetworkConfigMap()
		if err := ctx.Client.Create(ctx, &cm); err != nil {
			if !apierrors.IsAlreadyExists(err) {
				Expect(err).NotTo(HaveOccurred())
			}
		}
	})*/

	By("Create a tanzu kubernetes cluster and wait for it to exist", func() {
		ctx.TanzuKubernetesCluster = &vmwarev1.VSphereCluster{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ctx.Namespace,
				GenerateName: "test-",
			},
			Spec: vmwarev1.VSphereClusterSpec{},
		}
		Expect(ctx.Client.Create(s, ctx.TanzuKubernetesCluster)).To(Succeed())
		ctx.TanzuKubernetesClusterKey = client.ObjectKey{Namespace: ctx.TanzuKubernetesCluster.Namespace, Name: ctx.TanzuKubernetesCluster.Name}
		Eventually(func() error {
			return ctx.Client.Get(s, ctx.TanzuKubernetesClusterKey, ctx.TanzuKubernetesCluster)
		}).Should(Succeed())

		ph, err := patch.NewHelper(ctx.TanzuKubernetesCluster, ctx.Client)
		Expect(err).To(BeNil())
		ctx.PatchHelper = ph
	})

	By("Creating a extensions ca", func() {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ctx.TanzuKubernetesCluster.Name + "-extensions-ca",
				Namespace: ctx.Namespace,
			},
			Data: map[string][]byte{
				"ca.crt":  []byte("test-ca"),
				"tls.crt": []byte("test-tls.crt"),
				"tls.key": []byte("test-tls.key"),
			},
			Type: corev1.SecretTypeTLS,
		}
		Expect(ctx.Client.Create(s, secret)).To(Succeed())
		secretKey := client.ObjectKey{Namespace: secret.Namespace, Name: secret.Name}
		Eventually(func() error {
			return ctx.Client.Get(s, secretKey, secret)
		}).Should(Succeed())
	})
	return ctx
}

// NewIntegrationTestContextWithClusters should be invoked by ginkgo.BeforeEach.
//
// This function creates a TanzuKubernetesCluster with a generated name as well as a
// CAPI Cluster with the same name. The function also creates a test environment
// and starts its API server to serve as the control plane endpoint for the
// guest cluster.
//
// This function returns a TestSuite context.
//
// The resources created by this function may be cleaned up by calling AfterEach
// with the IntegrationTestContext returned by this function.
func (s *TestSuite) NewIntegrationTestContextWithClusters(integrationTestClient client.Client, simulateControlPlane, requiresPSP bool) *IntegrationTestContext {
	ctx := s.NewIntegrationTestContext(integrationTestClient)

	By("Create the CAPI Cluster and wait for it to exist", func() {
		cluster := &clusterv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ctx.TanzuKubernetesCluster.Namespace,
				Name:      ctx.TanzuKubernetesCluster.Name,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion:         ctx.TanzuKubernetesCluster.APIVersion,
						Kind:               ctx.TanzuKubernetesCluster.Kind,
						Name:               ctx.TanzuKubernetesCluster.Name,
						UID:                ctx.TanzuKubernetesCluster.UID,
						BlockOwnerDeletion: &boolTrue,
						Controller:         &boolTrue,
					},
				},
			},
			Spec: clusterv1.ClusterSpec{
				ClusterNetwork: &clusterv1.ClusterNetwork{
					Pods: &clusterv1.NetworkRanges{
						CIDRBlocks: []string{"1.0.0.0/16"},
					},
					Services: &clusterv1.NetworkRanges{
						CIDRBlocks: []string{"2.0.0.0/16"},
					},
				},
				InfrastructureRef: &corev1.ObjectReference{
					Name:      ctx.TanzuKubernetesCluster.Name,
					Namespace: ctx.TanzuKubernetesCluster.Namespace,
				},
			},
		}
		Expect(ctx.Client.Create(s, cluster)).To(Succeed())
		clusterKey := client.ObjectKey{Namespace: cluster.Namespace, Name: cluster.Name}
		Eventually(func() error {
			return ctx.Client.Get(s, clusterKey, cluster)
		}).Should(Succeed())

		ctx.Cluster = cluster
		ctx.ClusterKey = clusterKey
	})

	/*if requiresPSP {
		// Set PSP, but ensure that the other statuses aren't nil because Status.Addons
		// is initialized all in one go (see util/status/EnsureInitializedStatus)
		status.EnsureInitializedStatus(ctx.TanzuKubernetesCluster)
		addonStatus := status.FindAddonStatusByType(ctx.TanzuKubernetesCluster, tkgv1.PSP)
		addonStatus.SetStatus("defaultpsp", "")
		Expect(ctx.PatchHelper.Patch(ctx, ctx.TanzuKubernetesCluster)).To(Succeed())
	}*/

	if simulateControlPlane {
		var config *rest.Config
		By("Creating guest cluster control plane", func() {
			// Initialize a test environment to simulate the control plane of the
			// guest cluster.
			var err error
			ctx.envTest = &envtest.Environment{
				//KubeAPIServerFlags: append([]string{"--allow-privileged=true"}, envtest.DefaultKubeAPIServerFlags...),
				// Add some form of CRD so the CRD object is registered in the
				// scheme...
				CRDDirectoryPaths: []string{
					filepath.Join(s.flags.RootDir, "config", "supervisor", "crd"),
				},
			}
			ctx.envTest.ControlPlane.GetAPIServer().Configure().Set("allow-privileged", "true")
			config, err = ctx.envTest.Start()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(config).ShouldNot(BeNil())

			ctx.GuestClient, err = client.New(config, client.Options{})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(ctx.GuestClient).ShouldNot(BeNil())

			// Preload guest cluster client with addons that kubeadm would have deployed
			/*Expect(ctx.GuestClient.Create(ctx, fake.GetPrototypeCoreDNSDeployment())).To(Succeed())
			Expect(ctx.GuestClient.Create(ctx, fake.GetPrototypeKubeProxyDaemonSet())).To(Succeed())
			Expect(ctx.GuestClient.Create(ctx, fake.GetPrototypeKubeadmConfigMap())).To(Succeed())*/
		})

		By("Create the kubeconfig secret for the cluster", func() {
			buf, err := WriteKubeConfig(config)
			Expect(err).ToNot(HaveOccurred())
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: ctx.Cluster.Namespace,
					Name:      fmt.Sprintf("%s-kubeconfig", ctx.Cluster.Name),
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: ctx.Cluster.APIVersion,
							Kind:       ctx.Cluster.Kind,
							Name:       ctx.Cluster.Name,
							UID:        ctx.Cluster.UID,
						},
					},
				},
				Data: map[string][]byte{
					"value": buf,
				},
			}
			Expect(integrationTestClient.Create(s, secret)).To(Succeed())
			secretKey := client.ObjectKey{Namespace: secret.Namespace, Name: secret.Name}
			Eventually(func() error {
				return s.integrationTestClient.Get(s, secretKey, secret)
			}).Should(Succeed())
		})
	}

	return ctx
}

// WriteKubeConfig writes an existing *rest.Config out as the typical
// KubeConfig YAML data.
func WriteKubeConfig(config *rest.Config) ([]byte, error) {
	return clientcmd.Write(api.Config{
		Clusters: map[string]*api.Cluster{
			config.ServerName: {
				Server:                   config.Host,
				CertificateAuthorityData: config.CAData,
			},
		},
		Contexts: map[string]*api.Context{
			config.ServerName: {
				Cluster:  config.ServerName,
				AuthInfo: config.Username,
			},
		},
		AuthInfos: map[string]*api.AuthInfo{
			config.Username: {
				ClientKeyData:         config.KeyData,
				ClientCertificateData: config.CertData,
			},
		},
		CurrentContext: config.ServerName,
	})
}
