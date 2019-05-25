/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */
package osds

// Gibibytes returns the partition size in GiB units.
func (in *JournalInfo) Gibibytes() int {
	if in.Size != nil {
		return *in.Size
	}
	return 0
}
