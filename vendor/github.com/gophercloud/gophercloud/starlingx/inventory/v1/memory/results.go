/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package memory

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
)

// Extract interprets any commonResult as an Image.
func (r commonResult) Extract() (*Memory, error) {
	var s Memory
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

// Memory defines the data associated to a single Memory instance.
type Memory struct {
	// ID is the generated unique UUID for the Memory instance
	ID string `json:"uuid"`

	// Processor is the socket or NUMA node that this record relates to.
	Processor int `json:"numa_node"`

	// Total is the total amount of memory installed.
	Total int `json:"memtotal_mib"`

	// Available is the total available memory remaining.
	Available int `json:"memavail_mib"`

	// Platform is the total amount of memory reserved for platform use
	Platform int `json:"platform_reserved_mib"`

	// PlatformMinimum is the minimum amount of memory required for platform use
	PlatformMinimum int `json:"minimum_platform_reserved_mib"`

	// VM1GHugepagesCount represents the current number of 1G pages allocated
	// for VM instance usage.
	VM1GHugepagesCount int `json:"vm_hugepages_nr_1G"`

	// VM1GHugepagesEnabled indicates whether VM instances are able to use
	// 1G hugepages.
	VM1GHugepagesEnabled string `json:"vm_hugepages_use_1G"`

	// VM1GHugepagesPending indicates whether there is a change pending to the
	// current number of 1G hugepages allocated for VM usage.
	VM1GHugepagesPending *int `json:"vm_hugepages_nr_1G_pending,omitempty"`

	// VM1GHugepagesPossible represents the number of 1G hugepages available
	// for VM usage.
	VM1GHugepagesPossible int `json:"vm_hugepages_possible_1G"`

	// VM1GHugepagesAvailable represents the current number of pages still
	// available for usage by VM instances.
	VM1GHugepagesAvailable int `json:"vm_hugepages_avail_1G"`

	// VM2MHugepagesCount represents the current number of 2M pages allocated
	// for VM instance usage.
	VM2MHugepagesCount int `json:"vm_hugepages_nr_2M"`

	// VM2MHugepagesEnabled indicates whether VM instances are able to use
	// 2M hugepages.
	VM2MHugepagesEnabled string `json:"vm_hugepages_use_2M"`

	// VM2MHugepagesPending indicates whether there is a change pending to the
	// current number of 2M hugepages allocated for VM usage.
	VM2MHugepagesPending *int `json:"vm_hugepages_nr_2M_pending,omitempty"`

	// VM2MHugepagesPossible represents the number of 2M hugepages available
	// for VM usage.
	VM2MHugepagesPossible int `json:"vm_hugepages_possible_2M"`

	// VM2MHugepagesAvailable represents the current number of pages still
	// available for usage by VM instances.
	VM2MHugepagesAvailable int `json:"vm_hugepages_avail_2M"`

	// VSwitchHugepagesSize represents the current hugepage size that is used
	// for virtual switch memory.
	VSwitchHugepagesSize int `json:"vswitch_hugepages_size_mib"`

	// VSwitchHugepagesRequired indicates whether there is a change pending to
	// the current number of hugepages allocated for virtual switch usage.
	VSwitchHugepagesRequired *int `json:"vswitch_hugepages_reqd"`

	// VSwitchHugepagesAvailable represents the current number of pages still
	// available for virtual switch usage.
	VSwitchHugepagesAvailable int `json:"vswitch_hugepages_avail"`

	// VSwitchHugepagesCount
	VSwitchHugepagesCount int `json:"vswitch_hugepages_nr"`

	// CreatedAt defines the timestamp at which the resource was created.
	CreatedAt string `json:"created_at"`

	// UpdatedAt defines the timestamp at which the resource was last updated.
	UpdatedAt *string `json:"updated_at,omitempty"`
}

// MemoryPage is the page returned by a pager when traversing over a
// collection of Memorys.
type MemoryPage struct {
	pagination.SinglePageBase
}

// IsEmpty checks whether a MemoryPage struct is empty.
func (r MemoryPage) IsEmpty() (bool, error) {
	is, err := ExtractMemorys(r)
	return len(is) == 0, err
}

// ExtractMemorys accepts a Page struct, specifically a MemoryPage struct,
// and extracts the elements into a slice of Memory structs. In other words,
// a generic collection is mapped into a relevant slice.
func ExtractMemorys(r pagination.Page) ([]Memory, error) {
	var s struct {
		Memory []Memory `json:"imemorys"`
	}

	err := (r.(MemoryPage)).ExtractInto(&s)

	return s.Memory, err
}
