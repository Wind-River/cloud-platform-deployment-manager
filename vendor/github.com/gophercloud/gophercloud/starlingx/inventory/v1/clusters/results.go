/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Clusters, Inc. */

package clusters

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
)

// Defines the known storage deployment models
const (
	DeploymentModelAIO        = "aio-sx"
	DeploymentModelController = "controller-nodes"
	DeploymentModelStorage    = "storage-nodes"
	DeploymentModelUndefined  = "undefined"
)

// Extract interprets any commonResult as an Image.
func (r commonResult) Extract() (*Cluster, error) {
	var s Cluster
	err := r.ExtractInto(&s)
	return &s, err
}

type commonResult struct {
	gophercloud.Result
}

// GetResult represents the result of a get operation.
type GetResult struct {
	commonResult
}

// UpdateResult represents the result of an update operation.
type UpdateResult struct {
	commonResult
}

// Cluster defines the data associated to a single cluster instance.
type Cluster struct {
	// ID is the system generated unique UUID for the cluster
	ID string `json:"uuid"`

	// SystemName is the human readable name of the cluster.
	Name string `json:"name"`

	// Type is the storage cluster type. e.g., ceph
	Type string `json:"type"`

	// ClusterID is the file system UUID assigned to this cluster.
	ClusterID *string `json:"cluster_uuid,omitempty"`

	// DeploymentModel is the storage deployment mode that is currently in use.
	// This dictates on what nodes and at what stage OSDs can be configured.
	DeploymentModel string `json:"deployment_model"`
}

// ClusterPage is the page returned by a pager when traversing over a
// collection of systems.
type ClusterPage struct {
	pagination.SinglePageBase
}

// IsEmpty checks whether a ClusterPage struct is empty.
func (r ClusterPage) IsEmpty() (bool, error) {
	is, err := ExtractClusters(r)
	return len(is) == 0, err
}

// ExtractClusters accepts a Page struct, specifically a ClusterPage struct,
// and extracts the elements into a slice of Cluster structs. In other words,
// a generic collection is mapped into a relevant slice.
func ExtractClusters(r pagination.Page) ([]Cluster, error) {
	var s struct {
		Cluster []Cluster `json:"clusters"`
	}

	err := (r.(ClusterPage)).ExtractInto(&s)

	return s.Cluster, err
}
