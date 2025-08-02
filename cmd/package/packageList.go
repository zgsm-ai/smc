package pkg

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/iancoleman/orderedmap"
	"github.com/spf13/cobra"
	"github.com/zgsm-ai/smc/internal/utils"
)

/**
 *	Fields displayed in list format
 */
type Package_Columns struct {
	PackageName  string `json:"packageName"`
	Size         string `json:"size"`
	Checksum     string `json:"checksum"`
	ChecksumAlgo string `json:"checksumAlgo"`
	Version      string `json:"version"`
	Build        string `json:"build"`
	Os           string `json:"os"`
	Arch         string `json:"arch"`
	Description  string `json:"description"`
}

/**
 *	List package information
 */

func packageList(packageName string, verbose bool) error {
	// 获取 .costrict/package 目录路径
	_, _, packageDir := utils.GetCostrictDir()

	// 检查目录是否存在
	if _, err := os.Stat(packageDir); os.IsNotExist(err) {
		return fmt.Errorf("package directory '%s' does not exist", packageDir)
	}

	// 遍历目录中的 *.json 文件
	var packageInfos []utils.PackageInfo
	err := filepath.WalkDir(packageDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// 只处理 *.json 文件
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".json") {
			data, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("failed to read file '%s': %v", path, err)
			}

			var pkgInfo utils.PackageInfo
			if err := json.Unmarshal(data, &pkgInfo); err != nil {
				return fmt.Errorf("failed to unmarshal package info from '%s': %v", path, err)
			}

			// 如果指定了包名，则只返回匹配的包
			if packageName != "" && pkgInfo.PackageName != packageName {
				return nil
			}

			packageInfos = append(packageInfos, pkgInfo)
		}
		return nil
	})

	if err != nil {
		return err
	}

	if len(packageInfos) == 0 {
		return fmt.Errorf("no packages found in '%s'", packageDir)
	}

	// 如果指定了包名且只有一个包，显示详细信息
	if packageName != "" && len(packageInfos) == 1 {
		if verbose {
			utils.PrintYaml(packageInfos[0])
		} else {
			utils.PrintFormatByOrderMap(packageInfos[0])
		}
		return nil
	}

	// 格式化输出包列表
	var dataList []*orderedmap.OrderedMap
	for _, pkg := range packageInfos {
		row := Package_Columns{}
		row.PackageName = pkg.PackageName
		row.Os = pkg.Os
		row.Arch = pkg.Arch
		row.Size = fmt.Sprintf("%d", pkg.Size)
		row.Checksum = pkg.Checksum
		row.ChecksumAlgo = pkg.ChecksumAlgo
		row.Version = fmt.Sprintf("%d.%d.%d", pkg.VersionId.Major, pkg.VersionId.Minor, pkg.VersionId.Micro)
		row.Build = pkg.Build
		row.Description = pkg.Description

		recordMap, _ := utils.StructToOrderedMap(row)
		dataList = append(dataList, recordMap)
	}

	utils.PrintFormat(dataList)
	return nil
}

// packageListCmd represents the 'smc package list' command
var packageListCmd = &cobra.Command{
	Use:   "list {package | -p package}",
	Short: "List available packages",
	Long:  `Lists all available packages in the .costrict/package directory`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 1 {
			optListPackageName = args[0]
		}
		return packageList(optListPackageName, optListVerbose)
	},
}

const packageListExample = `  # List all packages
  smc package list
  # List specific package
  smc package list aip
  # Show package details
  smc package list -p aip -v
  # List remote versions
  smc package remote aip`

var optListPackageName string
var optListVerbose bool

func init() {
	packageCmd.AddCommand(packageListCmd)
	packageListCmd.Flags().SortFlags = false
	packageListCmd.Example = packageListExample
	packageListCmd.Flags().StringVarP(&optListPackageName, "package", "p", "", "Package name")
	packageListCmd.Flags().BoolVarP(&optListVerbose, "verbose", "v", false, "Show details")
}
