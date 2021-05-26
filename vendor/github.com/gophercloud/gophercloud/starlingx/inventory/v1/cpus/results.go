/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package cpus

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
)

// Extract interprets any commonResult as an Image.
func (r commonResult) Extract() (*CPU, error) {
	var s CPU
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

// CreateResult represents the result of an update operation.
type CreateResult struct {
	commonResult
}

// UpdateResult represents the result of a create operation.
type UpdateResult struct {
	commonResult
}

// CPU defines the data associated to a single cPU instance.
type CPU struct {
	// ID is the generated unique UUID for the CPU instance
	ID string `json:"uuid"`

	// Processor is the processor package number.  A host usually has 1 or 2
	// processors but that number could be higher.  Each is usually associated
	// to a seperate NUMA socket.
	Processor int `json:"numa_node"`

	// LogicalCore is the logical core number of the core.  If hypher-threading
	// is disabled then this may map directly to the physical core number.
	LogicalCore int `json:"cpu"`

	// PhysicalCore is the physical core number.  If hyper-threading is enabled
	// then multiple logical cores will share the same physical core.
	PhysicalCore int `json:"core"`

	// Thread is the hyper-threading thread number.
	Thread int `json:"thread"`

	// Function is the function assigned to the individual core.
	Function string `json:"allocated_function"`

	// CreatedAt defines the timestamp at which the resource was created.
	CreatedAt string `json:"created_at"`

	// UpdatedAt defines the timestamp at which the resource was last updated.
	UpdatedAt *string `json:"updated_at,omitempty"`
}

// CPUPage is the page returned by a pager when traversing over a
// collection of cPUs.
type CPUPage struct {
	pagination.SinglePageBase
}

// IsEmpty checks whether a CPUPage struct is empty.
func (r CPUPage) IsEmpty() (bool, error) {
	is, err := ExtractCPUs(r)
	return len(is) == 0, err
}

// ExtractCPUs accepts a Page struct, specifically a CPUPage struct,
// and extracts the elements into a slice of CPU structs. In other words,
// a generic collection is mapped into a relevant slice.
func ExtractCPUs(r pagination.Page) ([]CPU, error) {
	var s struct {
		CPU []CPU `json:"icpus"`
	}

	err := (r.(CPUPage)).ExtractInto(&s)

	return s.CPU, err
}
