// Copyright (c) 2019 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package manifests

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	k8sClient "sigs.k8s.io/controller-runtime/pkg/client"

	vmoprv1 "github.com/vmware-tanzu/vm-operator-api/api/v1alpha1"
)

const (
	// VirtualMachineDistributionProperty is the OVF property name for the
	// distribution information
	VirtualMachineDistributionProperty = "vmware-system.guest.kubernetes.distribution.image.version"

	// CompatibilityOfferingProperty is the OVF property name for compatibility information
	CompatibilityOfferingProperty = "vmware-system.compatibilityoffering"
)

// VirtualMachineDistribution contains the image name and distribution versions
// that are associated with it
type VirtualMachineDistribution struct {
	Image *vmoprv1.VirtualMachineImage `json:"image"`
	VirtualMachineDistributionSpec
}

// VirtualMachineDistributionSpec contains the resourced associated with a
// VMOperator VirtualMachineImage - these are parsed out of the annotation
// property defined in VirtualMachineDistributionProperty
type VirtualMachineDistributionSpec struct {
	Distribution DistributionVersion `json:"distribution"`
	Kubernetes   ImageVersion        `json:"kubernetes"`
	Etcd         ImageVersion        `json:"etcd"`
	CoreDNS      ImageVersion        `json:"coredns"`
	// XXX (GCM-2786, 2788): Remove compatibility hack
	CompatibleDay0   CompatibilityHack `json:"compatibility-7.0.0.10100"`
	CompatibleMonth1 CompatibilityHack `json:"compatibility-7.0.0.10200"`
	CompatibleMonth2 CompatibilityHack `json:"compatibility-7.0.0.10300"`
	CompatibleMonth3 CompatibilityHack `json:"compatibility-VC-7.0.0.1-MP3"`
}

// CompatibilityHack is a hacky way to ensure that we don't try to use an image we're not compatible with. It's needed
// because our real compatibility checking logic isn't in place yet (GCM-2256)
// XXX (GCM-2786, GCM-2788): Remove compatibility hack
type CompatibilityHack struct {
	Compatible string `json:"isCompatible"`
}

// This compatibility check was added as part of GCM-2729.
// XXX (GCM-2786, GCM-2788): Remove compatibility hack
func (c CompatibilityHack) IsCompatible() bool {
	return c.Compatible == "true"
}

// DistributionVersion contains a simple VirtualMachineImage distribution
// version
type DistributionVersion struct {
	Version string `json:"version"`
}

// ImageVersion contains the image repository and version associated with
// container images on a VirtualMachineImage
type ImageVersion struct {
	ImageRepository string `json:"imageRepository"`
	Version         string `json:"version"`
}

type ContextWithLogger interface {
	context.Context
	GetLogger() logr.Logger
}

// FindDistributionByVersion leverages a Kubernetes client to search for
// VMOperator VirtualMachineImages that include distribution metadata that
// matches to the searched version
func FindDistributionByVersion(ctx ContextWithLogger, client k8sClient.Client, version string) (*VirtualMachineDistribution, error) {
	images := &vmoprv1.VirtualMachineImageList{}
	// VirtualMachineImages are cluster-scoped and do not need a namespace
	err := client.List(ctx, images, &k8sClient.ListOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "unable to retrieve virtual machine images")
	}

	for _, image := range images.Items {
		rawJSON, ok := image.ObjectMeta.Annotations[VirtualMachineDistributionProperty]
		if !ok {
			// Image doesn't have our annotation - we can't select it, so we move on
			continue
		}

		distVersions := VirtualMachineDistributionSpec{}
		if err := json.Unmarshal([]byte(rawJSON), &distVersions); err != nil {
			ctx.GetLogger().V(4).Info("Invalid annotation on VirtualMachineImage", "image.Name", image.Name, VirtualMachineDistributionProperty, rawJSON, "err", err)
			// Image does have our annotation but we can't parse it - we can't select it, so we move on
			continue
		}

		if distVersions.Distribution.Version == version {
			return &VirtualMachineDistribution{
				Image:                          image.DeepCopy(),
				VirtualMachineDistributionSpec: distVersions,
			}, nil
		}
	}

	return nil, fmt.Errorf("unable to find VirtualMachineImage with distribution version %q", version)
}
