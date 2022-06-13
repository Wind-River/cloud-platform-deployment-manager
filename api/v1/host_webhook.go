/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019-2022 Wind River Systems, Inc. */

package v1

import (
	"errors"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// Webhook response reasons
const HostAllowedReason string = "allowed to be admitted"

// log is for logging in this package.
var hostlog = logf.Log.WithName("host-resource")

func (r *Host) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-starlingx-windriver-com-v1-host,mutating=true,failurePolicy=fail,sideEffects=None,groups=starlingx.windriver.com,resources=hosts,verbs=create;update,versions=v1,name=mhost.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &Host{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Host) Default() {
	hostlog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
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
// +kubebuilder:webhook:verbs=create;update,path=/validate-starlingx-windriver-com-v1-host,mutating=false,failurePolicy=fail,sideEffects=None,groups=starlingx.windriver.com,resources=hosts,versions=v1,name=vhost.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &Host{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Host) ValidateCreate() error {
	hostlog.Info("validate create", "name", r.Name)

	// TODO(user): fill in your validation logic upon object creation.
	return r.validateHost()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Host) ValidateUpdate(old runtime.Object) error {
	hostlog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return r.validateHost()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Host) ValidateDelete() error {
	hostlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
