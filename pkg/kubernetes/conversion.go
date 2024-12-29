/*
Copyright 2024 The Forge Authors.

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

// Package kubernetes implements conversion utilities.
package kubernetes

import (
	"context"
	"sort"
	"strings"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	buildv1 "github.com/forge-build/forge/pkg/api/v1alpha1"
)

const (
	// DataAnnotation is the annotation that conversion webhooks
	// use to retain the data in case of down-conversion from the hub.
	DataAnnotation = "forge.build/conversion-data"
)

var (
	contract = buildv1.GroupVersion.String()
)

// UpdateReferenceAPIContract takes a client and object reference, queries the API Server for
// the Custom Resource Definition and looks which one is the stored version available.
//
// The object passed as input is modified in place if an updated compatible version is found.
// NOTE: This version depends on CRDs being named correctly as defined by contract.CalculateCRDName.
func UpdateReferenceAPIContract(ctx context.Context, c client.Client, ref *corev1.ObjectReference) error {
	gvk := ref.GroupVersionKind()

	metadata, err := GetGVKMetadata(ctx, c, gvk)
	if err != nil {
		return errors.Wrapf(err, "failed to update apiVersion in ref")
	}

	chosen, err := getLatestAPIVersionFromContract(metadata)
	if err != nil {
		return errors.Wrapf(err, "failed to update apiVersion in ref")
	}

	// Modify the GroupVersionKind with the new version.
	if gvk.Version != chosen {
		gvk.Version = chosen
		ref.SetGroupVersionKind(gvk)
	}

	return nil
}

func getLatestAPIVersionFromContract(metadata metav1.Object) (string, error) {
	labels := metadata.GetLabels()

	// If there is no label, return early without changing the reference.
	supportedVersions, ok := labels[contract]
	if !ok || supportedVersions == "" {
		return "", errors.Errorf("cannot find any versions matching contract %q for CRD %v as contract version label(s) are either missing or empty (see https://cluster-api.sigs.k8s.io/developer/providers/contracts.html#api-version-labels)", contract, metadata.GetName())
	}

	// Pick the latest version in the slice and validate it.
	kubeVersions := KubeAwareAPIVersions(strings.Split(supportedVersions, "_"))
	sort.Sort(kubeVersions)
	return kubeVersions[len(kubeVersions)-1], nil
}
