/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package host

// listDelta is a utility function which calculates the difference between two
// lists.  If elements in 'b' are not present in 'a' then they will appear in the
// 'added' list.  If elements in a are not present in b then they will appear in
// the 'removed' list.
func listDelta(a, b []string) (added []string, removed []string, same []string) {
	added = make([]string, 0)
	removed = make([]string, 0)
	same = make([]string, 0)
	present := make(map[string]bool)

	for _, s := range a {
		found := false
		for _, x := range b {
			if s == x {
				present[x] = true
				found = true
				break
			}
		}

		if !found {
			removed = append(removed, s)
		}
	}

	for _, x := range b {
		if !present[x] {
			added = append(added, x)
		} else {
			same = append(same, x)
		}
	}

	return added, removed, same
}

// listChange is a utility function which determines if a list of names provided
// in a spec is equivalent to the list of names return by the system API.  Since
// the spec accepts nil as a list that wasn't specified we consider the nil
// case as an empty list when comparing against the system API.
func listChanged(a, b []string) bool {
	if len(a) != len(b) {
		return true
	}

	added, removed, _ := listDelta(a, b)

	return len(added) > 0 || len(removed) > 0
}
