/*
Copyright © 2023 xmz <xmz@sangfor.com.cn>
*/
package cmd

import (
	"github.com/spf13/cobra"
	"github.com/zgsm-ai/smc/internal/utils"
)

func upgradePackage() error {
	var err error
	if err = InitDebug(optDebug, optLogFile); err != nil {
		return err
	}
	cfg := utils.UpgradeConfig{}
	cfg.PackageName = optPackageName
	curVer := utils.VersionNumber{}
	if optPackageVersion != "" {
		var ver utils.VersionNumber
		ver, err = utils.ParseVersion(optVersion)
		if err != nil {
			return err
		}
		err = utils.UpgradePackage(&cfg, curVer, &ver)
	} else {
		err = utils.UpgradePackage(&cfg, curVer, nil)
	}
	return err
}

// upgradeCmd represents the 'smc task tags' command
var upgradeCmd = &cobra.Command{
	Use:   "upgrade {package-name | -p package-name}",
	Short: "Upgrade package",
	Long:  `'smc upgrade' upgrade package`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 1 {
			optPackageName = args[0]
		}
		return upgradePackage()
	},
}

const upgradeExample = `  # upgrade package
  smc upgrade -p codebase-syncer -v 1.0.0
  smc upgrade -p codebase-syncer`

var optPackageVersion string
var optPackageName string

func init() {
	rootCmd.AddCommand(upgradeCmd)
	upgradeCmd.Flags().SortFlags = false
	upgradeCmd.Example = upgradeExample

	upgradeCmd.Flags().StringVarP(&optPackageName, "package", "p", "", "package name")
	upgradeCmd.Flags().StringVarP(&optPackageVersion, "version", "v", "", "package version")
}
