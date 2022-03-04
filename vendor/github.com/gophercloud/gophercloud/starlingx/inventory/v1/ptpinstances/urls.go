/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2022 Wind River Systems, Inc. */

package ptpinstances

import "github.com/gophercloud/gophercloud"

func resourceURL(c *gophercloud.ServiceClient, id string) string {
	return c.ServiceURL("ptp_instances", id)
}

func rootURL(c *gophercloud.ServiceClient) string {
	return c.ServiceURL("ptp_instances")
}

func getURL(c *gophercloud.ServiceClient, id string) string {
	return resourceURL(c, id)
}

func listURL(c *gophercloud.ServiceClient) string {
	return rootURL(c)
}

func createURL(c *gophercloud.ServiceClient) string {
	return c.ServiceURL("ptp_instances")
}

func updateURL(c *gophercloud.ServiceClient, id string) string {
	return resourceURL(c, id)
}

func deleteURL(c *gophercloud.ServiceClient, id string) string {
	return resourceURL(c, id)
}

func applyURL(c *gophercloud.ServiceClient) string {
	return c.ServiceURL("ptp_instances", "apply")
}

func hostListURL(c *gophercloud.ServiceClient, hostid string) string {
	return c.ServiceURL("ihosts", hostid, "ptp_instances")
}

func hostUpdateURL(c *gophercloud.ServiceClient, hostid string) string {
	return c.ServiceURL("ihosts", hostid)
}
