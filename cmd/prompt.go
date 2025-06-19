/*
Copyright © 2022 zbc <zbc@sangfor.com.cn>
*/
package cmd

import (
	"github.com/spf13/cobra"
)

// promptCmd represents the 'smc prompt' command
var promptCmd = &cobra.Command{
	Use:   "prompt",
	Short: "Manages prompt templates in REDIS",
	Long:  `'smc prompt' manages prompt templates in REDIS`,
}

const promptExample = `  # View prompt template definitions in REDIS
  smc prompt list`

var optPromptId string

func init() {
	rootCmd.AddCommand(promptCmd)

	promptCmd.Example = promptExample
}
