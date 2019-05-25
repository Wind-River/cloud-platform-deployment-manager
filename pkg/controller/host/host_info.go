/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package host

import (
	"context"
	"encoding/json"
	perrors "github.com/pkg/errors"
	starlingxv1beta1 "github.com/wind-river/titanium-deployment-manager/pkg/apis/starlingx/v1beta1"
	v1info "github.com/wind-river/titanium-deployment-manager/pkg/platform"
)

// BuildHostDefaults takes the current set of host attributes and builds a
// fake host profile that can be used as a reference for the current settings
// applied to the host.  The default settings are saved on the host status.
func (r *ReconcileHost) BuildHostDefaults(instance *starlingxv1beta1.Host, host *v1info.HostInfo) (*starlingxv1beta1.HostProfileSpec, error) {
	defaults, err := starlingxv1beta1.NewHostProfileSpec(host)
	if defaults == nil || err != nil {
		return nil, err
	}

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

	return defaults, nil
}

// GetHostDefaults retrieves the default attributes for a host.  The set of
// default attributes are collected from the host before any user configurations
// are applied.
func (r *ReconcileHost) GetHostDefaults(instance *starlingxv1beta1.Host) (*starlingxv1beta1.HostProfileSpec, error) {
	if instance.Status.Defaults == nil {
		return nil, nil
	}

	defaults := starlingxv1beta1.HostProfileSpec{}
	err := json.Unmarshal([]byte(*instance.Status.Defaults), &defaults)
	if err != nil {
		err = perrors.Wrap(err, "failed to unmarshal host defaults")
		return nil, err
	}

	return &defaults, nil
}
