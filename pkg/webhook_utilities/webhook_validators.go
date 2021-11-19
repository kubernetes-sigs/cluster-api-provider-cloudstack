package webhook_utilities

import (
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

func EnsureFieldsAreEqual(new string, old string, name string, allErrs field.ErrorList) field.ErrorList {
	if new != old {
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
