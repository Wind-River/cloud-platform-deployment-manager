/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package licenses

import "github.com/gophercloud/gophercloud"

func rootURL(c *gophercloud.ServiceClient) string {
	return c.ServiceURL("license")
}

func getURL(c *gophercloud.ServiceClient) string {
	return c.ServiceURL("license", "get_license_file")
}

func createURL(c *gophercloud.ServiceClient) string {
	return c.ServiceURL("license", "install_license")
}
