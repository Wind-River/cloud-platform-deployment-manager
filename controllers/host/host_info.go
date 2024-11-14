/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package host

import (
	"context"
	"encoding/json"

	perrors "github.com/pkg/errors"
	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
	v1info "github.com/wind-river/cloud-platform-deployment-manager/platform"
)

// BuildHostDefaults takes the current set of host attributes and builds a
// fake host profile that can be used as a reference for the current settings
// applied to the host.  The default settings are saved on the host status.
func (r *HostReconciler) BuildHostDefaults(instance *starlingxv1.Host, host v1info.HostInfo) (*starlingxv1.HostProfileSpec, error) {
	logHost.Info("building host defaults", "host", host.ID)

	defaults, err := starlingxv1.NewHostProfileSpec(host)
	if defaults == nil || err != nil {
		err = perrors.Wrap(err, "failed to create host profile spec")
		return nil, err
	}

	logHost.V(2).Info("host profile spec created", "defaults", defaults)

	buffer, err := json.Marshal(defaults)
	if err != nil {
		err = perrors.Wrap(err, "failed to marshal host defaults")
		return nil, err
	}

	data := string(buffer)
	instance.Status.Defaults = &data

	err = r.Status().Update(context.Background(), instance)
	if err != nil {
		err = perrors.Wrap(err, "failed to update host defaults")
		return nil, err
	}

	logHost.Info("host defaults successfully built and updated", "host", host.ID)

	return defaults, nil
}

// GetHostDefaults retrieves the default attributes for a host.  The set of
// default attributes are collected from the host before any user configurations
// are applied.
func (r *HostReconciler) GetHostDefaults(instance *starlingxv1.Host) (*starlingxv1.HostProfileSpec, error) {
	if instance.Status.Defaults == nil {
		return nil, nil
	}

	defaults := starlingxv1.HostProfileSpec{}
	err := json.Unmarshal([]byte(*instance.Status.Defaults), &defaults)
	if err != nil {
		err = perrors.Wrap(err, "failed to unmarshal host defaults")
		return nil, err
	}

	return &defaults, nil
}
