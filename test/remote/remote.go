// Copyright (c) 2019 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package remote

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/yaml"

	//"sigs.k8s.io/cluster-api-provider-vsphere/test/builder"
	"time"

	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	//"gitlab.eng.vmware.com/core-build/guest-cluster-controller/controllers/util/encoding"
	//"gitlab.eng.vmware.com/core-build/guest-cluster-controller/controllers/util/kubeconfig"
)

// NewClusterClient creates a new client for the provided cluster.
func NewClusterClient(ctx context.Context, c client.Client, namespace, clusterName string, timeout time.Duration) (client.Client, error) {
	restConfig, err := GetRESTConfig(ctx, c, namespace, clusterName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create client configuration for Cluster %s/%s", namespace, clusterName)
	}
	restConfig.Timeout = timeout

	return client.New(restConfig, client.Options{})
}

// GetRESTConfig returns the *rest.Config for the provided cluster.
func GetRESTConfig(ctx context.Context, c client.Client, namespace, clusterName string) (*rest.Config, error) {
	secret, err := getSecret(ctx, c, namespace, clusterName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to retrieve kubeconfig secret for Cluster %s/%s", namespace, clusterName)
	}

	data, err := fromSecret(secret)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get kubeconfig from secret for Cluster %s/%s", namespace, clusterName)
	}

	return clientcmd.RESTConfigFromKubeConfig(data)
}

// ApplyYAML calls ApplyYAMLWithNamespace with an empty namespace.
// TODO: rename this function to CreateYAML() to avoid confusion
func ApplyYAML(ctx context.Context, c client.Client, data []byte) error {
	return ApplyYAMLWithNamespace(ctx, c, data, "")
}

// ApplyYAMLWithNamespace applies the provided YAML as unstructured data with
// the given client.
// The data may be a single YAML document or multidoc YAML
// This function is idempotent.
// When a non-empty namespace is provided then all objects are assigned the
// the namespace prior to being created.
func ApplyYAMLWithNamespace(ctx context.Context, c client.Client, data []byte, namespace string) error {
	return ForEachObjectInYAML(ctx, c, data, namespace, func(ctx context.Context, c client.Client, obj *unstructured.Unstructured) error {
		// Create the object on the API server.
		if err := c.Create(ctx, obj); err != nil {
			// The create call is idempotent, so if the object already exists
			// then do not consider it to be an error.
			if !apierrors.IsAlreadyExists(err) {
				return errors.Wrapf(
					err,
					"failed to create object %s %s/%s",
					obj.GroupVersionKind(),
					obj.GetNamespace(),
					obj.GetName())
			}
		}
		return nil
	})
}

// DeleteYAML calls DeleteYAMLWithNamespace with an empty namespace.
func DeleteYAML(ctx context.Context, c client.Client, data []byte) error {
	return DeleteYAMLWithNamespace(ctx, c, data, "")
}

// DeleteYAMLWithNamespace deletes the provided YAML as unstructured data with
// the given client.
// The data may be a single YAML document or multidoc YAML.
// When a non-empty namespace is provided then all objects are assigned the
// the namespace prior to any other actions being performed with or to the
// object.
func DeleteYAMLWithNamespace(ctx context.Context, c client.Client, data []byte, namespace string) error {
	return ForEachObjectInYAML(ctx, c, data, namespace, func(ctx context.Context, c client.Client, obj *unstructured.Unstructured) error {
		// Delete the object on the API server.
		if err := c.Delete(ctx, obj); err != nil {
			// The delete call is idempotent, so if the object does not
			// exist, then do not consider it to be an error.
			if !apierrors.IsNotFound(err) {
				return errors.Wrapf(
					err,
					"failed to delete object %s %s/%s",
					obj.GroupVersionKind(),
					obj.GetNamespace(),
					obj.GetName())
			}
		}
		return nil
	})
}

// ExistsYAML calls ExistsYAMLWithNamespace with an empty namespace.
func ExistsYAML(ctx context.Context, c client.Client, data []byte) error {
	return ExistsYAMLWithNamespace(ctx, c, data, "")
}

// ExistsYAMLWithNamespace verifies each object in the provided YAML exists on
// the API server.
// The data may be a single YAML document or multidoc YAML.
// When a non-empty namespace is provided then all objects are assigned the
// the namespace prior to any other actions being performed with or to the
// object.
// A nil error is returned if all objects exist.
func ExistsYAMLWithNamespace(ctx context.Context, c client.Client, data []byte, namespace string) error {
	return ForEachObjectInYAML(ctx, c, data, namespace, func(ctx context.Context, c client.Client, obj *unstructured.Unstructured) error {
		key := client.ObjectKeyFromObject(obj)
		//if err != nil {
		//	return errors.Wrapf(
		//		err,
		//		"failed to get object key for %s %s/%s",
		//		obj.GroupVersionKind(),
		//		obj.GetNamespace(),
		//		obj.GetName())
		//}
		if err := c.Get(ctx, key, obj); err != nil {
			return errors.Wrapf(
				err,
				"failed to find %s %s/%s",
				obj.GroupVersionKind(),
				obj.GetNamespace(),
				obj.GetName())
		}
		return nil
	})
}

// DoesNotExistYAML calls DoesNotExistYAMLWithNamespace with an empty namespace.
func DoesNotExistYAML(ctx context.Context, c client.Client, data []byte) (bool, error) {
	return DoesNotExistYAMLWithNamespace(ctx, c, data, "")
}

// DoesNotExistYAMLWithNamespace verifies each object in the provided YAML no
// longer exists on the API server.
// The data may be a single YAML document or multidoc YAML.
// When a non-empty namespace is provided then all objects are assigned the
// the namespace prior to any other actions being performed with or to the
// object.
// A boolean true is returned if none of the objects exist.
// An error is returned if the Get call returns an error other than
// 404 NotFound.
func DoesNotExistYAMLWithNamespace(ctx context.Context, c client.Client, data []byte, namespace string) (bool, error) {
	found := true
	err := ForEachObjectInYAML(ctx, c, data, namespace, func(ctx context.Context, c client.Client, obj *unstructured.Unstructured) error {
		key := client.ObjectKeyFromObject(obj)
		//if err != nil {
		//	return errors.Wrapf(
		//		err,
		//		"failed to get object key for %s %s/%s",
		//		obj.GroupVersionKind(),
		//		obj.GetNamespace(),
		//		obj.GetName())
		//}
		if err := c.Get(ctx, key, obj); err != nil {
			found = false
			if !apierrors.IsNotFound(err) {
				return errors.Wrapf(
					err,
					"failed to find %s %s/%s",
					obj.GroupVersionKind(),
					obj.GetNamespace(),
					obj.GetName())
			}
			return nil
		}
		return nil
	})
	return found, err
}

// ForEachObjectInYAMLActionFunc is a function that is executed against each
// object found in a YAML document.
// When a non-empty namespace is provided then the object is assigned the
// namespace prior to any other actions being performed with or to the object.
type ForEachObjectInYAMLActionFunc func(context.Context, client.Client, *unstructured.Unstructured) error

// ForEachObjectInYAML excutes actionFn for each object in the provided YAML.
// If an error is returned then no further objects are processed.
// The data may be a single YAML document or multidoc YAML.
// When a non-empty namespace is provided then all objects are assigned the
// the namespace prior to any other actions being performed with or to the
// object.
func ForEachObjectInYAML(
	ctx context.Context,
	c client.Client,
	data []byte,
	namespace string,
	actionFn ForEachObjectInYAMLActionFunc) error {

	chanObj, chanErr := DecodeYAML(data)
	for {
		select {
		case obj := <-chanObj:
			if obj == nil {
				return nil
			}
			if namespace != "" {
				obj.SetNamespace(namespace)
			}
			if err := actionFn(ctx, c, obj); err != nil {
				return err
			}
		case err := <-chanErr:
			if err == nil {
				return nil
			}
			return errors.Wrap(err, "received error while decoding yaml")
		}
	}
}

func getSecret(ctx context.Context, c client.Client, namespace, clusterName string) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	secretKey := client.ObjectKey{
		Namespace: namespace,
		Name:      fmt.Sprintf("%s-kubeconfig", clusterName),
	}

	if err := c.Get(ctx, secretKey, secret); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, errors.New("secret not found")
		}
		return nil, err
	}

	return secret, nil
}

func fromSecret(secret *corev1.Secret) ([]byte, error) {
	data, ok := secret.Data["value"]
	if !ok {
		return nil, errors.New("missing value in secret")
	}
	return data, nil
}

// DecodeYAML unmarshals a YAML document or multidoc YAML as unstructured
// objects, placing each decoded object into a channel.
func DecodeYAML(data []byte) (<-chan *unstructured.Unstructured, <-chan error) {

	var (
		chanErr        = make(chan error)
		chanObj        = make(chan *unstructured.Unstructured)
		multidocReader = utilyaml.NewYAMLReader(bufio.NewReader(bytes.NewReader(data)))
	)

	go func() {
		defer close(chanErr)
		defer close(chanObj)

		// Iterate over the data until Read returns io.EOF. Every successful
		// read returns a complete YAML document.
		for {
			buf, err := multidocReader.Read()
			if err != nil {
				if err == io.EOF {
					return
				}
				chanErr <- errors.New("failed to read yaml data")
				return
			}

			// Do not use this YAML doc if it is unkind.
			var typeMeta runtime.TypeMeta
			if err := yaml.Unmarshal(buf, &typeMeta); err != nil {
				continue
			}
			if typeMeta.Kind == "" {
				continue
			}

			// Define the unstructured object into which the YAML document will be
			// unmarshaled.
			obj := &unstructured.Unstructured{
				Object: map[string]interface{}{},
			}

			// Unmarshal the YAML document into the unstructured object.
			if err := yaml.Unmarshal(buf, &obj.Object); err != nil {
				chanErr <- errors.New("failed to unmarshal yaml data")
				return
			}

			// Place the unstructured object into the channel.
			chanObj <- obj
		}
	}()

	return chanObj, chanErr
}