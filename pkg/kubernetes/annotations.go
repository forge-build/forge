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

// Package kubernetes implements annotation helper functions.
package kubernetes

import (
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	buildv1 "github.com/forge-build/forge/pkg/api/v1alpha1"
)

// IsExternallyManaged returns true if the object has the `managed-by` annotation.
func IsExternallyManaged(o metav1.Object) bool {
	return hasAnnotation(o, buildv1.ManagedByAnnotation)
}

// HasWithPrefix returns true if at least one of the annotations has the prefix specified.
func HasWithPrefix(prefix string, annotations map[string]string) bool {
	for key := range annotations {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

// AddAnnotations sets the desired annotations on the object and returns true if the annotations have changed.
func AddAnnotations(o metav1.Object, desired map[string]string) bool {
	if len(desired) == 0 {
		return false
	}
	annotations := o.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	hasChanged := false
	for k, v := range desired {
		if cur, ok := annotations[k]; !ok || cur != v {
			annotations[k] = v
			hasChanged = true
		}
	}
	o.SetAnnotations(annotations)
	return hasChanged
}

// hasTruthyAnnotationValue returns true if the object has an annotation with a value that is not "false".
func hasTruthyAnnotationValue(o metav1.Object, annotation string) bool {
	annotations := o.GetAnnotations()
	if annotations == nil {
		return false
	}
	if val, ok := annotations[annotation]; ok {
		return val != "false"
	}
	return false
}
