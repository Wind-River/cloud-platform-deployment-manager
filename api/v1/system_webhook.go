/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019-2024 Wind River Systems, Inc. */

package v1

import (
	"context"
	"errors"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	apitypes "k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var systemlog = logf.Log.WithName("system-resource")
var cl client.Client

// Webhook response reasons
const SystemAllowedReason string = "allowed to be admitted"
const SecretRetrieveTryCount = 30
const SecretRetrieveTryInterval = 2 * time.Second

const (
	// Backend types
	file = "file"
	lvm  = "lvm"
	ceph = "ceph"
	rook = "ceph-rook"
)

const (
	// Backend services
	glance         = "glance"
	cinder         = "cinder"
	nova           = "nova"
	swift          = "swift"
	rbdProvisioner = "rbd-provisioner"
	block          = "block"
	ecblock        = "ecblock"
	filesystem     = "filesystem"
	object         = "object"
)

var validBackendServices = map[string]map[string]bool{
	file: {
		glance: true,
	},

	lvm: {
		cinder: true,
	},

	ceph: {
		glance:         true,
		cinder:         true,
		nova:           true,
		swift:          true,
		rbdProvisioner: true,
	},

	rook: {
		block:      true,
		ecblock:    true,
		filesystem: true,
		object:     true,
	},
}

func (r *System) SetupWebhookWithManager(mgr ctrl.Manager) error {
	cl = mgr.GetClient()

	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-starlingx-windriver-com-v1-system,mutating=true,failurePolicy=fail,sideEffects=None,groups=starlingx.windriver.com,resources=systems,verbs=create;update,versions=v1,name=msystem.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &System{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *System) Default() {
	systemlog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}

func validateBackendServices(backendType string, services []string) error {
	for _, s := range services {
		if !validBackendServices[backendType][s] {
			msg := fmt.Sprintf("%s service not allowed with %s backend.", s, backendType)
			return errors.New(msg)
		}
	}

	return nil
}

func validateBackendAttributes(backend StorageBackend) error {
	if backend.Deployment != "" {
		if backend.Type != rook {
			msg := fmt.Sprintf("deployment is only permitted with %s backend", rook)
			return errors.New(msg)
		}
	}

	if backend.PartitionSize != nil {
		if backend.Type != ceph {
			msg := fmt.Sprintf("partitionSize is only permitted with %s backend", ceph)
			return errors.New(msg)
		}
	}

	if backend.ReplicationFactor != nil {
		if backend.Type != ceph && backend.Type != rook {
			msg := fmt.Sprintf("replicationFactor is only permitted with %s and %s backends", ceph, rook)
			return errors.New(msg)
		}
	}

	return nil
}

func validateStorageBackends(obj *System) error {
	var present = make(map[string]bool)

	for _, b := range *obj.Spec.Storage.Backends {
		if present[b.Type] {
			return errors.New("backend services may only be specified once.")
		}

		if b.Services != nil {
			err := validateBackendServices(b.Type, b.Services)
			if err != nil {
				return err
			}
		}

		err := validateBackendAttributes(b)
		if err != nil {
			return err
		}

		present[b.Type] = true
	}

	if present[ceph] && present[rook] {
		return errors.New("ceph and ceph-rook backends are not supported at the same time")
	}

	return nil
}

func validateStorage(obj *System) error {
	if obj.Spec.Storage != nil && obj.Spec.Storage.Backends != nil {
		err := validateStorageBackends(obj)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateCertificates(obj *System) error {
	if obj.Spec.Certificates != nil {
		for _, c := range *obj.Spec.Certificates {
			// Ignore certificates installed during bootstrap/initial unlock
			// - Openstack_CA/OpenLDAP/Docker/SSL(HTTPS)
			if c.Type == OpenstackCACertificate || c.Type == OpenLDAPCertificate ||
				c.Type == DockerCertificate || c.Type == PlatformCertificate {
				continue
			}

			secret := &corev1.Secret{}
			secretName := apitypes.NamespacedName{Name: c.Secret, Namespace: obj.ObjectMeta.Namespace}
			found := false

			for count := 0; count < SecretRetrieveTryCount; count++ {
				err := cl.Get(context.TODO(), secretName, secret)
				if err != nil {
					systemlog.Info("unable to retrieve secret, try again...", "secretName", secretName, "count", count)
				} else {
					systemlog.Info("find secret", "secretName", secretName)
					found = true
					break
				}
				time.Sleep(SecretRetrieveTryInterval)
			}
			if !found {
				msg := fmt.Sprintf("unable to retrieve %s secret %s", c.Type, secretName)
				return errors.New(msg)
			}
		}
	}

	return nil
}

func (r *System) validatingSystem() error {
	err := validateStorage(r)
	if err != nil {
		return err
	}

	err = validateCertificates(r)
	if err != nil {
		return err
	}

	systemlog.Info(SystemAllowedReason)
	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:verbs=create;update,path=/validate-starlingx-windriver-com-v1-system,mutating=false,failurePolicy=fail,sideEffects=None,groups=starlingx.windriver.com,resources=systems,versions=v1,name=vsystem.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &System{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *System) ValidateCreate() error {
	systemlog.Info("validate create", "name", r.Name)

	return r.validatingSystem()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *System) ValidateUpdate(old runtime.Object) error {
	systemlog.Info("validate update", "name", r.Name)

	return r.validatingSystem()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *System) ValidateDelete() error {
	systemlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
