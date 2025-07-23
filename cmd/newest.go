/*
Copyright © 2022 zbc <zbc@sangfor.com.cn>
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zgsm-ai/smc/internal/utils"
)

/**
 *	Set the latest version
 */
func setNewest(packages *utils.VersionList, ver string) error {
	verId, err := utils.ParseVersion(ver)
	if err != nil {
		return err
	}
	for _, v := range packages.Versions {
		if utils.CompareVersion(v.VersionId, verId) != 0 {
			continue
		}
		packages.Newest = v
		return nil
	}
	return fmt.Errorf("version '%s' not exist", ver)
}

/**
 *	Load package list file
 */
func loadPackagesFile(fname string) (*utils.VersionList, error) {
	bytes, err := os.ReadFile(fname)
	if err != nil {
		return nil, err
	}
	packages := &utils.VersionList{}
	if err = json.Unmarshal(bytes, packages); err != nil {
		return nil, err
	}
	return packages, nil
}

/**
 *	Save package list file
 */
func savePackagesFile(fname string, packages *utils.VersionList) error {
	data, err := json.MarshalIndent(packages, "", "  ")
	if err != nil {
		return err
	}
	if err = os.WriteFile(fname, data, 0664); err != nil {
		return err
	}
	return nil
}

/**
 *	Build packages-{os}-{arch}.json for each platform
 */
func editPackages() error {
	packages, err := loadPackagesFile(optPackagesFile)
	if err != nil {
		return err
	}
	if err = setNewest(packages, optNewestVer); err != nil {
		return err
	}
	if err = savePackagesFile(optPackagesFile, packages); err != nil {
		return err
	}
	return nil
}

// newestCmd represents the 'smc newest' command
var newestCmd = &cobra.Command{
	Use:   "newest {packages | -p packages} -v version",
	Short: "Modify the latest version setting in package list file",
	Long:  `smc newest modifies the latest version setting in package list file, 'costrict upgrade' command will update to this version by default`,
	Args:  cobra.MaximumNArgs(1),

	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 1 {
			optPackagesFile = args[0]
		}
		if err := editPackages(); err != nil {
			fmt.Println(err)
		}
	},
}

var optPackagesFile string
var optNewestVer string

func init() {
	rootCmd.AddCommand(newestCmd)

	newestCmd.Example = `  # Modify latest version in build/packages-windows-amd64.json to 1.2.1213
	 # Setting latest version allows publishing test packages without affecting users, unless user specifies version during update
	 smc newest build/packages-windows-amd64.json -v 1.2.1213`
	newestCmd.Flags().SortFlags = false
	newestCmd.Flags().StringVarP(&optPackagesFile, "packages", "p", "", "package list file")
	newestCmd.Flags().StringVarP(&optNewestVer, "version", "n", "", "Default latest version for user updates")
}
