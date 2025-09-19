package component

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zgsm-ai/smc/cmd/common"
	"github.com/zgsm-ai/smc/internal/utils"
)

func activatePackage() error {
	var err error
	if err = common.InitCommonEnv(); err != nil {
		return err
	}

	if optActivatePackageName == "" {
		fmt.Println("Error: package name is required (either as positional argument or via -p/--package option)")
		return fmt.Errorf("miss parameter")
	}
	if optActivatePackageVersion == "" {
		fmt.Println("Error: package version is required")
		return fmt.Errorf("miss parameter")
	}
	ver, err := utils.ParseVersion(optActivatePackageVersion)
	if err != nil {
		fmt.Printf("The version '%s' is invalid", optActivatePackageVersion)
		return err
	}
	var cfg utils.UpgradeConfig
	cfg.PackageName = optActivatePackageName
	cfg.Correct()

	if err = utils.ActivatePackage(cfg, ver); err != nil {
		fmt.Printf("The '%s-%s' activate failed: %v", optActivatePackageName, optActivatePackageVersion, err)
		return err
	}

	fmt.Printf("The '%s-%s' is activated successfully\n", optActivatePackageName, optActivatePackageVersion)
	return nil
}

var activateCmd = &cobra.Command{
	Use:   "activate {package-name | -p package-name}",
	Short: "Activate package",
	Long:  `Activate package`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 1 {
			optActivatePackageName = args[0]
		}
		return activatePackage()
	},
}

const activateExample = `  # activate package
  smc package activate -p codebase-syncer -v 1.2.0
  smc package activate codebase-syncer -v 1.2.0`

var optActivatePackageName string
var optActivatePackageVersion string

func init() {
	componentCmd.AddCommand(activateCmd)
	activateCmd.Flags().SortFlags = false
	activateCmd.Example = activateExample

	activateCmd.Flags().StringVarP(&optActivatePackageName, "package", "p", "", "package name")
	activateCmd.Flags().StringVarP(&optActivatePackageVersion, "version", "v", "", "special package version")
}
