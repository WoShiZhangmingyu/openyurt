/*
Copyright 2023 The OpenYurt Authors.

Licensed under the Apache License, Version 2.0 (the License);
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an AS IS BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	unitv1beta1 "github.com/openyurtio/openyurt/pkg/apis/apps/v1beta1"
	"github.com/openyurtio/openyurt/pkg/apis/iot/v1alpha1"
	util "github.com/openyurtio/openyurt/pkg/yurtmanager/controller/platformadmin/utils"
)

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type.
func (webhook *PlatformAdminHandler) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	platformAdmin, ok := obj.(*v1alpha1.PlatformAdmin)
	if !ok {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("expected a PlatformAdmin but got a %T", obj))
	}

	//validate
	if allErrs := webhook.validate(ctx, platformAdmin); len(allErrs) > 0 {
		return nil, apierrors.NewInvalid(v1alpha1.GroupVersion.WithKind("PlatformAdmin").GroupKind(), platformAdmin.Name, allErrs)
	}

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type.
func (webhook *PlatformAdminHandler) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	newPlatformAdmin, ok := newObj.(*v1alpha1.PlatformAdmin)
	if !ok {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("expected a PlatformAdmin but got a %T", newObj))
	}
	oldPlatformAdmin, ok := oldObj.(*v1alpha1.PlatformAdmin)
	if !ok {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("expected a PlatformAdmin but got a %T", oldObj))
	}

	// validate
	newErrorList := webhook.validate(ctx, newPlatformAdmin)
	oldErrorList := webhook.validate(ctx, oldPlatformAdmin)
	if allErrs := append(newErrorList, oldErrorList...); len(allErrs) > 0 {
		return nil, apierrors.NewInvalid(v1alpha1.GroupVersion.WithKind("PlatformAdmin").GroupKind(), newPlatformAdmin.Name, allErrs)
	}
	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type.
func (webhook *PlatformAdminHandler) ValidateDelete(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func (webhook *PlatformAdminHandler) validate(ctx context.Context, platformAdmin *v1alpha1.PlatformAdmin) field.ErrorList {
	// verify that the poolname nodepool
	if nodePoolErrs := webhook.validatePlatformAdminWithNodePools(ctx, platformAdmin); nodePoolErrs != nil {
		return nodePoolErrs
	}
	return nil
}

func (webhook *PlatformAdminHandler) validatePlatformAdminWithNodePools(ctx context.Context, platformAdmin *v1alpha1.PlatformAdmin) field.ErrorList {
	// verify that the poolnames are right nodepool names
	nodePools := &unitv1beta1.NodePoolList{}
	if err := webhook.Client.List(ctx, nodePools); err != nil {
		return field.ErrorList{
			field.Invalid(field.NewPath("spec", "nodePools"), platformAdmin.Spec.NodePools, "can not list nodepools, cause"+err.Error()),
		}
	}

	nodePoolMap := make(map[string]bool)
	for _, nodePool := range nodePools.Items {
		nodePoolMap[nodePool.ObjectMeta.Name] = true
	}

	invalidPools := []string{}
	for _, poolName := range platformAdmin.Spec.NodePools {
		if !nodePoolMap[poolName] {
			invalidPools = append(invalidPools, poolName)
		}
	}
	if len(invalidPools) > 0 {
		return field.ErrorList{
			field.Invalid(field.NewPath("spec", "nodePools"), invalidPools, "can not find the nodepools"),
		}
	}

	// verify that no other platformadmin in the nodepools
	var platformadmins v1alpha1.PlatformAdminList
	if err := webhook.Client.List(ctx, &platformadmins); err != nil {
		return field.ErrorList{
			field.Invalid(field.NewPath("spec", "nodePools"), platformAdmin.Spec.NodePools, "can not list platformadmins, cause"+err.Error()),
		}
	}

	for _, other := range platformadmins.Items {
		if platformAdmin.Name != other.Name {
			for _, poolName := range platformAdmin.Spec.NodePools {
				if util.Contains(other.Spec.NodePools, poolName) {
					return field.ErrorList{
						field.Invalid(field.NewPath("spec", "nodePools"), poolName, "already used by other platformadmin instance"),
					}
				}
			}
		}
	}

	return nil
}
