// Copyright (c) 2019-2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package builder

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"
	"strings"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	bootstrapv1 "sigs.k8s.io/cluster-api/bootstrap/kubeadm/api/v1alpha3"
	kcpv1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1alpha3"

	vmoprv1 "github.com/vmware-tanzu/vm-operator-api/api/v1alpha1"

	infrav1 "gitlab.eng.vmware.com/core-build/cluster-api-provider-wcp/api/v1alpha3"

	tkgv1 "gitlab.eng.vmware.com/core-build/guest-cluster-controller/apis/run.tanzu/v1alpha2"
	"gitlab.eng.vmware.com/core-build/guest-cluster-controller/controllers/addons"
	"gitlab.eng.vmware.com/core-build/guest-cluster-controller/controllers/common"
	"gitlab.eng.vmware.com/core-build/guest-cluster-controller/controllers/util"
	"gitlab.eng.vmware.com/core-build/guest-cluster-controller/controllers/util/vlabel"
	"gitlab.eng.vmware.com/core-build/guest-cluster-controller/pkg/manifests"
)

// nolint:misspell
const (
	// FakeDistributionVersion must be in semver format to pass validation
	FakeDistributionVersion             = "v1.15.5+vmware.1.66-guest.1.821"
	FakeKubernetesVersion               = "v1.15.5+vmware.1"
	FakeEtcdVersion                     = "v3.3.10_vmware.1"
	FakeDNSVersion                      = "v1.3.1_vmware.1"
	FakeCNIVersion                      = "v3.1.1"
	FakeCSIVersion                      = "v3.1.1"
	FakeCPIVersion                      = "v3.1.1"
	FakeAuthServiceVersion              = "v3.1.1"
	FakeMetricsServerVersion            = "v0.4.1"
	FakeNonExistDistributionVersion     = "v1.15.0+vmware.1.33-guest.1.693"
	FakeIncompatibleDistributionVersion = "v1.19.0+vmware.1-guest.1"
	VMwareSystemCompatibilityOffering   = "vmware-system.compatibilityoffering"
	// FakeAddonTemplates pushed into the VirtualMachineimage for tests.
	FakeCalicoAddOnTemplate = `
---
# Source: calico/templates/calico-config.yaml
# This ConfigMap is used to configure a self-hosted Calico installation.
kind: ConfigMap
apiVersion: v1
metadata:
  name: calico-config
  namespace: kube-system
data:
  # Typha is disabled.
  typha_service_name: "none"
  # Configure the Calico backend to use.
  calico_backend: "bird"

  # Configure the MTU to use
  veth_mtu: "1440"

  # The CNI network configuration to install on each node.  The special
  # values in this config will be automatically populated.
  cni_network_config: |-
    {
      "name": "k8s-pod-network",
      "cniVersion": "0.3.0",
      "plugins": [
        {
          "type": "calico",
          "log_level": "info",
          "datastore_type": "kubernetes",
          "nodename": "__KUBERNETES_NODE_NAME__",
          "mtu": __CNI_MTU__,
          "ipam": {
            "type": "calico-ipam"
          },
          "policy": {
              "type": "k8s"
          },
          "kubernetes": {
              "kubeconfig": "__KUBECONFIG_FILEPATH__"
          }
        },
        {
          "type": "portmap",
          "snat": true,
          "capabilities": {"portMappings": true}
        }
      ]
    }

---
# Source: calico/templates/kdd-crds.yaml
# Create all the CustomResourceDefinitions needed for
# Calico policy and networking mode.

apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
   name: felixconfigurations.crd.projectcalico.org
spec:
  scope: Cluster
  group: crd.projectcalico.org
  version: v1
  names:
    kind: FelixConfiguration
    plural: felixconfigurations
    singular: felixconfiguration
---

apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: ipamblocks.crd.projectcalico.org
spec:
  scope: Cluster
  group: crd.projectcalico.org
  version: v1
  names:
    kind: IPAMBlock
    plural: ipamblocks
    singular: ipamblock

---

apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: blockaffinities.crd.projectcalico.org
spec:
  scope: Cluster
  group: crd.projectcalico.org
  version: v1
  names:
    kind: BlockAffinity
    plural: blockaffinities
    singular: blockaffinity

---

apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: ipamhandles.crd.projectcalico.org
spec:
  scope: Cluster
  group: crd.projectcalico.org
  version: v1
  names:
    kind: IPAMHandle
    plural: ipamhandles
    singular: ipamhandle

---

apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: ipamconfigs.crd.projectcalico.org
spec:
  scope: Cluster
  group: crd.projectcalico.org
  version: v1
  names:
    kind: IPAMConfig
    plural: ipamconfigs
    singular: ipamconfig

---

apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: bgppeers.crd.projectcalico.org
spec:
  scope: Cluster
  group: crd.projectcalico.org
  version: v1
  names:
    kind: BGPPeer
    plural: bgppeers
    singular: bgppeer

---

apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: bgpconfigurations.crd.projectcalico.org
spec:
  scope: Cluster
  group: crd.projectcalico.org
  version: v1
  names:
    kind: BGPConfiguration
    plural: bgpconfigurations
    singular: bgpconfiguration

---

apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: ippools.crd.projectcalico.org
spec:
  scope: Cluster
  group: crd.projectcalico.org
  version: v1
  names:
    kind: IPPool
    plural: ippools
    singular: ippool

---

apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: hostendpoints.crd.projectcalico.org
spec:
  scope: Cluster
  group: crd.projectcalico.org
  version: v1
  names:
    kind: HostEndpoint
    plural: hostendpoints
    singular: hostendpoint

---

apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: clusterinformations.crd.projectcalico.org
spec:
  scope: Cluster
  group: crd.projectcalico.org
  version: v1
  names:
    kind: ClusterInformation
    plural: clusterinformations
    singular: clusterinformation

---

apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: globalnetworkpolicies.crd.projectcalico.org
spec:
  scope: Cluster
  group: crd.projectcalico.org
  version: v1
  names:
    kind: GlobalNetworkPolicy
    plural: globalnetworkpolicies
    singular: globalnetworkpolicy

---

apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: globalnetworksets.crd.projectcalico.org
spec:
  scope: Cluster
  group: crd.projectcalico.org
  version: v1
  names:
    kind: GlobalNetworkSet
    plural: globalnetworksets
    singular: globalnetworkset

---

apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: networkpolicies.crd.projectcalico.org
spec:
  scope: Namespaced
  group: crd.projectcalico.org
  version: v1
  names:
    kind: NetworkPolicy
    plural: networkpolicies
    singular: networkpolicy
---
# Source: calico/templates/rbac.yaml

# Include a clusterrole for the kube-controllers component,
# and bind it to the calico-kube-controllers serviceaccount.
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: calico-kube-controllers
rules:
  # Nodes are watched to monitor for deletions.
  - apiGroups: [""]
    resources:
      - nodes
    verbs:
      - watch
      - list
      - get
  # Pods are queried to check for existence.
  - apiGroups: [""]
    resources:
      - pods
    verbs:
      - get
  # IPAM resources are manipulated when nodes are deleted.
  - apiGroups: ["crd.projectcalico.org"]
    resources:
      - ippools
    verbs:
      - list
  - apiGroups: ["crd.projectcalico.org"]
    resources:
      - blockaffinities
      - ipamblocks
      - ipamhandles
    verbs:
      - get
      - list
      - create
      - update
      - delete
  # Needs access to update clusterinformations.
  - apiGroups: ["crd.projectcalico.org"]
    resources:
      - clusterinformations
    verbs:
      - get
      - create
      - update
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: calico-kube-controllers
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: calico-kube-controllers
subjects:
- kind: ServiceAccount
  name: calico-kube-controllers
  namespace: kube-system
---
# Include a clusterrole for the calico-node DaemonSet,
# and bind it to the calico-node serviceaccount.
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: calico-node
rules:
  # The CNI plugin needs to get pods, nodes, and namespaces.
  - apiGroups: [""]
    resources:
      - pods
      - nodes
      - namespaces
    verbs:
      - get
  - apiGroups: [""]
    resources:
      - endpoints
      - services
    verbs:
      # Used to discover service IPs for advertisement.
      - watch
      - list
      # Used to discover Typhas.
      - get
  - apiGroups: [""]
    resources:
      - nodes/status
    verbs:
      # Needed for clearing NodeNetworkUnavailable flag.
      - patch
      # Calico stores some configuration information in node annotations.
      - update
  # Watch for changes to Kubernetes NetworkPolicies.
  - apiGroups: ["networking.k8s.io"]
    resources:
      - networkpolicies
    verbs:
      - watch
      - list
  # Used by Calico for policy information.
  - apiGroups: [""]
    resources:
      - pods
      - namespaces
      - serviceaccounts
    verbs:
      - list
      - watch
  # The CNI plugin patches pods/status.
  - apiGroups: [""]
    resources:
      - pods/status
    verbs:
      - patch
  # Calico monitors various CRDs for config.
  - apiGroups: ["crd.projectcalico.org"]
    resources:
      - globalfelixconfigs
      - felixconfigurations
      - bgppeers
      - globalbgpconfigs
      - bgpconfigurations
      - ippools
      - ipamblocks
      - globalnetworkpolicies
      - globalnetworksets
      - networkpolicies
      - clusterinformations
      - hostendpoints
    verbs:
      - get
      - list
      - watch
  # Calico must create and update some CRDs on startup.
  - apiGroups: ["crd.projectcalico.org"]
    resources:
      - ippools
      - felixconfigurations
      - clusterinformations
    verbs:
      - create
      - update
  # Calico stores some configuration information on the node.
  - apiGroups: [""]
    resources:
      - nodes
    verbs:
      - get
      - list
      - watch
  # These permissions are only requried for upgrade from v2.6, and can
  # be removed after upgrade or on fresh installations.
  - apiGroups: ["crd.projectcalico.org"]
    resources:
      - bgpconfigurations
      - bgppeers
    verbs:
      - create
      - update
  # These permissions are required for Calico CNI to perform IPAM allocations.
  - apiGroups: ["crd.projectcalico.org"]
    resources:
      - blockaffinities
      - ipamblocks
      - ipamhandles
    verbs:
      - get
      - list
      - create
      - update
      - delete
  - apiGroups: ["crd.projectcalico.org"]
    resources:
      - ipamconfigs
    verbs:
      - get
  # Block affinities must also be watchable by confd for route aggregation.
  - apiGroups: ["crd.projectcalico.org"]
    resources:
      - blockaffinities
    verbs:
      - watch
  # The Calico IPAM migration needs to get daemonsets. These permissions can be
  # removed if not upgrading from an installation using host-local IPAM.
  - apiGroups: ["apps"]
    resources:
      - daemonsets
    verbs:
      - get
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: calico-node
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: calico-node
subjects:
- kind: ServiceAccount
  name: calico-node
  namespace: kube-system

---
# Source: calico/templates/calico-node.yaml
# This manifest installs the node container, as well
# as the Calico CNI plugins and network config on
# each control plane and control plane node in a Kubernetes cluster.
kind: DaemonSet
apiVersion: apps/v1
metadata:
  name: calico-node
  namespace: kube-system
  labels:
    k8s-app: calico-node
spec:
  selector:
    matchLabels:
      k8s-app: calico-node
  updateStrategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
  template:
    metadata:
      labels:
        k8s-app: calico-node
      annotations:
        # This, along with the CriticalAddonsOnly toleration below,
        # marks the pod as a critical add-on, ensuring it gets
        # priority scheduling and that its resources are reserved
        # if it ever gets evicted.
        scheduler.alpha.kubernetes.io/critical-pod: ''
    spec:
      nodeSelector:
        beta.kubernetes.io/os: linux
      hostNetwork: true
      tolerations:
        # Make sure calico-node gets scheduled on all nodes.
        - effect: NoSchedule
          operator: Exists
        # Mark the pod as a critical add-on for rescheduling.
        - key: CriticalAddonsOnly
          operator: Exists
        - effect: NoExecute
          operator: Exists
      serviceAccountName: calico-node
      # Minimize downtime during a rolling upgrade or deletion; tell Kubernetes to do a "force
      # deletion": https://kubernetes.io/docs/concepts/workloads/pods/pod/#termination-of-pods.
      terminationGracePeriodSeconds: 0
      initContainers:
        # This container performs upgrade from host-local IPAM to calico-ipam.
        # It can be deleted if this is a fresh installation, or if you have already
        # upgraded to use calico-ipam.
        - name: upgrade-ipam
          image: vmware/calico-cni:%s
          command: ["/opt/cni/bin/calico-ipam", "-upgrade"]
          env:
            - name: KUBERNETES_NODE_NAME
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
            - name: CALICO_NETWORKING_BACKEND
              valueFrom:
                configMapKeyRef:
                  name: calico-config
                  key: calico_backend
          volumeMounts:
            - mountPath: /var/lib/cni/networks
              name: host-local-net-dir
            - mountPath: /host/opt/cni/bin
              name: cni-bin-dir
        # This container installs the Calico CNI binaries
        # and CNI network config file on each node.
        - name: install-cni
          image: vmware/calico-cni:%s
          command: ["/install-cni.sh"]
          env:
            # Name of the CNI config file to create.
            - name: CNI_CONF_NAME
              value: "10-calico.conflist"
            # The CNI network config to install on each node.
            - name: CNI_NETWORK_CONFIG
              valueFrom:
                configMapKeyRef:
                  name: calico-config
                  key: cni_network_config
            # Set the hostname based on the k8s node name.
            - name: KUBERNETES_NODE_NAME
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
            # CNI MTU Config variable
            - name: CNI_MTU
              valueFrom:
                configMapKeyRef:
                  name: calico-config
                  key: veth_mtu
            # Prevents the container from sleeping forever.
            - name: SLEEP
              value: "false"
          volumeMounts:
            - mountPath: /host/opt/cni/bin
              name: cni-bin-dir
            - mountPath: /host/etc/cni/net.d
              name: cni-net-dir
      containers:
        # Runs node container on each Kubernetes node.  This
        # container programs network policy and routes on each
        # host.
        - name: calico-node
          image: vmware/calico-node:%s
          env:
            # Use Kubernetes API as the backing datastore.
            - name: DATASTORE_TYPE
              value: "kubernetes"
            # Wait for the datastore.
            - name: WAIT_FOR_DATASTORE
              value: "true"
            # Set based on the k8s node name.
            - name: NODENAME
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
            # Choose the backend to use.
            - name: CALICO_NETWORKING_BACKEND
              valueFrom:
                configMapKeyRef:
                  name: calico-config
                  key: calico_backend
            # Cluster type to identify the deployment type
            - name: CLUSTER_TYPE
              value: "k8s,bgp"
            # Auto-detect the BGP IP address.
            - name: IP
              value: "autodetect"
            # Enable IPIP
            - name: CALICO_IPV4POOL_IPIP
              value: "Always"
            # Set MTU for tunnel device used if ipip is enabled
            - name: FELIX_IPINIPMTU
              valueFrom:
                configMapKeyRef:
                  name: calico-config
                  key: veth_mtu
            # The default IPv4 pool to create on startup if none exists. Pod IPs will be
            # chosen from this range. Changing this value after installation will have
            # no effect. This should fall within --cluster-cidr.
            - name: CALICO_IPV4POOL_CIDR
              value: {{ .PodCIDR }}
            # Disable file logging so kubectl logs works.
            - name: CALICO_DISABLE_FILE_LOGGING
              value: "true"
            # Set Felix endpoint to host default action to ACCEPT.
            - name: FELIX_DEFAULTENDPOINTTOHOSTACTION
              value: "ACCEPT"
            # Disable IPv6 on Kubernetes.
            - name: FELIX_IPV6SUPPORT
              value: "false"
            # Set Felix logging to "info"
            - name: FELIX_LOGSEVERITYSCREEN
              value: "info"
            - name: FELIX_HEALTHENABLED
              value: "true"
          securityContext:
            privileged: true
          resources:
            requests:
              cpu: 250m
          livenessProbe:
            httpGet:
              path: /liveness
              port: 9099
              host: localhost
            periodSeconds: 10
            initialDelaySeconds: 10
            failureThreshold: 6
          readinessProbe:
            exec:
              command:
              - /bin/calico-node
              - -bird-ready
              - -felix-ready
            periodSeconds: 10
          volumeMounts:
            - mountPath: /lib/modules
              name: lib-modules
              readOnly: true
            - mountPath: /run/xtables.lock
              name: xtables-lock
              readOnly: false
            - mountPath: /var/run/calico
              name: var-run-calico
              readOnly: false
            - mountPath: /var/lib/calico
              name: var-lib-calico
              readOnly: false
      volumes:
        # Used by node.
        - name: lib-modules
          hostPath:
            path: /lib/modules
        - name: var-run-calico
          hostPath:
            path: /var/run/calico
        - name: var-lib-calico
          hostPath:
            path: /var/lib/calico
        - name: xtables-lock
          hostPath:
            path: /run/xtables.lock
            type: FileOrCreate
        # Used to install CNI.
        - name: cni-bin-dir
          hostPath:
            path: /opt/cni/bin
        - name: cni-net-dir
          hostPath:
            path: /etc/cni/net.d
        # Mount in the directory for host-local IPAM allocations. This is
        # used when upgrading from host-local to calico-ipam, and can be removed
        # if not using the upgrade-ipam init container.
        - name: host-local-net-dir
          hostPath:
            path: /var/lib/cni/networks
---

apiVersion: v1
kind: ServiceAccount
metadata:
  name: calico-node
  namespace: kube-system

---
# Source: calico/templates/calico-kube-controllers.yaml
# This manifest deploys the Calico node controller.
# See https://github.com/projectcalico/kube-controllers
apiVersion: apps/v1
kind: Deployment
metadata:
  name: calico-kube-controllers
  namespace: kube-system
  labels:
    k8s-app: calico-kube-controllers
  annotations:
    scheduler.alpha.kubernetes.io/critical-pod: ''
spec:
  # The controllers can only have a single active instance.
  replicas: 1
  selector:
    matchLabels:
      k8s-app: calico-kube-controllers
  strategy:
    type: Recreate
  template:
    metadata:
      name: calico-kube-controllers
      namespace: kube-system
      labels:
        k8s-app: calico-kube-controllers
      annotations:
        scheduler.alpha.kubernetes.io/critical-pod: ''
    spec:
      nodeSelector:
        beta.kubernetes.io/os: linux
      tolerations:
        # Mark the pod as a critical add-on for rescheduling.
        - key: CriticalAddonsOnly
          operator: Exists
        - key: node-role.kubernetes.io/master
          effect: NoSchedule
      serviceAccountName: calico-kube-controllers
      containers:
        - name: calico-kube-controllers
          image: vmware/calico-kube-controller:%s
          env:
            # Choose which controllers to run.
            - name: ENABLED_CONTROLLERS
              value: node
            - name: DATASTORE_TYPE
              value: kubernetes
          readinessProbe:
            exec:
              command:
              - /usr/bin/check-status
              - -r

---

apiVersion: v1
kind: ServiceAccount
metadata:
  name: calico-kube-controllers
  namespace: kube-system
---
# Source: calico/templates/calico-etcd-secrets.yaml

---
# Source: calico/templates/calico-typha.yaml

---
# Source: calico/templates/configure-canal.yaml
`

	FakeAntreaAddOnTemplate = `
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  labels:
    app: antrea
  name: antreaagentinfos.clusterinformation.antrea.tanzu.vmware.com
spec:
  group: clusterinformation.antrea.tanzu.vmware.com
  names:
    kind: AntreaAgentInfo
    plural: antreaagentinfos
    shortNames:
    - aai
    singular: antreaagentinfo
  scope: Cluster
  versions:
  - name: v1beta1
    served: true
    storage: true
---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  labels:
    app: antrea
  name: antreacontrollerinfos.clusterinformation.antrea.tanzu.vmware.com
spec:
  group: clusterinformation.antrea.tanzu.vmware.com
  names:
    kind: AntreaControllerInfo
    plural: antreacontrollerinfos
    shortNames:
    - aci
    singular: antreacontrollerinfo
  scope: Cluster
  versions:
  - name: v1beta1
    served: true
    storage: true
---
apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    app: antrea
  name: antctl
  namespace: kube-system
---
apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    app: antrea
  name: antrea-agent
  namespace: kube-system
---
apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    app: antrea
  name: antrea-controller
  namespace: kube-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  labels:
    app: antrea
  name: antctl
rules:
- apiGroups:
  - networking.antrea.tanzu.vmware.com
  resources:
  - networkpolicies
  - appliedtogroups
  - addressgroups
  verbs:
  - get
  - list
- apiGroups:
  - system.antrea.tanzu.vmware.com
  resources:
  - controllerinfos
  - agentinfos
  verbs:
  - get
- apiGroups:
  - system.antrea.tanzu.vmware.com
  resources:
  - supportbundles
  verbs:
  - get
  - post
- apiGroups:
  - system.antrea.tanzu.vmware.com
  resources:
  - supportbundles/download
  verbs:
  - get
- nonResourceURLs:
  - /agentinfo
  - /addressgroups
  - /appliedtogroups
  - /networkpolicies
  - /ovsflows
  - /ovstracing
  - /podinterfaces
  verbs:
  - get
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  labels:
    app: antrea
  name: antrea-agent
rules:
- apiGroups:
  - ""
  resources:
  - nodes
  verbs:
  - get
  - watch
  - list
- apiGroups:
  - ""
  resources:
  - pods
  - services
  verbs:
  - get
  - list
- apiGroups:
  - clusterinformation.antrea.tanzu.vmware.com
  resources:
  - antreaagentinfos
  verbs:
  - get
  - create
  - update
  - delete
- apiGroups:
  - networking.antrea.tanzu.vmware.com
  resources:
  - networkpolicies
  - appliedtogroups
  - addressgroups
  verbs:
  - get
  - watch
  - list
- apiGroups:
  - authentication.k8s.io
  resources:
  - tokenreviews
  verbs:
  - create
- apiGroups:
  - authorization.k8s.io
  resources:
  - subjectaccessreviews
  verbs:
  - create
- apiGroups:
  - ""
  resourceNames:
  - extension-apiserver-authentication
  resources:
  - configmaps
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - ""
  resourceNames:
  - antrea-ca
  resources:
  - configmaps
  verbs:
  - get
  - watch
  - list
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  labels:
    app: antrea
  name: antrea-controller
rules:
- apiGroups:
  - ""
  resources:
  - nodes
  - pods
  - namespaces
  verbs:
  - get
  - watch
  - list
- apiGroups:
  - networking.k8s.io
  resources:
  - networkpolicies
  verbs:
  - get
  - watch
  - list
- apiGroups:
  - clusterinformation.antrea.tanzu.vmware.com
  resources:
  - antreacontrollerinfos
  verbs:
  - get
  - create
  - update
  - delete
- apiGroups:
  - clusterinformation.antrea.tanzu.vmware.com
  resources:
  - antreaagentinfos
  verbs:
  - list
  - delete
- apiGroups:
  - authentication.k8s.io
  resources:
  - tokenreviews
  verbs:
  - create
- apiGroups:
  - authorization.k8s.io
  resources:
  - subjectaccessreviews
  verbs:
  - create
- apiGroups:
  - ""
  resourceNames:
  - extension-apiserver-authentication
  resources:
  - configmaps
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - ""
  resourceNames:
  - antrea-ca
  resources:
  - configmaps
  verbs:
  - get
  - update
- apiGroups:
  - apiregistration.k8s.io
  resourceNames:
  - v1beta1.system.antrea.tanzu.vmware.com
  - v1beta1.networking.antrea.tanzu.vmware.com
  resources:
  - apiservices
  verbs:
  - get
  - update
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  labels:
    app: antrea
  name: antctl
  namespace: kube-system
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: antctl
subjects:
- kind: ServiceAccount
  name: antctl
  namespace: kube-system
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  labels:
    app: antrea
  name: antrea-agent
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: antrea-agent
subjects:
- kind: ServiceAccount
  name: antrea-agent
  namespace: kube-system
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  labels:
    app: antrea
  name: antrea-controller
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: antrea-controller
subjects:
- kind: ServiceAccount
  name: antrea-controller
  namespace: kube-system
---
apiVersion: v1
data:
  antrea-agent.conf: |
    # Name of the OpenVSwitch bridge antrea-agent will create and use.
    # Make sure it doesn't conflict with your existing OpenVSwitch bridges.
    #ovsBridge: br-int

    # Datapath type to use for the OpenVSwitch bridge created by Antrea. Supported values are:
    # - system
    # - netdev
    # 'system' is the default value and corresponds to the kernel datapath. Use 'netdev' to run
    # OVS in userspace mode. Userspace mode requires the tun device driver to be available.
    #ovsDatapathType: system

    # Name of the interface antrea-agent will create and use for host <--> pod communication.
    # Make sure it doesn't conflict with your existing interfaces.
    hostGateway: antrea-gw0

    # Encapsulation mode for communication between Pods across Nodes, supported values:
    # - vxlan (default)
    # - geneve
    # - gre
    # - stt
    tunnelType: geneve

    # Default MTU to use for the host gateway interface and the network interface of each Pod. If
    # omitted, antrea-agent will default this value to 1450 to accommodate for tunnel encapsulate
    # overhead.
    #defaultMTU: 1450

    # Whether or not to enable IPsec encryption of tunnel traffic. IPsec encryption is only supported
    # for the GRE tunnel type.
    #enableIPSecTunnel: false

    # CIDR Range for services in cluster. It's required to support egress network policy, should
    # be set to the same value as the one specified by --service-cluster-ip-range for kube-apiserver.
    serviceCIDR: {{.ClusterIPCIDR}}

    # Determines how traffic is encapsulated. It has the following options
    # encap(default): Inter-node Pod traffic is always encapsulated and Pod to outbound traffic is masqueraded.
    # noEncap: Inter-node Pod traffic is not encapsulated, but Pod to outbound traffic is masqueraded.
    #          Underlying network must be capable of supporting Pod traffic across IP subnet.
    # hybrid: noEncap if worker Nodes on same subnet, otherwise encap.
    # networkPolicyOnly: Antrea enforces NetworkPolicy only, and utilizes CNI chaining and delegates Pod IPAM and connectivity to primary CNI.
    #
    #trafficEncapMode: encap

    # The port for the antrea-agent APIServer to serve on.
    # Note that if it's set to another value, the 'containerPort' of the 'api' port of the
    # 'antrea-agent' container must be set to the same value.
    #apiPort: 10350

    # Enable metrics exposure via Prometheus. Initializes Prometheus metrics listener.
    #enablePrometheusMetrics: false
  antrea-cni.conflist: |
    {
        "cniVersion":"0.3.0",
        "name": "antrea",
        "plugins": [
            {
                "type": "antrea",
                "ipam": {
                    "type": "host-local"
                }
            },
            {
                "type": "portmap",
                "capabilities": {"portMappings": true}
            }
        ]
    }
  antrea-controller.conf: |
    # The port for the antrea-controller APIServer to serve on.
    # Note that if it's set to another value, the 'containerPort' of the 'api' port of the
    # 'antrea-controller' container must be set to the same value.
    #apiPort: 10349

    # Enable metrics exposure via Prometheus. Initializes Prometheus metrics listener.
    #enablePrometheusMetrics: false

    # Indicates whether to use auto-generated self-signed TLS certificate.
    # If false, A secret named "kube-system/antrea-controller-tls" must be provided with the following keys:
    #   ca.crt: <CA certificate>
    #   tls.crt: <TLS certificate>
    #   tls.key: <TLS private key>
    #selfSignedCert: true
    {{- if .UseCertFromProvider}}
    selfSignedCert: false
    {{- end}}
kind: ConfigMap
metadata:
  annotations: {}
  labels:
    app: antrea
  name: antrea-config-mf4t8c67c8
  namespace: kube-system
---
apiVersion: v1
kind: Service
metadata:
  labels:
    app: antrea
  name: antrea
  namespace: kube-system
spec:
  type: ClusterIp
  ports:
  - port: 443
    protocol: TCP
    targetPort: api
  selector:
    app: antrea
    component: antrea-controller
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: antrea
    component: antrea-controller
  name: antrea-controller
  namespace: kube-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: antrea
      component: antrea-controller
  strategy:
    type: Recreate
  template:
    metadata:
      labels:
        app: antrea
        component: antrea-controller
    spec:
      containers:
      - args:
        - --config
        - /etc/antrea/antrea-controller.conf
        - --logtostderr=false
        - --log_dir
        - /var/log/antrea
        - --alsologtostderr
        command:
        - antrea-controller
        env:
        - name: POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: POD_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        - name: NODE_NAME
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
        image: vmware.io/antrea/antrea-photon:%s
        imagePullPolicy: IfNotPresent
        name: antrea-controller
        ports:
        - containerPort: 10349
          name: api
          protocol: TCP
        readinessProbe:
          failureThreshold: 5
          httpGet:
            host: 127.0.0.1
            path: /healthz
            port: api
            scheme: HTTPS
          initialDelaySeconds: 5
          periodSeconds: 10
          timeoutSeconds: 5
        resources:
          requests:
            cpu: 200m
        volumeMounts:
        - mountPath: /etc/antrea/antrea-controller.conf
          name: antrea-config
          readOnly: true
          subPath: antrea-controller.conf
{{- if .UseCertFromProvider}}
        - mountPath: /var/run/antrea/antrea-controller-tls
          name: antrea-controller-tls
{{- end}}
        - mountPath: /var/log/antrea
          name: host-var-log-antrea
      hostNetwork: true
      nodeSelector:
        kubernetes.io/os: linux
      priorityClassName: system-cluster-critical
      serviceAccountName: antrea-controller
      tolerations:
      - key: CriticalAddonsOnly
        operator: Exists
      - effect: NoSchedule
        key: node-role.kubernetes.io/master
      volumes:
      - configMap:
          name: antrea-config-mf4t8c67c8
        name: antrea-config
{{- if .UseCertFromProvider}}
      - name: antrea-controller-tls
        secret:
          defaultMode: 256
          secretName: {{.ProviderCertSecretName}}
{{- end}}
      - hostPath:
          path: /var/log/antrea
          type: DirectoryOrCreate
        name: host-var-log-antrea
---
apiVersion: apps/v1
kind: DaemonSet
metadata:
  labels:
    app: antrea
    component: antrea-agent
  name: antrea-agent
  namespace: kube-system
spec:
  selector:
    matchLabels:
      app: antrea
      component: antrea-agent
  template:
    metadata:
      labels:
        app: antrea
        component: antrea-agent
    spec:
      containers:
      - args:
        - --config
        - /etc/antrea/antrea-agent.conf
        - --logtostderr=false
        - --log_dir
        - /var/log/antrea
        - --alsologtostderr
        command:
        - antrea-agent
        env:
        - name: POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: POD_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        - name: NODE_NAME
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
        image: vmware.io/antrea/antrea-photon:%s
        imagePullPolicy: IfNotPresent
        livenessProbe:
          exec:
            command:
            - /bin/sh
            - -c
            - container_liveness_probe agent
          failureThreshold: 5
          initialDelaySeconds: 5
          periodSeconds: 10
          timeoutSeconds: 5
        name: antrea-agent
        ports:
        - containerPort: 10350
          name: api
          protocol: TCP
        readinessProbe:
          failureThreshold: 5
          httpGet:
            host: 127.0.0.1
            path: /healthz
            port: api
            scheme: HTTPS
          initialDelaySeconds: 5
          periodSeconds: 10
          timeoutSeconds: 5
        resources:
          requests:
            cpu: 200m
        securityContext:
          privileged: true
        volumeMounts:
        - mountPath: /etc/antrea/antrea-agent.conf
          name: antrea-config
          readOnly: true
          subPath: antrea-agent.conf
        - mountPath: /var/run/antrea
          name: host-var-run-antrea
        - mountPath: /var/run/openvswitch
          name: host-var-run-antrea
          subPath: openvswitch
        - mountPath: /var/lib/cni
          name: host-var-run-antrea
          subPath: cni
        - mountPath: /var/log/antrea
          name: host-var-log-antrea
        - mountPath: /host/proc
          name: host-proc
          readOnly: true
        - mountPath: /host/var/run/netns
          mountPropagation: HostToContainer
          name: host-var-run-netns
          readOnly: true
        - mountPath: /run/xtables.lock
          name: xtables-lock
      - command:
        - start_ovs
        image: vmware.io/antrea/antrea-photon:%s
        imagePullPolicy: IfNotPresent
        livenessProbe:
          exec:
            command:
            - /bin/sh
            - -c
            - timeout 5 container_liveness_probe ovs
          initialDelaySeconds: 5
          periodSeconds: 5
        name: antrea-ovs
        resources:
          requests:
            cpu: 200m
        securityContext:
          capabilities:
            add:
            - SYS_NICE
            - NET_ADMIN
            - SYS_ADMIN
            - IPC_LOCK
        volumeMounts:
        - mountPath: /var/run/openvswitch
          name: host-var-run-antrea
          subPath: openvswitch
        - mountPath: /var/log/openvswitch
          name: host-var-log-antrea
          subPath: openvswitch
      hostNetwork: true
      initContainers:
      - command:
        - install_cni
        image: vmware.io/antrea/antrea-photon:%s
        imagePullPolicy: IfNotPresent
        name: install-cni
        resources:
          requests:
            cpu: 100m
        securityContext:
          capabilities:
            add:
            - SYS_MODULE
        volumeMounts:
        - mountPath: /etc/antrea/antrea-cni.conflist
          name: antrea-config
          readOnly: true
          subPath: antrea-cni.conflist
        - mountPath: /host/etc/cni/net.d
          name: host-cni-conf
        - mountPath: /host/opt/cni/bin
          name: host-cni-bin
        - mountPath: /lib/modules
          name: host-lib-modules
          readOnly: true
        - mountPath: /sbin/depmod
          name: host-depmod
          readOnly: true
      nodeSelector:
        kubernetes.io/os: linux
      priorityClassName: system-node-critical
      serviceAccountName: antrea-agent
      tolerations:
      - key: CriticalAddonsOnly
        operator: Exists
      - effect: NoSchedule
        operator: Exists
      volumes:
      - configMap:
          name: antrea-config-mf4t8c67c8
        name: antrea-config
      - hostPath:
          path: /etc/cni/net.d
        name: host-cni-conf
      - hostPath:
          path: /opt/cni/bin
        name: host-cni-bin
      - hostPath:
          path: /proc
        name: host-proc
      - hostPath:
          path: /var/run/netns
        name: host-var-run-netns
      - hostPath:
          path: /var/run/antrea
          type: DirectoryOrCreate
        name: host-var-run-antrea
      - hostPath:
          path: /var/log/antrea
          type: DirectoryOrCreate
        name: host-var-log-antrea
      - hostPath:
          path: /lib/modules
        name: host-lib-modules
      - hostPath:
          path: /sbin/depmod
        name: host-depmod
      - hostPath:
          path: /run/xtables.lock
          type: FileOrCreate
        name: xtables-lock
  updateStrategy:
    type: RollingUpdate
`

	FakePvcsiAddOnTemplate = `
apiVersion: v1
kind: Namespace
metadata:
  name: {{ .PVCSINamespace }}
---
kind: ServiceAccount
apiVersion: v1
metadata:
  name: vsphere-csi-controller
  namespace: {{ .PVCSINamespace }}
---
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: vsphere-csi-controller-role
rules:
  - apiGroups: [""]
    resources: ["nodes", "persistentvolumeclaims", "pods"]
    verbs: ["get", "list", "watch"]
  - apiGroups: [""]
    resources: ["persistentvolumes"]
    verbs: ["get", "list", "watch", "create", "update", "delete"]
  - apiGroups: [""]
    resources: ["events"]
    verbs: ["get", "list", "watch", "create", "update", "patch"]
  - apiGroups: ["coordination.k8s.io"]
    resources: ["leases"]
    verbs: ["get", "watch", "list", "delete", "update", "create"]
  - apiGroups: ["storage.k8s.io"]
    resources: ["storageclasses", "csinodes"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["storage.k8s.io"]
    resources: ["volumeattachments"]
    verbs: ["get", "list", "watch", "update"]
  - apiGroups: ["policy"]
    resources: ["podsecuritypolicies"]
    verbs: ["use"]
    resourceNames: ["vmware-system-privileged"]
---
kind: Role
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: vsphere-csi-node-role
  namespace: {{ .PVCSINamespace }}
rules:
  - apiGroups:
    - "policy"
    resources:
    - podsecuritypolicies
    verbs:
    - use
    resourceNames:
    - vmware-system-privileged
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: vsphere-csi-controller-binding
subjects:
  - kind: ServiceAccount
    name: vsphere-csi-controller
    namespace: {{ .PVCSINamespace }}
roleRef:
  kind: ClusterRole
  name: vsphere-csi-controller-role
  apiGroup: rbac.authorization.k8s.io
---
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: vsphere-csi-node-binding
  namespace: {{ .PVCSINamespace }}
subjects:
  - kind: ServiceAccount
    name: default
    namespace: {{ .PVCSINamespace }}
roleRef:
  kind: Role
  name: vsphere-csi-node-role
  apiGroup: rbac.authorization.k8s.io
---
kind: Deployment
apiVersion: apps/v1
metadata:
  name: vsphere-csi-controller
  namespace: {{ .PVCSINamespace }}
spec:
  replicas: 1
  strategy:
    type: Recreate
  selector:
    matchLabels:
      app: vsphere-csi-controller
  template:
    metadata:
      labels:
        app: vsphere-csi-controller
        role: vsphere-csi
    spec:
      serviceAccountName: vsphere-csi-controller
      nodeSelector:
        node-role.kubernetes.io/master: ""
      tolerations:
        - operator: "Exists"
          key: node-role.kubernetes.io/master
          effect: NoSchedule
      containers:
        - name: csi-attacher
          image: vmware/csi-attacher/csi-attacher:%s
          args:
            - "--v=4"
            - "--timeout=300s"
            - "--csi-address=$(ADDRESS)"
            - "--leader-election"
            - "--leader-election-type=leases"
          imagePullPolicy: "IfNotPresent"
          env:
            - name: ADDRESS
              value: /csi/csi.sock
          volumeMounts:
            - mountPath: /csi
              name: socket-dir
        - name: vsphere-csi-controller
          image: vmware/vsphere-csi:%s
          lifecycle:
            preStop:
              exec:
                command: ["/bin/sh", "-c", "rm -rf /var/lib/csi/sockets/pluginproxy/csi.vsphere.vmware.com"]
          args:
            - "--v=4"
          imagePullPolicy: "IfNotPresent"
          env:
            - name: CSI_ENDPOINT
              value: unix:///var/lib/csi/sockets/pluginproxy/csi.sock
            - name: CLUSTER_FLAVOR
              value: "GUEST_CLUSTER"
            - name: X_CSI_MODE
              value: "controller"
            - name: X_CSI_GC_CONFIG
              value: /etc/cloud/pvcsi-config/cns-csi.conf
            - name: PROVISION_TIMEOUT_MINUTES
              value: "4"
            - name: ATTACHER_TIMEOUT_MINUTES
              value: "4"
          volumeMounts:
            - mountPath: /etc/cloud/pvcsi-provider
              name: pvcsi-provider-volume
              readOnly: true
            - mountPath: /etc/cloud/pvcsi-config
              name: pvcsi-config-volume
              readOnly: true
            - mountPath: /var/lib/csi/sockets/pluginproxy/
              name: socket-dir
        - name: vsphere-syncer
          image: vmware/syncer:%s
          args:
            - "--v=4"
            - "--leader-election"
          imagePullPolicy: "IfNotPresent"
          env:
            - name: X_CSI_FULL_SYNC_INTERVAL_MINUTES
              value: "30"
            - name: X_CSI_GC_CONFIG
              value: /etc/cloud/pvcsi-config/cns-csi.conf
            - name: CLUSTER_FLAVOR
              value: "GUEST_CLUSTER"
          volumeMounts:
          - mountPath: /etc/cloud/pvcsi-provider
            name: pvcsi-provider-volume
            readOnly: true
          - mountPath: /etc/cloud/pvcsi-config
            name: pvcsi-config-volume
            readOnly: true
        - name: liveness-probe
          image: vmware/csi-livenessprobe/csi-livenessprobe:%s
          args:
            - "--csi-address=$(ADDRESS)"
          imagePullPolicy: "IfNotPresent"
          env:
            - name: ADDRESS
              value: /var/lib/csi/sockets/pluginproxy/csi.sock
          volumeMounts:
            - mountPath: /var/lib/csi/sockets/pluginproxy/
              name: socket-dir
        - name: csi-provisioner
          image: vmware/csi-provisioner/csi-provisioner:%s
          args:
            - "--v=4"
            - "--timeout=300s"
            - "--csi-address=$(ADDRESS)"
            - "--enable-leader-election"
            - "--leader-election-type=leases"
          imagePullPolicy: "IfNotPresent"
          env:
            - name: ADDRESS
              value: /csi/csi.sock
          volumeMounts:
            - mountPath: /csi
              name: socket-dir
      volumes:
         - name: pvcsi-provider-volume
           secret:
             secretName: pvcsi-provider-creds
         - name: pvcsi-config-volume
           configMap:
             name: pvcsi-config
         - name: socket-dir
           hostPath:
             path: /var/lib/csi/sockets/pluginproxy/csi.vsphere.vmware.com
             type: DirectoryOrCreate
---
apiVersion: storage.k8s.io/v1beta1
kind: CSIDriver
metadata:
  name: csi.vsphere.vmware.com
spec:
  attachRequired: true
  podInfoOnMount: false
---
kind: DaemonSet
apiVersion: apps/v1
metadata:
  name: vsphere-csi-node
  namespace: {{ .PVCSINamespace }}
spec:
  selector:
    matchLabels:
      app: vsphere-csi-node
  updateStrategy:
    type: "RollingUpdate"
  template:
    metadata:
      labels:
        app: vsphere-csi-node
        role: vsphere-csi
    spec:
      containers:
      - name: node-driver-registrar
        image: vmware.io/csi-node-driver-registrar:%s
        imagePullPolicy: "IfNotPresent"
        lifecycle:
          preStop:
            exec:
              command: ["/bin/sh", "-c", "rm -rf /registration/csi.vsphere.vmware.com /var/lib/kubelet/plugins_registry/csi.vsphere.vmware.com /var/lib/kubelet/plugins_registry/csi.vsphere.vmware.com-reg.sock"]
        args:
          - "--v=5"
          - "--csi-address=$(ADDRESS)"
          - "--kubelet-registration-path=$(DRIVER_REG_SOCK_PATH)"
        env:
          - name: ADDRESS
            value: /csi/csi.sock
          - name: DRIVER_REG_SOCK_PATH
            value: /var/lib/kubelet/plugins_registry/csi.vsphere.vmware.com/csi.sock
        securityContext:
          privileged: true
        volumeMounts:
          - name: plugin-dir
            mountPath: /csi
          - name: registration-dir
            mountPath: /registration
      - name: vsphere-csi-node
        image: vmware/vsphere-csi:%s
        imagePullPolicy: "IfNotPresent"
        env:
        - name: NODE_NAME
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
        - name: CSI_ENDPOINT
          value: unix:///csi/csi.sock
        - name: X_CSI_MODE
          value: "node"
        - name: X_CSI_SPEC_REQ_VALIDATION
          value: "false"
        - name: CLUSTER_FLAVOR
          value: "GUEST_CLUSTER"
        - name: X_CSI_DEBUG
          value: "true"
        args:
          - "--v=4"
        securityContext:
          privileged: true
          capabilities:
            add: ["SYS_ADMIN"]
          allowPrivilegeEscalation: true
        volumeMounts:
        - name: plugin-dir
          mountPath: /csi
        - name: pods-mount-dir
          mountPath: /var/lib/kubelet
          mountPropagation: "Bidirectional"
        - name: device-dir
          mountPath: /dev
      - name: liveness-probe
        image: vmware/csi-livenessprobe/csi-livenessprobe:%s
        args:
        - "--csi-address=$(ADDRESS)"
        imagePullPolicy: "IfNotPresent"
        env:
        - name: ADDRESS
          value: /csi/csi.sock
        volumeMounts:
        - name: plugin-dir
          mountPath: /csi
      volumes:
      - name: registration-dir
        hostPath:
          path: /var/lib/kubelet/plugins_registry
          type: DirectoryOrCreate
      - name: plugin-dir
        hostPath:
          path: /var/lib/kubelet/plugins_registry/csi.vsphere.vmware.com
          type: DirectoryOrCreate
      - name: pods-mount-dir
        hostPath:
          path: /var/lib/kubelet
          type: Directory
      - name: device-dir
        hostPath:
          path: /dev
---
apiVersion: v1
data:
  cns-csi.conf: "[GC]\nendpoint = \"{{ .SupervisorMasterEndpointHostName }}\"\nport = \"{{ .SupervisorMasterPort }}\"\nmanagedcluster-uid = \"{{ .TanzuKubernetesClusterUID }}\" \n"
kind: ConfigMap
metadata:
  name: pvcsi-config
  namespace: {{ .PVCSINamespace }}
---
`

	FakeCloudProviderAddOnTemplate = `
apiVersion: v1
kind: Namespace
metadata:
  name: vmware-system-cloud-provider
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: cloud-provider-svc-account
  namespace: vmware-system-cloud-provider
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: cloud-provider-cluster-role
rules:
- apiGroups:
  - ""
  resources:
  - configmaps
  - services
  - nodes
  - endpoints
  - secrets
  verbs:
  - get
  - watch
  - list
- apiGroups:
  - "policy"
  resources:
  - podsecuritypolicies
  verbs:
  - use
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: cloud-provider-patch-cluster-role
rules:
- apiGroups:
  - ""
  resources:
  - endpoints
  - events
  verbs:
  - create
  - update
  - replace
  - patch
- apiGroups:
  - ""
  resources:
  - services/status
  verbs:
  - patch
- apiGroups:
  - ""
  resources:
  - nodes
  - nodes/status
  verbs:
  - update
  - patch
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: cloud-provider-cluster-role-binding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cloud-provider-cluster-role
subjects:
- kind: ServiceAccount
  name: cloud-provider-svc-account
  namespace: vmware-system-cloud-provider
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: cloud-provider-patch-cluster-role-binding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cloud-provider-patch-cluster-role
subjects:
- kind: ServiceAccount
  name: cloud-provider-svc-account
  namespace: vmware-system-cloud-provider
---
apiVersion: v1
data:
  owner-reference: "{\"apiVersion\": \"{{ .TanzuKubernetesClusterAPIVersion }}\", \"kind\": \"{{ .TanzuKubernetesClusterKind }}\", \"name\": \"{{ .TanzuKubernetesClusterName }}\", \"uid\": \"{{ .TanzuKubernetesClusterUID }}\"}"
kind: ConfigMap
metadata:
  name: ccm-owner-reference
  namespace: vmware-system-cloud-provider
---
apiVersion: v1
data:
  cloud-config: ""
kind: ConfigMap
metadata:
  name: ccm-cloud-config
  namespace: vmware-system-cloud-provider
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: guest-cluster-cloud-provider
  namespace: vmware-system-cloud-provider
spec:
  selector:
    matchLabels:
      name: guest-cluster-cloud-provider
  template:
    metadata:
      labels:
        name: guest-cluster-cloud-provider
    spec:
      priorityClassName: system-cluster-critical
      containers:
      - args:
        - --controllers=service
        - --controllers=cloud-node
        - --cloud-config=/config/cloud-config
        - --cluster-name={{ .TanzuKubernetesClusterName }}
        - --cloud-provider=vsphere-paravirtual
        image: vmware/cloud-provider-vsphere-manager:%s
        imagePullPolicy: IfNotPresent
        name: guest-cluster-cloud-provider
        env:
        - name: SUPERVISOR_APISERVER_ENDPOINT_IP
          value: "{{ .SupervisorMasterEndpointIP }}"
        - name: SUPERVISOR_APISERVER_PORT
          value: "{{ .SupervisorMasterPort }}"
        volumeMounts:
        - mountPath: /config
          name: ccm-config
          readOnly: true
        - mountPath: /etc/kubernetes/guestclusters/managedcluster
          name: ccm-owner-reference
          readOnly: true
        - mountPath: /etc/cloud/ccm-provider
          name: ccm-provider
          readOnly: true
      nodeSelector:
        node-role.kubernetes.io/master: ""
      serviceAccountName: cloud-provider-svc-account
      tolerations:
      - key: node.cloudprovider.kubernetes.io/uninitialized
        value: "true"
        effect: NoSchedule
      - effect: NoSchedule
        key: node-role.kubernetes.io/master
      volumes:
      - name: ccm-config
        projected:
          sources:
          - configMap:
              items:
              - key: cloud-config
                path: cloud-config
              name: ccm-cloud-config
      - name: ccm-owner-reference
        projected:
          sources:
          - configMap:
              items:
              - key: owner-reference
                path: ownerref.json
              name: ccm-owner-reference
      - name: ccm-provider
        projected:
          sources:
          - secret:
              items:
              - key: ca.crt
                path: ca.crt
              - key: token
                path: token
              - key: namespace
                path: namespace
              name: cloud-provider-creds

`

	FakeAuthServiceAddonTemplate = `
apiVersion: v1
kind: Namespace
metadata:
  name: vmware-system-auth
---
apiVersion: v1
kind: ConfigMap
metadata:
  namespace: vmware-system-auth
  name: guest-cluster-auth-svc-public-keys
---
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: guest-cluster-auth-svc
  namespace: vmware-system-auth
  annotations:
    wellknown-key: hi
spec:
  selector:
    matchLabels:
      name: guest-cluster-auth-svc
  template:
    metadata:
      labels:
        name: guest-cluster-auth-svc
    spec:
      nodeSelector:
        node-role.kubernetes.io/master: ""
      tolerations:
      - key: node-role.kubernetes.io/master
        effect: NoSchedule
      containers:
      - command:
          - /guest-cluster-auth-service
          - /config/authsvc.yaml
        image: vmware.io/guest-cluster-auth-service:%s
        imagePullPolicy: IfNotPresent
        # The last thing the auth service does after loading the key, certs etc. is to invoke ListenAndServeTLS()
        # If we can make a GET request to the healthz endpoint, then it must be ready to serve requests.
        readinessProbe:
          httpGet:
            port: 5443
            scheme: HTTPS
            path: /healthz
            host: 127.0.0.1
          initialDelaySeconds: 5
          periodSeconds: 5
        name: guest-cluster-auth-service
        volumeMounts:
          # The secret containing the key used for TLS.
        - mountPath: /keys
          name: guest-cluster-auth-svc-key
          readOnly: true
          # The ConfigMap containing the signed certificate for TLS.
        - mountPath: /certs
          name: guest-cluster-auth-svc-cert
          readOnly: true
          # The ConfigMap containing the service configuration.
        - mountPath: /config
          name: guest-cluster-auth-svc-config
          readOnly: true
          # The ConfigMap containing the public keys to verify VC issued JWTs.
        - mountPath: /public_keys/
          name: guest-cluster-auth-svc-public-keys
          readOnly: true
      # Use the host network so the apiserver can reach the service over localhost.
      hostNetwork: true
      volumes:
      - name: guest-cluster-auth-svc-key
        secret:
          secretName: guest-cluster-auth-svc-key
      - name: guest-cluster-auth-svc-cert
        configMap:
          name: guest-cluster-auth-svc-cert
      - name: guest-cluster-auth-svc-config
        configMap:
          name: guest-cluster-auth-svc-config
      - name: guest-cluster-auth-svc-public-keys
        configMap:
          name: guest-cluster-auth-svc-public-keys
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: guest-cluster-auth-svc-config
  namespace: vmware-system-auth

data:
  authsvc.yaml: |
    endpoint: /tokenreview
    port: 5443
    certpath: /certs/cert.crt
    keypath: /keys/key.pem
    configfolderpath: /public_keys/vsphere.local.json
`

	FakeMetricsServerAddonTemplate = `
apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    k8s-app: metrics-server
  name: metrics-server
  namespace: kube-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  labels:
    k8s-app: metrics-server
    rbac.authorization.k8s.io/aggregate-to-admin: "true"
    rbac.authorization.k8s.io/aggregate-to-edit: "true"
    rbac.authorization.k8s.io/aggregate-to-view: "true"
  name: system:aggregated-metrics-reader
rules:
  - apiGroups:
      - metrics.k8s.io
    resources:
      - pods
      - nodes
    verbs:
      - get
      - list
      - watch
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  labels:
    k8s-app: metrics-server
  name: system:metrics-server
rules:
  - apiGroups:
      - ""
    resources:
      - pods
      - nodes
      - nodes/stats
      - namespaces
      - configmaps
    verbs:
      - get
      - list
      - watch
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  labels:
    k8s-app: metrics-server
  name: metrics-server-auth-reader
  namespace: kube-system
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: extension-apiserver-authentication-reader
subjects:
  - kind: ServiceAccount
    name: metrics-server
    namespace: kube-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  labels:
    k8s-app: metrics-server
  name: metrics-server:system:auth-delegator
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:auth-delegator
subjects:
  - kind: ServiceAccount
    name: metrics-server
    namespace: kube-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  labels:
    k8s-app: metrics-server
  name: system:metrics-server
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:metrics-server
subjects:
  - kind: ServiceAccount
    name: metrics-server
    namespace: kube-system
---
apiVersion: v1
kind: Service
metadata:
  labels:
    k8s-app: metrics-server
  name: metrics-server
  namespace: kube-system
spec:
  type: ClusterIp
  ports:
    - name: https
      port: 443
      protocol: TCP
      targetPort: https
  selector:
    k8s-app: metrics-server
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    k8s-app: metrics-server
  name: metrics-server
  namespace: kube-system
spec:
  selector:
    matchLabels:
      k8s-app: metrics-server
  strategy:
    rollingUpdate:
      maxUnavailable: 0
      maxSurge: 1
  template:
    metadata:
      labels:
        k8s-app: metrics-server
    spec:
      containers:
        - args:
            - --cert-dir=/tmp
            - --secure-port=4443
            - --kubelet-preferred-address-types=InternalIP,ExternalIP,Hostname
            - --kubelet-use-node-status-port
            - --kubelet-insecure-tls
            - --client-ca-file=/etc/ssl/certs/extensions-tls.crt
            - --requestheader-client-ca-file=/etc/kubernetes/pki/front-proxy-ca.crt
            - --tls-cert-file=/metrics-server-certs/tls.crt
            - --tls-private-key-file=/metrics-server-certs/tls.key
          image: 5000/vmware.io/metrics-server:%s
          imagePullPolicy: IfNotPresent
          livenessProbe:
            failureThreshold: 3
            httpGet:
              path: /livez
              port: https
              scheme: HTTPS
            periodSeconds: 10
          name: metrics-server
          ports:
            - containerPort: 4443
              name: https
              protocol: TCP
          readinessProbe:
            failureThreshold: 3
            httpGet:
              path: /readyz
              port: https
              scheme: HTTPS
            periodSeconds: 10
          securityContext:
            readOnlyRootFilesystem: true
            runAsNonRoot: true
            runAsUser: 1000
          volumeMounts:
            - mountPath: /tmp
              name: tmp-dir
            - mountPath: /metrics-server-certs
              name: guest-cluster-metrics-server-cert-path
              readOnly: true
            - mountPath: /etc/kubernetes/pki
              name: k8s-certs
              readOnly: true
            - mountPath: /etc/ssl/certs
              name: ca-certs
              readOnly: true
      nodeSelector:
        kubernetes.io/os: linux
        node-role.kubernetes.io/master: ""
      priorityClassName: system-cluster-critical
      serviceAccountName: metrics-server
      tolerations:
        - key: node-role.kubernetes.io/master
          operator: Exists
          effect: NoSchedule
      volumes:
        - emptyDir: {}
          name: tmp-dir
        - secret:
            secretName: guest-cluster-metrics-server-cert
          name: guest-cluster-metrics-server-cert-path
        - hostPath:
            path: /etc/kubernetes/pki
            type: DirectoryOrCreate
          name: k8s-certs
        - hostPath:
            path: /etc/ssl/certs
            type: DirectoryOrCreate
          name: ca-certs

---
apiVersion: apiregistration.k8s.io/v1
kind: APIService
metadata:
  labels:
    k8s-app: metrics-server
  name: v1beta1.metrics.k8s.io
spec:
  group: metrics.k8s.io
  groupPriorityMinimum: 100
  insecureSkipTLSVerify: false
  service:
    name: metrics-server
    namespace: kube-system
  version: v1beta1
  versionPriority: 100
  caBundle: test`
)

func CreateFakeVirtualMachineImage() *vmoprv1.VirtualMachineImage {
	image := FakeVirtualMachineImage(FakeDistributionVersion, FakeKubernetesVersion, FakeEtcdVersion, FakeDNSVersion, FakeCNIVersion, FakeCSIVersion, FakeCPIVersion, FakeAuthServiceVersion, FakeMetricsServerVersion)
	// Using image.GenerateName causes problems with unit tests
	// This doesn't need to be a secure prng
	// nolint:gosec
	image.Name = fmt.Sprintf("test-%d", rand.Int())
	return image
}

func CreateFakeTKR(vmi *vmoprv1.VirtualMachineImage) (*tkgv1.TanzuKubernetesRelease, []*tkgv1.TanzuKubernetesAddon) {
	tkr, tkas := NewTKR(vmi, FakeDistributionVersion, FakeKubernetesVersion, FakeEtcdVersion, FakeDNSVersion, FakeCNIVersion, FakeCSIVersion, FakeCPIVersion, FakeAuthServiceVersion, FakeMetricsServerVersion, true, true)
	return tkr, tkas
}

func CreateFakeIncompatibleVirtualMachineImage() *vmoprv1.VirtualMachineImage {
	image := FakeIncompatibleVirtualMachineImage(FakeIncompatibleDistributionVersion)
	// Using image.GenerateName causes problems with unit tests
	// This doesn't need to be a secure prng
	// nolint:gosec
	image.Name = fmt.Sprintf("incompatible-%d", rand.Int())
	return image
}

func CreateFakeIncompatibleTKR(vmi *vmoprv1.VirtualMachineImage) (*tkgv1.TanzuKubernetesRelease, []*tkgv1.TanzuKubernetesAddon) {
	return NewTKR(vmi, FakeIncompatibleDistributionVersion, FakeKubernetesVersion, FakeEtcdVersion, FakeDNSVersion, FakeCNIVersion, FakeCSIVersion, FakeCPIVersion, FakeAuthServiceVersion, FakeMetricsServerVersion, false, true)
}

func FakeVirtualMachineImage(distroVersion, kubernetesVersion, etcdVersion, coreDNSVersion, cniVersion, csiVersion, cpiVersion, authsvcVersion, metricsServerVersion string) *vmoprv1.VirtualMachineImage {
	return fakeVirtualMachineImage(distroVersion, kubernetesVersion, etcdVersion, coreDNSVersion, cniVersion, csiVersion, cpiVersion, authsvcVersion, metricsServerVersion, true, true)
}

func FakeAntreaDisabledVirtualMachineImage(distroVersion, kubernetesVersion, etcdVersion, coreDNSVersion, cniVersion, csiVersion, cpiVersion, authsvcVersion, metricsServerVersion string) *vmoprv1.VirtualMachineImage {
	return fakeVirtualMachineImage(distroVersion, FakeKubernetesVersion, FakeEtcdVersion, FakeDNSVersion, FakeCNIVersion, FakeCSIVersion, FakeCPIVersion, FakeAuthServiceVersion, FakeMetricsServerVersion, true, false)
}

func FakeIncompatibleVirtualMachineImage(distroVersion string) *vmoprv1.VirtualMachineImage {
	return fakeVirtualMachineImage(distroVersion, FakeKubernetesVersion, FakeEtcdVersion, FakeDNSVersion, FakeCNIVersion, FakeCSIVersion, FakeCPIVersion, FakeAuthServiceVersion, FakeMetricsServerVersion, false, true)
}

func addVersionToCalico(containerTag string) string {
	return fmt.Sprintf(FakeCalicoAddOnTemplate, containerTag, containerTag, containerTag, containerTag)
}

func addVersionToAntrea(containerTag string) string {
	return fmt.Sprintf(FakeAntreaAddOnTemplate, containerTag, containerTag, containerTag, containerTag)
}

func addVersionToCSI(containerTag string) string {
	return fmt.Sprintf(FakePvcsiAddOnTemplate, containerTag, containerTag, containerTag, containerTag, containerTag, containerTag, containerTag, containerTag)
}

func addVersionToCPI(containerTag string) string {
	return fmt.Sprintf(FakeCloudProviderAddOnTemplate, containerTag)
}

func addVersionToAuthSvc(containerTag string) string {
	return fmt.Sprintf(FakeAuthServiceAddonTemplate, containerTag)
}

func addVersionToMetricsServer(containerTag string) string {
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

var vmiAPIVersion, vmiKind = vmoprv1.SchemeGroupVersion.WithKind(reflect.TypeOf(vmoprv1.VirtualMachineImage{}).Name()).ToAPIVersionAndKind()

func compatibilityCondition(compatible bool) clusterv1.Condition {
	if !compatible {
		return clusterv1.Condition{
			Type:               tkgv1.ConditionCompatible,
			Status:             corev1.ConditionFalse,
			Severity:           clusterv1.ConditionSeverityWarning,
			Message:            "incompatible",
			LastTransitionTime: metav1.Now(),
		}
	}
	return clusterv1.Condition{
		Type:               tkgv1.ConditionCompatible,
		Status:             corev1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
	}
}

func NewTKR(vmi *vmoprv1.VirtualMachineImage, distroVersion, kubernetesVersion, etcdVersion, coreDNSVersion, cniVersion, csiVersion, cpiVersion, authsvcVersion, metricsServerVersion string, compatible bool, antreaEnabled bool) (*tkgv1.TanzuKubernetesRelease, []*tkgv1.TanzuKubernetesAddon) {
	distroVersion, _ = util.NormalizeSemver(distroVersion)
	vmDist := vmDistSpec(distroVersion, kubernetesVersion, etcdVersion, coreDNSVersion)

	tkr := &tkgv1.TanzuKubernetesRelease{
		ObjectMeta: metav1.ObjectMeta{
			Name:            util.TKRName(distroVersion),
			OwnerReferences: ownerRef(vmi),
		},
		Spec: tkgv1.TanzuKubernetesReleaseSpec{
			Version:           distroVersion,
			KubernetesVersion: vmDist.Kubernetes.Version,
			Repository:        vmDist.Kubernetes.ImageRepository,
			Images: []tkgv1.ContainerImage{{
				Repository: vmDist.Etcd.ImageRepository,
				Name:       common.EtcdImageName,
				Tag:        vmDist.Etcd.Version,
			}, {
				Repository: vmDist.CoreDNS.ImageRepository,
				Name:       common.CoreDNSImageName,
				Tag:        vmDist.CoreDNS.Version,
			}},
			NodeImageRef: util.ObjectReference(vmi),
		},
		Status: tkgv1.TanzuKubernetesReleaseStatus{
			Conditions: []clusterv1.Condition{compatibilityCondition(compatible)},
		},
	}
	vlabel.EnsureLabels(tkr)
	tkr.Labels[tkr.Name] = vlabel.ValueLatest
	if !compatible {
		tkr.Labels[tkgv1.LabelIncompatible] = ""
		tkr.Labels[tkr.Name] = ""
	}
	tkr.Labels = labels.Merge(tkr.Labels, vlabel.DefaultOSLabels())
	addonList := addonList(tkr, cniVersion, csiVersion, cpiVersion, authsvcVersion, metricsServerVersion, antreaEnabled)

	return tkr, addonList
}

func ownerRef(vmi *vmoprv1.VirtualMachineImage) []metav1.OwnerReference {
	if vmi == nil {
		return nil
	}
	return []metav1.OwnerReference{{
		APIVersion: vmiAPIVersion,
		Kind:       vmiKind,
		Name:       vmi.Name,
		UID:        vmi.UID,
	}}
}

func vmDistSpec(distVersion string, kubernetesVersion, etcdVersion, coreDNSVersion string) manifests.VirtualMachineDistributionSpec {
	return manifests.VirtualMachineDistributionSpec{
		Distribution: manifests.DistributionVersion{
			Version: distVersion,
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
}

func addonList(tkr *tkgv1.TanzuKubernetesRelease, cniVersion, csiVersion, cpiVersion, authsvcVersion, metricsServerVersion string, antreaEnabled bool) []*tkgv1.TanzuKubernetesAddon {
	tanzuKubernetesAddons := []*tkgv1.TanzuKubernetesAddon{
		genAddon(tkr, addons.Calico, cniVersion, addVersionToCalico),
		genAddon(tkr, addons.VMwareGuestClusterCSI, csiVersion, addVersionToCSI),
		genAddon(tkr, addons.VMwareGuestClusterCPI, cpiVersion, addVersionToCPI),
		genAddon(tkr, addons.VMwareGuestClusterAuthsvc, authsvcVersion, addVersionToAuthSvc),
		genAddon(tkr, addons.VMwareGuestClusterMetricsServer, metricsServerVersion, addVersionToMetricsServer),
	}
	if antreaEnabled {
		tanzuKubernetesAddons = append(tanzuKubernetesAddons, genAddon(tkr, addons.Antrea, cniVersion, addVersionToAntrea))
	}
	return tanzuKubernetesAddons
}

func genAddon(tkr *tkgv1.TanzuKubernetesRelease, addonName addons.AddonName, addonVersion string, templateGen func(containerTag string) string) *tkgv1.TanzuKubernetesAddon {
	res := &tkgv1.ManifestResource{
		Version: addonVersion,
		Type:    "inline",
		Value:   templateGen(addonVersion),
	}
	return &tkgv1.TanzuKubernetesAddon{
		ObjectMeta: metav1.ObjectMeta{
			Name:   addonObjectName(string(addonName), addonVersion, res),
			Labels: map[string]string{util.AddonTKRLabel(tkr.Name): ""},
		},
		Spec: tkgv1.TanzuKubernetesAddonSpec{
			AddonName: string(addonName),
			Version:   addonVersion,
			Resource:  res,
		},
	}
}

func addonObjectName(addonName, addonVersion string, resource *tkgv1.ManifestResource) string {
	hash := manifestHash(resource)
	return fmt.Sprintf("%s-%s-%s", addonName, addonVersion, hash)
}

func manifestHash(resource *tkgv1.ManifestResource) string {
	hash := sha256.New()
	jsonEncoder := json.NewEncoder(hash)
	_ = jsonEncoder.Encode(resource)
	return hex.EncodeToString(hash.Sum(nil)[:5])
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

	// TODO: after antrea is made as default CNI, change to calicoEnabled
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
					VMwareSystemCompatibilityOffering:            compatibilityOffering,
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
			Annotations: map[string]string{
				manifests.VirtualMachineDistributionProperty: string(rawJSON),
				cniCalicoAddOnAnnotation:                     string(rawCniCalicoJSON),
				csiAddOnAnnotation:                           string(rawCsiJSON),
				cpAddOnAnnotation:                            string(rawCloudProviderJSON),
				authsvcAddOnAnnotation:                       string(rawAuthSvcJSON),
				metricsServerAddOnAnnotation:                 string(rawMetricsServerJSON),
				VMwareSystemCompatibilityOffering:            compatibilityOffering,
			},
		},
		Spec: vmoprv1.VirtualMachineImageSpec{
			ProductInfo: vmoprv1.VirtualMachineImageProductInfo{
				FullVersion: distroVersion,
			},
		},
	}
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

func NewKubeadmControlPlane(name, namespace string) *kcpv1.KubeadmControlPlane {
	return &kcpv1.KubeadmControlPlane{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func NewWCPMachine(name, namespace string) *infrav1.WCPMachine {
	return &infrav1.WCPMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func NewMachine(name, namespace string) *clusterv1.Machine {
	return &clusterv1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func NewKubeadmConfig(name, namespace string) *bootstrapv1.KubeadmConfig {
	return &bootstrapv1.KubeadmConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func NewWCPMachineTemplate(name, namespace string) *infrav1.WCPMachineTemplate {
	return &infrav1.WCPMachineTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func NewKubeadmConfigTemplate(name, namespace string) *bootstrapv1.KubeadmConfigTemplate {
	return &bootstrapv1.KubeadmConfigTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func NewMachineDeployment(name, namespace string) *clusterv1.MachineDeployment {
	return &clusterv1.MachineDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func NewJob(name, namespace string) *batchv1.Job {
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func NewNetworkConfigMap() corev1.ConfigMap {
	return corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vmoperator-network-config",
			Namespace: "vmware-system-vmop",
		},
		Data: map[string]string{
			"ntpservers": "time.vmware.com",
		},
	}
}
