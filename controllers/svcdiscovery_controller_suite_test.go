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
	"context"
	"strconv"
	"strings"
	"testing"

	// nolint
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	bootstrapapi "k8s.io/cluster-bootstrap/token/api"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/controller-runtime/pkg/client"

	vmwarev1beta1 "sigs.k8s.io/cluster-api-provider-vsphere/apis/vmware/v1beta1"
	"sigs.k8s.io/cluster-api-provider-vsphere/test/builder"
)

// serviceAccountProviderTestsuite is used for unit and integration testing this controller.
var serviceDiscoveryTestSuite = builder.NewTestSuiteForController(AddServiceDiscoveryControllerToManager, newServiceDiscoveryReconciler)

// TODO: add back integration tests
func TestServiceDiscoveryController(t *testing.T) {
	serviceDiscoveryTestSuite.Register(t, "Service Discovery controller serviceAccountProviderTestsuite", serviceDiscoveryUnitTests)
}

const (
	testSupervisorAPIServerVIP         = "10.0.0.100"
	testSupervisorAPIServerVIPHostName = "vip.example.com"
	testSupervisorAPIServerFIP         = "192.168.1.100"
	testSupervisorAPIServerFIPHostName = "fip.example.com"
	testSupervisorAPIServerPort        = 6443

	supervisorHeadlessSvcPort = 6443
)

func assertEventuallyDoesNotExistInNamespace(ctx context.Context, guestClient client.Client, namespace, name string, obj client.Object) {
	EventuallyWithOffset(4, func() error {
		key := client.ObjectKey{Namespace: namespace, Name: name}
		return guestClient.Get(ctx, key, obj)
	}).ShouldNot(Succeed())
}

func assertHeadlessSvc(ctx context.Context, guestClient client.Client, namespace, name string) {
	headlessSvc := &corev1.Service{}
	EventuallyWithOffset(4, func() error {
		key := client.ObjectKey{Namespace: namespace, Name: name}
		return guestClient.Get(ctx, key, headlessSvc)
	}).Should(Succeed())
	Expect(headlessSvc.Spec.Ports[0].Port).To(Equal(int32(supervisorHeadlessSvcPort)))
	Expect(headlessSvc.Spec.Ports[0].TargetPort.IntVal).To(Equal(int32(supervisorAPIServerPort)))
}

// nolint
func assertHeadlessSvcWithNoEndpoints(ctx context.Context, guestClient client.Client, namespace, name string) {
	assertHeadlessSvc(ctx, guestClient, namespace, name)
	headlessEndpoints := &corev1.Endpoints{}
	assertEventuallyDoesNotExistInNamespace(ctx, guestClient, namespace, name, headlessEndpoints)
}

func assertHeadlessSvcWithVIPEndpoints(ctx context.Context, guestClient client.Client, namespace, name string) {
	assertHeadlessSvc(ctx, guestClient, namespace, name)
	headlessEndpoints := &corev1.Endpoints{}
	assertEventuallyExistsInNamespace(ctx, guestClient, namespace, name, headlessEndpoints)
	Expect(headlessEndpoints.Subsets[0].Addresses[0].IP).To(Equal(testSupervisorAPIServerVIP))
	Expect(headlessEndpoints.Subsets[0].Ports[0].Port).To(Equal(int32(supervisorAPIServerPort)))
}

func assertHeadlessSvcWithVIPHostnameEndpoints(ctx context.Context, guestClient client.Client, namespace, name string) {
	assertHeadlessSvc(ctx, guestClient, namespace, name)
	headlessEndpoints := &corev1.Endpoints{}
	assertEventuallyExistsInNamespace(ctx, guestClient, namespace, name, headlessEndpoints)
	Expect(headlessEndpoints.Subsets[0].Addresses[0].Hostname).To(Equal(testSupervisorAPIServerVIPHostName))
	Expect(headlessEndpoints.Subsets[0].Ports[0].Port).To(Equal(int32(supervisorAPIServerPort)))
}

func assertHeadlessSvcWithFIPEndpoints(ctx context.Context, guestClient client.Client, namespace, name string) {
	assertHeadlessSvc(ctx, guestClient, namespace, name)
	headlessEndpoints := &corev1.Endpoints{}
	assertEventuallyExistsInNamespace(ctx, guestClient, namespace, name, headlessEndpoints)
	Expect(headlessEndpoints.Subsets[0].Addresses[0].IP).To(Equal(testSupervisorAPIServerFIP))
	Expect(headlessEndpoints.Subsets[0].Ports[0].Port).To(Equal(int32(testSupervisorAPIServerPort)))
}

func assertHeadlessSvcWithFIPHostNameEndpoints(ctx context.Context, guestClient client.Client, namespace, name string) {
	assertHeadlessSvc(ctx, guestClient, namespace, name)
	headlessEndpoints := &corev1.Endpoints{}
	assertEventuallyExistsInNamespace(ctx, guestClient, namespace, name, headlessEndpoints)
	Expect(headlessEndpoints.Subsets[0].Addresses[0].Hostname).To(Equal(testSupervisorAPIServerFIPHostName))
	Expect(headlessEndpoints.Subsets[0].Ports[0].Port).To(Equal(int32(testSupervisorAPIServerPort)))
}

func assertServiceDiscoveryCondition(vsphereCluster *vmwarev1beta1.VSphereCluster, status corev1.ConditionStatus,
	message string, reason string, severity clusterv1.ConditionSeverity) {
	c := conditions.Get(vsphereCluster, vmwarev1beta1.ServiceDiscoveryReadyCondition)
	Expect(c).NotTo(BeNil())
	if message == "" {
		Expect(c.Message).To(BeEmpty())
	} else {
		Expect(strings.Contains(c.Message, message)).To(BeTrue(), "expect condition message contains: %s, actual: %s", message, c.Message)
	}
	Expect(c.Status).To(Equal(status))
	Expect(c.Reason).To(Equal(reason))
	Expect(c.Severity).To(Equal(severity))
}

func newTestSupervisorLBServiceWithIPStatus() *corev1.Service {
	svc := newTestSupervisorLBService()
	svc.Status = corev1.ServiceStatus{
		LoadBalancer: corev1.LoadBalancerStatus{
			Ingress: []corev1.LoadBalancerIngress{
				{
					IP: testSupervisorAPIServerVIP,
				},
			},
		},
	}
	return svc
}

func newTestSupervisorLBServiceWithHostnameStatus() *corev1.Service {
	svc := newTestSupervisorLBService()
	svc.Status = corev1.ServiceStatus{
		LoadBalancer: corev1.LoadBalancerStatus{
			Ingress: []corev1.LoadBalancerIngress{
				{
					Hostname: testSupervisorAPIServerVIPHostName,
				},
			},
		},
	}
	return svc
}

func newTestSupervisorLBService() *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      supervisorLoadBalancerSvcName,
			Namespace: supervisorLoadBalancerSvcNamespace,
		},
		Spec: corev1.ServiceSpec{
			// Note: This will be service with no selectors. The endpoints will be manually created.
			Ports: []corev1.ServicePort{
				{
					Name:       "kube-apiserver",
					Port:       6443,
					TargetPort: intstr.FromInt(6443),
				},
			},
		},
	}
}

func newTestConfigMapWithHost(serverHost string) *corev1.ConfigMap {
	testKubeconfigData := `apiVersion: v1
clusters:
- cluster:
    server: https://` + serverHost + ":" + strconv.Itoa(testSupervisorAPIServerPort) + `
  name: ""
contexts: []
current-context: ""
kind: Config
preferences: {}
users: []
`
	return newTestConfigMapWithData(
		map[string]string{
			bootstrapapi.KubeConfigKey: testKubeconfigData,
		})
}

func newTestConfigMapWithData(data map[string]string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      bootstrapapi.ConfigMapClusterInfo,
			Namespace: metav1.NamespacePublic,
		},
		Data: data,
	}
}
