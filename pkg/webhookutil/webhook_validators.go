/*
Copyright 2022 The Kubernetes Authors.

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

package webhookutil

import (
	"reflect"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func EnsureFieldExists(value string, name string, allErrs field.ErrorList) field.ErrorList {
	if value == "" {
		allErrs = append(allErrs, field.Required(field.NewPath("spec", name), name))
	}
	return allErrs
}

func EnsureAtLeastOneFieldExists(value1 string, value2 string, name string, allErrs field.ErrorList) field.ErrorList {
	if value1 == "" && value2 == "" {
		allErrs = append(allErrs, field.Required(field.NewPath("spec", name), name))
	}
	return allErrs
}

func EnsureEqualStrings(newString string, oldString string, name string, allErrs field.ErrorList) field.ErrorList {
	if newString != oldString {
		allErrs = append(allErrs, field.Forbidden(field.NewPath("spec", name), name))
	}
	return allErrs
}

func EnsureIntFieldsAreNotNegative(newInt int64, name string, allErrs field.ErrorList) field.ErrorList {
	if newInt < 0 {
		allErrs = append(allErrs, field.Forbidden(field.NewPath("spec", name), name))
	}
	return allErrs
}

func EnsureEqualMapStringString(newMap *map[string]string, oldMap *map[string]string, name string, allErrs field.ErrorList) field.ErrorList {
	if oldMap == nil && newMap == nil {
		return allErrs
	}
	if newMap == nil || oldMap == nil {
		allErrs = append(allErrs, field.Forbidden(field.NewPath("spec", name), name))
	}
	if !reflect.DeepEqual(oldMap, newMap) {
		allErrs = append(allErrs, field.Forbidden(field.NewPath("spec", name), name))
	}
	return allErrs
}

func AggregateObjErrors(gk schema.GroupKind, name string, allErrs field.ErrorList) error {
	if len(allErrs) == 0 {
		return nil
	}

	return errors.NewInvalid(
		gk,
		name,
		allErrs,
	)
}
