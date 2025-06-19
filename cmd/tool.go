/*
Copyright © 2022 zbc <zbc@sangfor.com.cn>
*/
package cmd

import (
	"github.com/spf13/cobra"
)

// toolCmd represents the 'smc tool' command
var toolCmd = &cobra.Command{
	Use:   "tool",
	Short: "Manages tool definitions in REDIS",
	Long:  `'smc tool' can be used to manage tool definitions in REDIS`,
}

const toolExample = `  # View tool definitions in REDIS
  smc tool list`

var optToolId string

func init() {
	rootCmd.AddCommand(toolCmd)

	toolCmd.Example = toolExample
}
