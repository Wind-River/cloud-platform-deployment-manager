/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package memory

import "github.com/gophercloud/gophercloud"

func resourceURL(c *gophercloud.ServiceClient, hostid string) string {
	return c.ServiceURL("ihosts", hostid, "imemorys")
}

func listURL(c *gophercloud.ServiceClient, hostid string) string {
	return resourceURL(c, hostid)
}

func updateURL(c *gophercloud.ServiceClient, id string) string {
	return c.ServiceURL("imemorys", id)
}
