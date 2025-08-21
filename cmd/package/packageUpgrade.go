package pkg

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zgsm-ai/smc/cmd/common"
	"github.com/zgsm-ai/smc/internal/env"
	"github.com/zgsm-ai/smc/internal/utils"
)

func upgradePackage() error {
	var err error
	var retVer utils.VersionNumber
	if err = common.InitCommonEnv(); err != nil {
		return err
	}
	cfg := utils.UpgradeConfig{}
	cfg.BaseUrl = env.BaseUrl + "/costrict"
	cfg.PackageName = optPackageName
	if optPublicKey != "" {
		// 读取公钥文件内容
		keyData, err := os.ReadFile(optPublicKey)
		if err != nil {
			return fmt.Errorf("failed to read public key file: %v", err)
		}
		cfg.PublicKey = string(keyData)
	}
	cfg.Correct()
	curVer, _ := utils.GetLocalVersion(cfg)
	if optPackageVersion != "" {
		var ver utils.VersionNumber
		ver, err = utils.ParseVersion(optVersion)
		if err != nil {
			return err
		}
		retVer, err = utils.UpgradePackage(cfg, curVer, &ver)
	} else {
		retVer, err = utils.UpgradePackage(cfg, curVer, nil)
	}
	if err != nil {
		fmt.Printf("The '%s' upgrade failed: %v", cfg.PackageName, err)
		return err
	}
	if utils.CompareVersion(retVer, curVer) == 0 {
		fmt.Printf("The '%s' version is up to date\n", cfg.PackageName)
	} else {
		fmt.Printf("The '%s' is upgraded to version %s\n", cfg.PackageName, utils.PrintVersion(retVer))
	}
	return err
}

var upgradeCmd = &cobra.Command{
	Use:   "upgrade {package-name | -p package-name}",
	Short: "Upgrade package",
	Long:  `Upgrade package`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 1 {
			optPackageName = args[0]
		}
		if optPackageName == "" {
			fmt.Println("Error: package name is required (either as positional argument or via -p/--package option)")
			return fmt.Errorf("miss parameter")
		}
		return upgradePackage()
	},
}

const upgradeExample = `  # upgrade package
  smc package upgrade -p codebase-syncer -v 1.0.0
  smc package upgrade -p codebase-syncer
  smc package upgrade -p codebase-syncer --public /path/to/public.key`

var optPackageVersion string
var optPackageName string
var optPublicKey string

func init() {
	packageCmd.AddCommand(upgradeCmd)
	upgradeCmd.Flags().SortFlags = false
	upgradeCmd.Example = upgradeExample

	upgradeCmd.Flags().StringVarP(&optPackageName, "package", "p", "", "package name")
	upgradeCmd.Flags().StringVarP(&optPackageVersion, "version", "v", "", "package version")
	upgradeCmd.Flags().StringVar(&optPublicKey, "public", "", "public key file for package verification")
}
