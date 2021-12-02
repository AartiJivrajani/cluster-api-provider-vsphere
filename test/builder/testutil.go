package builder

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"os/exec"
	"strings"
	"time"

	vmoprv1 "github.com/vmware-tanzu/vm-operator-api/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/cluster-api-provider-vsphere/controllers/vmware/addons"
	"sigs.k8s.io/cluster-api-provider-vsphere/pkg/manifests"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/klog/v2"
)

// FindModuleDir returns the on-disk directory for the provided Go module.
func FindModuleDir(module string) string {
	cmd := exec.Command("go", "mod", "download", "-json", module)
	out, err := cmd.Output()
	if err != nil {
		klog.Fatalf("Failed to run go mod to find module %q directory", module)
	}
	info := struct{ Dir string }{}
	if err := json.Unmarshal(out, &info); err != nil {
		klog.Fatalf("Failed to unmarshal output from go mod command: %v", err)
	} else if info.Dir == "" {
		klog.Fatalf("Failed to find go module %q directory, received %v", module, string(out))
	}
	return info.Dir
}

const (
	minTCPPort         = 0
	maxTCPPort         = 65535
	maxReservedTCPPort = 1024
	maxRandTCPPort     = maxTCPPort - (maxReservedTCPPort + 1)
)

func randomTCPPort() int {
	tcpPortRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := maxReservedTCPPort; i < maxTCPPort; i++ {
		p := tcpPortRand.Intn(maxRandTCPPort) + maxReservedTCPPort + 1
		if isTCPPortAvailable(p) {
			return p
		}
	}
	return -1
}

// isTCPPortAvailable returns a flag indicating whether or not a TCP port is
// available.
func isTCPPortAvailable(port int) bool {
	if port < minTCPPort || port > maxTCPPort {
		return false
	}
	conn, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func FakeVirtualMachineClass() *vmoprv1.VirtualMachineClass {
	return &vmoprv1.VirtualMachineClass{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-",
		},
		Spec: vmoprv1.VirtualMachineClassSpec{
			Hardware: vmoprv1.VirtualMachineClassHardware{
				Cpus:   int64(2),
				Memory: resource.MustParse("4Gi"),
			},
			Policies: vmoprv1.VirtualMachineClassPolicies{
				Resources: vmoprv1.VirtualMachineClassResources{
					Requests: vmoprv1.VirtualMachineResourceSpec{
						Cpu:    resource.MustParse("2Gi"),
						Memory: resource.MustParse("4Gi"),
					},
					Limits: vmoprv1.VirtualMachineResourceSpec{
						Cpu:    resource.MustParse("2Gi"),
						Memory: resource.MustParse("4Gi"),
					},
				},
			},
		},
	}
}

const (
	// FakeDistributionVersion must be in semver format to pass validation
	FakeDistributionVersion  = "v1.15.5+vmware.1.66-guest.1.821"
	FakeKubernetesVersion    = "v1.15.5+vmware.1"
	FakeEtcdVersion          = "v3.3.10_vmware.1"
	FakeDNSVersion           = "v1.3.1_vmware.1"
	FakeCNIVersion           = "v3.1.1"
	FakeCSIVersion           = "v3.1.1"
	FakeCPIVersion           = "v3.1.1"
	FakeAuthServiceVersion   = "v3.1.1"
	FakeMetricsServerVersion = "v0.4.1"
)

func CreateFakeVirtualMachineImage() *vmoprv1.VirtualMachineImage {
	image := FakeVirtualMachineImage(FakeDistributionVersion, FakeKubernetesVersion, FakeEtcdVersion, FakeDNSVersion, FakeCNIVersion, FakeCSIVersion, FakeCPIVersion, FakeAuthServiceVersion, FakeMetricsServerVersion)
	// Using image.GenerateName causes problems with unit tests
	// This doesn't need to be a secure prng
	// nolint:gosec
	image.Name = fmt.Sprintf("test-%d", rand.Int())
	return image
}

func FakeVirtualMachineImage(distroVersion, kubernetesVersion, etcdVersion, coreDNSVersion, cniVersion, csiVersion, cpiVersion, authsvcVersion, metricsServerVersion string) *vmoprv1.VirtualMachineImage {
	return fakeVirtualMachineImage(distroVersion, kubernetesVersion, etcdVersion, coreDNSVersion, cniVersion, csiVersion, cpiVersion, authsvcVersion, metricsServerVersion, true, true)
}

func fakeVirtualMachineImage(distroVersion, kubernetesVersion, etcdVersion, coreDNSVersion, cniVersion, csiVersion, cpiVersion, authsvcVersion, metricsServerVersion string, compatible, antreaEnabled bool) *vmoprv1.VirtualMachineImage {
	distro := manifests.VirtualMachineDistributionSpec{
		Distribution: manifests.DistributionVersion{
			Version: distroVersion,
		},
		Kubernetes: manifests.ImageVersion{
			ImageRepository: "vmware",
			Version:         kubernetesVersion,
		},
		Etcd: manifests.ImageVersion{
			ImageRepository: "vmware",
			Version:         etcdVersion,
		},
		CoreDNS: manifests.ImageVersion{
			ImageRepository: "vmware",
			Version:         coreDNSVersion,
		},
	}

	cniCalicoAddOnAnnotation := addons.AddOnsAnnotationPrefix + string(addons.Calico)
	csiAddOnAnnotation := addons.AddOnsAnnotationPrefix + string(addons.VMwareGuestClusterCSI)
	cpAddOnAnnotation := addons.AddOnsAnnotationPrefix + string(addons.VMwareGuestClusterCPI)
	authsvcAddOnAnnotation := addons.AddOnsAnnotationPrefix + string(addons.VMwareGuestClusterAuthsvc)
	metricsServerAddOnAnnotation := addons.AddOnsAnnotationPrefix + string(addons.VMwareGuestClusterMetricsServer)

	calicoTemplate := addVersionToCalico(cniVersion)
	csiTemplate := addVersionToCSI(csiVersion)
	cpiTemplate := addVersionToCPI(cpiVersion)
	authsvcTemplate := addVersionToAuthSvc(authsvcVersion)
	metricsServerTemplate := addVersionToMetricsServer(metricsServerVersion)

	calicoAddOn := addons.AddOnSpec{
		Name:    string(addons.Calico),
		Type:    "inline",
		Version: cniVersion,
		Value:   calicoTemplate,
	}

	csiAddOn := addons.AddOnSpec{
		Name:    string(addons.VMwareGuestClusterCSI),
		Type:    "inline",
		Version: csiVersion,
		Value:   csiTemplate,
	}

	cpAddOn := addons.AddOnSpec{
		Name:    string(addons.VMwareGuestClusterCPI),
		Type:    "inline",
		Version: cpiVersion,
		Value:   cpiTemplate,
	}

	asAddOn := addons.AddOnSpec{
		Name:    string(addons.VMwareGuestClusterAuthsvc),
		Type:    "inline",
		Version: authsvcVersion,
		Value:   authsvcTemplate,
	}

	metricsServerAddOn := addons.AddOnSpec{
		Name:    string(addons.VMwareGuestClusterMetricsServer),
		Type:    "inline",
		Version: metricsServerVersion,
		Value:   metricsServerTemplate,
	}

	rawJSON, err := json.Marshal(distro)
	if err != nil {
		panic(err)
	}

	rawCniCalicoJSON, err := json.Marshal(calicoAddOn)
	if err != nil {
		panic(err)
	}

	rawCsiJSON, err := json.Marshal(csiAddOn)
	if err != nil {
		panic(err)
	}

	rawCloudProviderJSON, err := json.Marshal(cpAddOn)
	if err != nil {
		panic(err)
	}

	rawAuthSvcJSON, err := json.Marshal(asAddOn)
	if err != nil {
		panic(err)
	}

	rawMetricsServerJSON, err := json.Marshal(metricsServerAddOn)
	if err != nil {
		panic(err)
	}

	compatibilityOffering := string(compatibilityOfferingJSON())
	if !compatible {
		compatibilityOffering = strings.Replace(compatibilityOffering, "v1alpha1", "BOGUS", 1)
	}

	if antreaEnabled {
		cniAntreaAddOnAnnotation := addons.AddOnsAnnotationPrefix + string(addons.Antrea)
		antreaTemplate := addVersionToAntrea(cniVersion)
		antreaAddOn := addons.AddOnSpec{
			Name:    string(addons.Antrea),
			Type:    "inline",
			Version: cniVersion,
			Value:   antreaTemplate,
		}
		rawCniAntreaJSON, err := json.Marshal(antreaAddOn)
		if err != nil {
			panic(err)
		}

		return &vmoprv1.VirtualMachineImage{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("test-%s", strings.Replace(distroVersion, "+", "-", -1)),
				Annotations: map[string]string{
					manifests.VirtualMachineDistributionProperty: string(rawJSON),
					cniCalicoAddOnAnnotation:                     string(rawCniCalicoJSON),
					cniAntreaAddOnAnnotation:                     string(rawCniAntreaJSON),
					csiAddOnAnnotation:                           string(rawCsiJSON),
					cpAddOnAnnotation:                            string(rawCloudProviderJSON),
					authsvcAddOnAnnotation:                       string(rawAuthSvcJSON),
					metricsServerAddOnAnnotation:                 string(rawMetricsServerJSON),
					//VMwareSystemCompatibilityOffering:            compatibilityOffering,
				},
			},
			Spec: vmoprv1.VirtualMachineImageSpec{
				ProductInfo: vmoprv1.VirtualMachineImageProductInfo{
					FullVersion: distroVersion,
				},
			},
		}
	}
	// Without antrea, calico as default
	return &vmoprv1.VirtualMachineImage{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("test-%s", strings.Replace(distroVersion, "+", "-", -1)),
			// TODO: Check if these annotations are needed
			Annotations: map[string]string{
				manifests.VirtualMachineDistributionProperty: string(rawJSON),
				cniCalicoAddOnAnnotation:                     string(rawCniCalicoJSON),
				csiAddOnAnnotation:                           string(rawCsiJSON),
				cpAddOnAnnotation:                            string(rawCloudProviderJSON),
				authsvcAddOnAnnotation:                       string(rawAuthSvcJSON),
				metricsServerAddOnAnnotation:                 string(rawMetricsServerJSON),
				//VMwareSystemCompatibilityOffering:            compatibilityOffering,
			},
		},
		Spec: vmoprv1.VirtualMachineImageSpec{
			ProductInfo: vmoprv1.VirtualMachineImageProductInfo{
				FullVersion: distroVersion,
			},
		},
	}
}

func addVersionToCalico(containerTag string) string {
	FakeCalicoAddOnTemplate := ""
	return fmt.Sprintf(FakeCalicoAddOnTemplate, containerTag, containerTag, containerTag, containerTag)
}

func addVersionToAntrea(containerTag string) string {

	FakeAntreaAddOnTemplate := ""
	return fmt.Sprintf(FakeAntreaAddOnTemplate, containerTag, containerTag, containerTag, containerTag)
}

func addVersionToCSI(containerTag string) string {
	FakePvcsiAddOnTemplate := ""
	return fmt.Sprintf(FakePvcsiAddOnTemplate, containerTag, containerTag, containerTag, containerTag, containerTag, containerTag, containerTag, containerTag)
}

func addVersionToCPI(containerTag string) string {
	FakeCloudProviderAddOnTemplate := ""
	return fmt.Sprintf(FakeCloudProviderAddOnTemplate, containerTag)
}

func addVersionToAuthSvc(containerTag string) string {
	FakeAuthServiceAddonTemplate := ""
	return fmt.Sprintf(FakeAuthServiceAddonTemplate, containerTag)
}

func addVersionToMetricsServer(containerTag string) string {
	FakeMetricsServerAddonTemplate := ""
	return fmt.Sprintf(FakeMetricsServerAddonTemplate, containerTag)
}

func compatibilityOfferingJSON() []byte {
	return []byte(`
[
  {
    "requires": {
      "VirtualMachine": [
        {
          "#data_object_or_protocol": "data object",
          "predicate": {
            "operation": "isVersionSatisfied",
            "arguments": {
              "initiator": "hatchway.vmware.com/wcpguest",
              "receiver": "k8s.io/kubernetes",
              "versions": [
                "vmoperator.vmware.com/v1alpha1"
              ]
            }
          }
        }
      ]
    },
    "version": "v1",
    "offers": {}
  }
]
`)

}