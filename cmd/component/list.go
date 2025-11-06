package component

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/iancoleman/orderedmap"
	"github.com/spf13/cobra"
	"github.com/zgsm-ai/smc/internal/env"
	"github.com/zgsm-ai/smc/internal/utils"
)

/**
 *	Fields displayed in list format
 */
type Package_Columns struct {
	A           string `json:"A"`
	PackageName string `json:"packageName"`
	Size        string `json:"size"`
	Checksum    string `json:"checksum"`
	Algo        string `json:"algo"`
	Version     string `json:"version"`
	Os          string `json:"os"`
	Arch        string `json:"arch"`
	Description string `json:"description"`
}

type PackageInfo struct {
	Ver       *utils.PackageVersion
	Activated bool
}

/**
 *	判断包是否已经在包列表中
 *	基于包名、操作系统、架构和版本判断唯一性
 */
func findPackage(pkgInfo utils.PackageVersion, pkgList []PackageInfo) *PackageInfo {
	for i, p := range pkgList {
		pkg := p.Ver
		if pkg.PackageName == pkgInfo.PackageName &&
			pkg.Os == pkgInfo.Os &&
			pkg.Arch == pkgInfo.Arch &&
			pkg.VersionId.Major == pkgInfo.VersionId.Major &&
			pkg.VersionId.Minor == pkgInfo.VersionId.Minor &&
			pkg.VersionId.Micro == pkgInfo.VersionId.Micro {
			return &pkgList[i]
		}
	}
	return nil
}

/**
 *	扫描目录并收集包信息
 */
func scanPackageDirectory(packageDir string, packageName string) ([]PackageInfo, error) {
	// 检查目录是否存在
	if _, err := os.Stat(packageDir); os.IsNotExist(err) {
		return nil, err
	}

	// 遍历目录中的 *.json 文件
	var packageInfos []PackageInfo
	err := filepath.WalkDir(packageDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != packageDir {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".json") {
			return nil
		}
		// 只处理 *.json 文件
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file '%s': %v", path, err)
		}
		var pkgInfo utils.PackageVersion
		if err := json.Unmarshal(data, &pkgInfo); err != nil {
			return fmt.Errorf("failed to unmarshal package info from '%s': %v", path, err)
		}
		if pkgInfo.PackageName == "" {
			return nil
		}
		// 如果指定了包名，则只返回匹配的包
		if packageName != "" && pkgInfo.PackageName != packageName {
			return nil
		}

		// 检查包是否已经在列表中，确保唯一性
		pkg := findPackage(pkgInfo, packageInfos)
		if pkg != nil {
			pkg.Activated = true
		} else {
			packageInfos = append(packageInfos, PackageInfo{
				Ver:       &pkgInfo,
				Activated: false,
			})
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return packageInfos, nil
}

/**
 *	List package information
 */
func packageList(packageName string, verbose bool) error {
	// 获取 .costrict/package 目录路径
	u := utils.NewUpgrader(packageName, utils.UpgradeConfig{
		BaseUrl: env.BaseUrl + "/costrict",
	})
	packageDir := filepath.Join(u.BaseDir, "package")

	// 扫描目录并收集包信息
	packageInfos, err := scanPackageDirectory(packageDir, packageName)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// 如果指定了包名且只有一个包，显示详细信息
	if len(packageInfos) == 1 && verbose {
		utils.PrintYaml(packageInfos[0].Ver)
		return nil
	}

	// 格式化输出包列表
	var dataList []*orderedmap.OrderedMap
	for _, p := range packageInfos {
		pkg := p.Ver
		row := Package_Columns{}
		row.PackageName = pkg.PackageName
		row.Os = pkg.Os
		row.Arch = pkg.Arch
		row.Size = fmt.Sprintf("%d", pkg.Size)
		row.Checksum = pkg.Checksum
		row.Algo = pkg.ChecksumAlgo
		row.Version = fmt.Sprintf("%d.%d.%d", pkg.VersionId.Major, pkg.VersionId.Minor, pkg.VersionId.Micro)
		row.Description = pkg.Description
		if p.Activated {
			row.A = "*"
		} else {
			row.A = " "
		}

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
	componentCmd.AddCommand(packageListCmd)
	packageListCmd.Flags().SortFlags = false
	packageListCmd.Example = packageListExample
	packageListCmd.Flags().StringVarP(&optListPackageName, "package", "p", "", "Package name")
	packageListCmd.Flags().BoolVarP(&optListVerbose, "verbose", "v", false, "Show details")
}
