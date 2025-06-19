/*
Copyright © 2022 zbc <zbc@sangfor.com.cn>
*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

/**
 * Root command configuration with base flags and examples
 */
var rootCmd = &cobra.Command{
	Use:   "smc",
	Short: "Shenma client tool",
	Long: `smc is the Shenma client tool for managing backend services
Features include task management, tagging, and extension management`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

const rootExample = `  smc task list
  smc template add
  smc pool add
`

/**
 * Initialize root command with examples and persistent flags
 */
func init() {
	rootCmd.Example = rootExample
	rootCmd.PersistentFlags().StringVarP(&optLogFile, "logfile", "L", "", "Log file path (empty for stderr, +xx.log for both stderr and file)")
	rootCmd.PersistentFlags().StringVarP(&optDebug, "debug", "D", "", "Debug level: Off,Err,Dbg")
}
