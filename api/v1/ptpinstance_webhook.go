/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2022,2025 Wind River Systems, Inc. */

package v1

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// Webhook response reasons
const PtpInstanceAllowedReason string = "allowed to be admitted"

// log is for logging in this package.
var ptpinstancelog = logf.Log.WithName("ptpinstance-resource")

func (r *PtpInstance) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithDefaulter(&PtpInstanceCustomDefaulter{}).
		WithValidator(&PtpInstanceCustomValidator{}).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-starlingx-windriver-com-v1-ptpinstance,mutating=true,failurePolicy=fail,sideEffects=None,groups=starlingx.windriver.com,resources=ptpinstances,verbs=create;update,versions=v1,name=mptpinstance.kb.io,admissionReviewVersions=v1,timeoutSeconds=30

type PtpInstanceCustomDefaulter struct{}

var _ webhook.CustomDefaulter = &PtpInstanceCustomDefaulter{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (d *PtpInstanceCustomDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	ptpInstance, ok := obj.(*PtpInstance)
	if !ok {
		return fmt.Errorf("expected a PtpInstance object but got %T", obj)
	}
	ptpinstancelog.Info("default", "name", ptpInstance.Name)
	return nil
}

// Validates an incoming resource update/create request.  The intent of this validation is to perform only the
// minimum amount of validation which should normally be done by the CRD validation schema, but until kubebuilder
// supports the necessary validation annotations we need to do this in a webhook.  All other validation is left
// to the system API and any errors generated by that API will be reported in the resource status and events.
func (r *PtpInstance) validatePtpInstance() error {
	present := make(map[string]bool)
	for _, parameter := range r.Spec.InstanceParameters {

		//TODO check if '=' exists, and only one, not the first character
		delim := "="
		if strings.Count(parameter, delim) != 1 || parameter[0:1] == delim {
			msg := fmt.Sprintf("Invalid parameter %s. Parameters must come in the form <parameterKey>=<parameterValue>.",
				parameter)
			return errors.New(msg)
		}
		key := strings.TrimSpace(strings.Split(parameter, delim)[0])

		if _, ok := present[key]; ok {
			msg := fmt.Sprintf("duplicate parameter keys are not allowed for %s.",
				parameter)
			return errors.New(msg)
		}
		present[key] = true
	}

	ptpinstancelog.Info(PtpInstanceAllowedReason)
	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:verbs=create;update,path=/validate-starlingx-windriver-com-v1-ptpinstance,mutating=false,failurePolicy=fail,sideEffects=None,groups=starlingx.windriver.com,resources=ptpinstances,versions=v1,name=vptpinstance.kb.io,admissionReviewVersions=v1,timeoutSeconds=30

type PtpInstanceCustomValidator struct{}

var _ webhook.CustomValidator = &PtpInstanceCustomValidator{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (v *PtpInstanceCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	ptpInstance, ok := obj.(*PtpInstance)
	if !ok {
		return nil, fmt.Errorf("expected a PtpInstance object but got %T", obj)
	}
	ptpinstancelog.Info("validate create", "name", ptpInstance.Name)
	return nil, ptpInstance.validatePtpInstance()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (d *PtpInstanceCustomValidator) ValidateUpdate(cxt context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	ptpInstance, ok := newObj.(*PtpInstance)
	if !ok {
		return nil, fmt.Errorf("expected a PtpInstance object but got %T", newObj)
	}
	ptpinstancelog.Info("validate update", "name", ptpInstance.Name)
	return nil, ptpInstance.validatePtpInstance()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (d *PtpInstanceCustomValidator) ValidateDelete(cxt context.Context, obj runtime.Object) (admission.Warnings, error) {
	ptpInstance, ok := obj.(*PtpInstance)
	if !ok {
		return nil, fmt.Errorf("expected a PtpInstance object but got %T", obj)
	}
	ptpinstancelog.Info("validate delete", "name", ptpInstance.Name)
	// TODO(user): fill in your validation logic upon object deletion.
	return nil, nil
}
