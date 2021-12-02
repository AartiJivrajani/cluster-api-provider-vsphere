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

package fake

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	//apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"

	apiregv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	kubeadmv1 "sigs.k8s.io/cluster-api/bootstrap/kubeadm/types/upstreamv1beta3"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"

	vmwarecontext "sigs.k8s.io/cluster-api-provider-vsphere/pkg/context/vmware"
)

type GClusterContext struct {
	*vmwarecontext.ClusterContext

	// GuestClient can be used to access the guest cluster.
	GClient client.Client
}

// GetPrototypeCoreDNSDeployment gets a new Deployment for CoreDNS
// this is used to simulate the addon that kubeadm will pre-load the
// cluster with
func GetPrototypeCoreDNSDeployment() *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "coredns",
			Namespace: metav1.NamespaceSystem,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"k8s-app": "kube-dns",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"k8s-app": "kube-dns",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "coredns",
							Image: "blah",
						},
					},
				},
			},
		},
	}
}

// GetPrototypeKubeadmConfigMap gets a new ConfigMap that simulates the kubeadm
// ConfigMap that Kubeadm creates containing Cluster configuration
func GetPrototypeKubeadmConfigMap() *corev1.ConfigMap {
	clusterConfig := kubeadmv1.ClusterConfiguration{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterConfiguration",
			APIVersion: kubeadmv1.GroupVersion.Version,
		},
		DNS: kubeadmv1.DNS{
			//Type: kubeadmv1.CoreDNS,
			ImageMeta: kubeadmv1.ImageMeta{
				ImageRepository: "blahblahblah.com",
				ImageTag:        "my cool image!!", // Guaranteed to fail parsing in a real KCP
			},
		},
	}

	rawClusterConfig, err := yaml.Marshal(clusterConfig)
	if err != nil {
		panic("An error occurred marshaling the dummy kubeadm ConfigMap")
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubeadm-config",
			Namespace: metav1.NamespaceSystem,
		},
		Data: map[string]string{
			"ClusterConfiguration": string(rawClusterConfig),
		},
	}
}

// GetPrototypeKubeProxyDaemonSet gets a new DaemonSet for kube-proxy
// this is used to simulate the addon that kubeadm will pre-load the
// cluster with
func GetPrototypeKubeProxyDaemonSet() *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kube-proxy",
			Namespace: metav1.NamespaceSystem,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"k8s-app": "kube-proxy",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"k8s-app": "kube-proxy",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "kube-proxy",
							Image: "blah",
						},
					},
				},
			},
		},
	}
}

// NewGuestClusterContext returns a fake GuestClusterContext for unit testing
// guest cluster controllers with a fake client.
func NewGClusterContext(ctx vmwarecontext.ClusterContext, prototypeCluster bool, gcInitObjects ...runtime.Object) *GClusterContext {

	if prototypeCluster {
		cluster := newCluster(ctx.VSphereCluster)

		if err := ctx.Client.Create(ctx, cluster); err != nil {
			panic(err)
		}
	}

	// Add prototype addons that kubeadm would have generated for us
	gcInitObjects = append(gcInitObjects, GetPrototypeKubeadmConfigMap())
	gcInitObjects = append(gcInitObjects, GetPrototypeCoreDNSDeployment())
	gcInitObjects = append(gcInitObjects, GetPrototypeKubeProxyDaemonSet())

	return &GClusterContext{
		ClusterContext: &ctx,
		GClient:        NewFakeGuestClusterClient(gcInitObjects...),
	}
}

func NewFakeGuestClusterClient(initObjects ...runtime.Object) client.Client {
	newScheme := scheme.Scheme
	_ = apiregv1.AddToScheme(newScheme)
	// TODO: [Aarti] - not sure why this is throwing an error with the parameter type. Solve and uncomment
	// _ = apiextv1.AddToScheme(newScheme)

	fake.NewFakeClient(initObjects...)
	return fake.NewFakeClientWithScheme(newScheme, initObjects...)
}
