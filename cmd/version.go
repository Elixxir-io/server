////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles command-line version functionality

package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"gitlab.com/elixxir/primitives/utils"
)

func printVersion() {
	fmt.Printf("Elixxir Server v%s -- %s\n\n", SEMVER, GITVERSION)
	//fmt.Printf("Dependencies:\n\n%s\n", DEPENDENCIES)
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(generateCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version and dependency information for the Elixxir binary",
	Long:  `Print the version and dependency information for the Elixxir binary`,
	Run: func(cmd *cobra.Command, args []string) {
		printVersion()
	},
}

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generates version and dependency information for the Elixxir binary",
	Long:  `Generates version and dependency information for the Elixxir binary`,
	Run: func(cmd *cobra.Command, args []string) {
		utils.GenerateVersionFile()
	},
}
