package component

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zgsm-ai/smc/cmd/common"
	"github.com/zgsm-ai/smc/internal/utils"
)

func removePackage() error {
	var err error
	if err = common.InitCommonEnv(); err != nil {
		return err
	}

	if optRemovePackageName == "" {
		fmt.Println("Error: package name is required (either as positional argument or via -p/--package option)")
		return fmt.Errorf("miss parameter")
	}

	if err = utils.RemovePackage("", optRemovePackageName); err != nil {
		fmt.Printf("The '%s' remove failed: %v", optRemovePackageName, err)
		return err
	}

	fmt.Printf("The '%s' is removed successfully\n", optRemovePackageName)
	return nil
}

var removeCmd = &cobra.Command{
	Use:   "remove {package-name | -p package-name}",
	Short: "Remove package",
	Long:  `Remove package`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 1 {
			optRemovePackageName = args[0]
		}
		return removePackage()
	},
}

const removeExample = `  # remove package
  smc package remove -p codebase-syncer
  smc package remove codebase-syncer`

var optRemovePackageName string

func init() {
	componentCmd.AddCommand(removeCmd)
	removeCmd.Flags().SortFlags = false
	removeCmd.Example = removeExample

	removeCmd.Flags().StringVarP(&optRemovePackageName, "package", "p", "", "package name")
}
