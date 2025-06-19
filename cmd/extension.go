/*
Copyright © 2022 zbc <zbc@sangfor.com.cn>
*/
package cmd

import (
	"github.com/spf13/cobra"
)

// extensionCmd represents the 'smc extension' command
var extensionCmd = &cobra.Command{
	Use:   "extension",
	Short: "Manage extension definitions in REDIS",
	Long:  `'smc extension' can be used to manage extension definitions in REDIS`,
}

const extensionExample = `  # List extension definitions in REDIS
  smc extension list`

var optExtensionId string

func init() {
	rootCmd.AddCommand(extensionCmd)

	extensionCmd.Example = extensionExample
}
