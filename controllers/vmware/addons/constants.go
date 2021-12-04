package addons

// AddonName is the type of an add-on.
type AddonName string

const (
	// Calico returns the Calico CNI add-on.
	Calico AddonName = "calico"

	// Antrea returns the Antrea CNI add-on.
	Antrea AddonName = "antrea"

	// VMwareGuestClusterCPI returns the VMware Guest Cluster CPI
	// add-on.
	VMwareGuestClusterCPI AddonName = "vmware-guest-cluster"

	// VMwareGuestClusterCSI returns the VMware Guest Cluster CSI add-on.
	VMwareGuestClusterCSI AddonName = "pvcsi"

	// VMwareGuestClusterAuthsvc returns the VMware Guest Cluster Auth service add-on
	VMwareGuestClusterAuthsvc AddonName = "authsvc"

	// VMwareGuestClusterMetricsServer returns the VMware Guest Cluster Metrics Server service add-on
	VMwareGuestClusterMetricsServer AddonName = "metrics-server"

	//VirtualMachineImage Addon Annotation Prefix
	// AddOnsAnnotationPrefix returns prefix of VirtualMachineImage Addon Annotation
	AddOnsAnnotationPrefix = "vmware-system.guest.kubernetes.addons."

	// AntreaNSXRouted returns the antrea-nsx-routed add-on
	AntreaNSXRouted AddonName = "antrea-nsx-routed"
)

type AddOnSpec struct {
	// String Version of the AddOn AddonName ("calico", "pvcsi"..)
	Name string `json:"name"`
	// Currently supported type - "inline"
	Type string `json:"type"`
	// Version of the AddOn
	Version string `json:"version"`
	// Yaml Template for AddOn
	Value string `json:"value"`
}
