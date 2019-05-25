/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package labels

import "github.com/gophercloud/gophercloud"

func resourceURL(c *gophercloud.ServiceClient, id string) string {
	return c.ServiceURL("labels", id)
}

func rootURL(c *gophercloud.ServiceClient, hostid string) string {
	return c.ServiceURL("labels", hostid)
}

func listURL(c *gophercloud.ServiceClient, hostid string) string {
	return c.ServiceURL("ihosts", hostid, "labels")
}

func createURL(c *gophercloud.ServiceClient, hostid string) string {
	return rootURL(c, hostid)
}

func deleteURL(c *gophercloud.ServiceClient, id string) string {
	return resourceURL(c, id)
}
