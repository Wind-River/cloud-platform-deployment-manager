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

func VersionCmdRun(cmd *cobra.Command, args []string) {
	fmt.Printf("Version: %s (%s: %s)\n", GitLastTag, GitBranch, GitHead)
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
