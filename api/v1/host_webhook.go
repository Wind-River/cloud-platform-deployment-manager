/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019-2022, 2024-2025 Wind River Systems, Inc. */

package v1

import (
	"context"
	"errors"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// Webhook response reasons
const HostAllowedReason string = "allowed to be admitted"

// log is for logging in this package.
var hostlog = logf.Log.WithName("host-resource")

func (r *Host) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithDefaulter(&HostCustomDefaulter{}).
		WithValidator(&HostCustomValidator{}).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-starlingx-windriver-com-v1-host,mutating=true,failurePolicy=fail,sideEffects=None,groups=starlingx.windriver.com,resources=hosts,verbs=create;update,versions=v1,name=mhost.kb.io,admissionReviewVersions=v1,timeoutSeconds=30

type HostCustomDefaulter struct{}

var _ webhook.CustomDefaulter = &HostCustomDefaulter{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (d *HostCustomDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	host, ok := obj.(*Host)
	if !ok {
		return fmt.Errorf("expected a Host object but got %T", obj)
	}
	hostlog.Info("default", "name", host.Name)
	return nil
}

func (r *Host) validateMatchBMInfo() error {
	if r.Spec.Match.BoardManagement.Address == nil {
		return errors.New("board management address must be supplied in match criteria")
	}

	return nil
}

func (r *Host) validateMatchDMIInfo() error {
	if r.Spec.Match.DMI.SerialNumber == nil || r.Spec.Match.DMI.AssetTag == nil {
		return errors.New("DMI Serial Number or Asset Tag must be supplied in match criteria")
	}

	return nil
}

func (r *Host) validateMatchInfo() error {
	match := r.Spec.Match

	if match.BootMAC == nil && match.BoardManagement == nil && match.DMI == nil {
		return errors.New("host must be configured with at least 1 match criteria")
	}

	if match.BoardManagement != nil {
		err := r.validateMatchBMInfo()
		if err != nil {
			return err
		}
	}

	if match.DMI != nil {
		err := r.validateMatchDMIInfo()
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Host) validateHost() error {
	if r.Spec.Match != nil {
		err := r.validateMatchInfo()
		if err != nil {
			return err
		}
	}
	hostlog.Info(HostAllowedReason)
	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:verbs=create;update,path=/validate-starlingx-windriver-com-v1-host,mutating=false,failurePolicy=fail,sideEffects=None,groups=starlingx.windriver.com,resources=hosts,versions=v1,name=vhost.kb.io,admissionReviewVersions=v1,timeoutSeconds=30

type HostCustomValidator struct{}

var _ webhook.CustomValidator = &HostCustomValidator{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (v *HostCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	host, ok := obj.(*Host)
	if !ok {
		return nil, fmt.Errorf("expected a Host object but got %T", obj)
	}
	hostlog.Info("validate create", "name", host.Name)
	return nil, host.validateHost()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (v *HostCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	host, ok := newObj.(*Host)
	if !ok {
		return nil, fmt.Errorf("expected a Host object but got %T", newObj)
	}
	hostlog.Info("validate update", "name", host.Name)
	return nil, host.validateHost()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (v *HostCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	host, ok := obj.(*Host)
	if !ok {
		return nil, fmt.Errorf("expected a Host object but got %T", obj)
	}

	hostlog.Info("validate delete", "name", host.Name)
	return nil, nil
}
