/*
Copyright © 2022 zbc <zbc@sangfor.com.cn>
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zgsm-ai/smc/internal/utils"
)

/**
 *	Package list file for each OS and architecture: packages-{os}-{arch}.json
 */
var packagesList map[string]*utils.VersionList = make(map[string]*utils.VersionList)

/**
 *	Add a package with filename fname (with .json extension) to the package list
 */
func addPackage(fname string) {
	//package-windows-amd64-1.1.1125.json
	_, file := filepath.Split(fname)
	if filepath.Ext(file) != ".json" {
		return
	}
	names := strings.Split(file, "-")
	if len(names) != 4 {
		fmt.Printf("ignore %s\n", fname)
		return
	}
	pkgData := &utils.PackageInfo{}
	bytes, err := os.ReadFile(fname)
	if err != nil {
		fmt.Printf("read error, ignore %s\n", fname)
		return
	}
	if err = json.Unmarshal(bytes, &pkgData); err != nil {
		fmt.Printf("unmarshal error, ignore %s\n", fname)
		return
	}
	fmt.Printf("found package definition: %s\n", fname)
	ver := &utils.VersionAddr{}
	ver.VersionId = pkgData.VersionId
	///{package}/package-windows-amd64-1.1.1130.json
	///{package}/aip-windows-amd64-1.1.1130.exe
	exeExt := ""
	if pkgData.Os == "windows" {
		exeExt = ".exe"
	}
	verStr := utils.PrintVersion(ver.VersionId)
	ver.AppUrl = fmt.Sprintf("/%s/%s-%s-%s-%s%s", pkgData.PackageName, pkgData.PackageName,
		pkgData.Os, pkgData.Arch, verStr, exeExt)
	ver.PackageUrl = ver.AppUrl
	ver.InfoUrl = fmt.Sprintf("/%s/package-%s-%s-%s.json",
		pkgData.PackageName, pkgData.Os, pkgData.Arch, verStr)

	keyStr := fmt.Sprintf("%s-%s", pkgData.Os, pkgData.Arch)
	packages, ok := packagesList[keyStr]
	if !ok {
		packages = &utils.VersionList{}
		packages.PackageName = pkgData.PackageName
		packages.Os = pkgData.Os
		packages.Arch = pkgData.Arch
		packagesList[keyStr] = packages
	}
	packages.Versions = append(packages.Versions, *ver)
}

/**
 *	Get version info of the newest package
 */
func getNewest() {
	for _, pkgs := range packagesList {
		newest := utils.VersionAddr{}
		for _, v := range pkgs.Versions {
			if utils.CompareVersion(v.VersionId, newest.VersionId) > 0 {
				newest = v
			}
		}
		pkgs.Newest = newest
	}
}

/**
 *	Save packages-{os}-{arch}.json files to the directory specified by --build
 */
func savePackages() {
	for _, v := range packagesList {
		data, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			fmt.Println(err)
		}
		fname := fmt.Sprintf("packages-%s-%s.json", v.Os, v.Arch)
		fpath := filepath.Join(optBuildDir, fname)

		fmt.Printf("create %s\n", fpath)
		if err = os.WriteFile(fpath, data, 0666); err != nil {
			fmt.Println(err)
		}
	}
}

/**
 *	Build packages-{os}-{arch}.json for each platform
 */
func makePackages() error {
	// Traverse all package-{os}-{arch}-{ver}.json files to build version list
	filepath.Walk(optBuildDir, func(path string, f os.FileInfo, err error) error {
		if f == nil {
			return err
		}
		if f.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".json" {
			addPackage(path)
		}
		return nil
	})
	// Get the newest version
	getNewest()
	savePackages()
	return nil
}

// packagesCmd represents the 'smc packages' command
var packagesCmd = &cobra.Command{
	Use:   "packages [-b build-dir]",
	Short: "Generate package list file (packages.json)",
	Long:  `smc packages scans all packages in specified directory and generates packages.json`,

	Run: func(cmd *cobra.Command, args []string) {
		if err := makePackages(); err != nil {
			fmt.Println(err)
		}
	},
}

var optBuildDir string

func init() {
	rootCmd.AddCommand(packagesCmd)

	packagesCmd.Example = `  # Scan ./build directory and generate packages-<os>-<arch>.json list files based on signed packages
	 smc packages -b ./build`
	packagesCmd.Flags().SortFlags = false
	packagesCmd.Flags().StringVarP(&optBuildDir, "build", "b", "./build", "Build directory: location of package files")
}
