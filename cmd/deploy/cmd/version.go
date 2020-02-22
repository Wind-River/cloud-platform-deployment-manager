/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2020 Wind River Systems, Inc. */

package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

var GitLastTag string
var GitHead string
var GitBranch string

func VersionToString() string {
	return (fmt.Sprintf("%s (%s: %s)", GitLastTag, GitBranch, GitHead))
}

func VersionCmdRun(cmd *cobra.Command, args []string) {
	fmt.Printf("Version: %s\n", VersionToString())
	os.Exit(0)
}

// versionCmd represents the build command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display version information",
	Run:   VersionCmdRun,
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
