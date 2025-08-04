package pkg

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/iancoleman/orderedmap"
	"github.com/spf13/cobra"
	"github.com/zgsm-ai/smc/internal/utils"
)

/**
 *	格式化文件大小
 */
func formatSize(size uint64) string {
	if size < 1024 {
		return strconv.FormatUint(size, 10) + "B"
	} else if size < 1024*1024 {
		return strconv.FormatUint(size/1024, 10) + "KB"
	} else if size < 1024*1024*1024 {
		return strconv.FormatUint(size/(1024*1024), 10) + "MB"
	} else {
		return strconv.FormatUint(size/(1024*1024*1024), 10) + "GB"
	}
}

/**
 *	获取包详细元数据信息
 */
func getPackageDetailInfo(infoUrl string) (*utils.PackageInfo, error) {
	data, err := utils.GetBytes(infoUrl, nil)
	if err != nil {
		return nil, err
	}
	pkg := &utils.PackageInfo{}
	if err = json.Unmarshal(data, pkg); err != nil {
		return nil, fmt.Errorf("unmarshal package info error: %v", err)
	}
	return pkg, nil
}

/**
 *	Fields displayed in list format (non-verbose mode)
 */
type RemotePackage_Columns struct {
	PackageName string `json:"packageName"`
	Version     string `json:"version"`
	Os          string `json:"os"`
	Arch        string `json:"arch"`
}

/**
 *	Fields displayed in list format (verbose mode)
 */
type RemotePackage_Columns_Verbose struct {
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

func listPackages(verbose bool) error {
	// 格式化输出版本列表
	var dataList []*orderedmap.OrderedMap
	if optRemotePackageName != "" {
		ret, err := listPackage(optRemotePackageName, verbose)
		if err != nil {
			return err
		}
		dataList = append(dataList, ret...)
	} else {
		cfg := utils.UpgradeConfig{}
		cfg.Correct()
		packages, err := utils.GetRemotePackages(cfg)
		if err != nil {
			return err
		}
		for _, pkg := range packages.Packages {
			ret, err := listPackage(pkg, verbose)
			if err != nil {
				fmt.Printf("error: %v\n", err.Error())
			} else {
				dataList = append(dataList, ret...)
			}
		}
	}
	utils.PrintFormat(dataList)
	return nil
}

/**
 *	List remote package information
 */
func listPackage(packageName string, verbose bool) ([]*orderedmap.OrderedMap, error) {
	// 创建升级配置
	cfg := utils.UpgradeConfig{
		PackageName: packageName,
	}
	cfg.Correct()

	// 获取该软件包支持的所有平台
	platformList, err := utils.GetRemotePlatforms(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to get remote platforms: %v", err)
	}

	// 格式化输出版本列表
	var dataList []*orderedmap.OrderedMap

	// 遍历所有支持的平台，根据 os 和 arch 参数进行过滤
	for _, platform := range platformList.Platforms {
		// 如果 os 和 arch 都指定了，只显示匹配的平台
		if optRemoteOs != "" && optRemoteArch != "" {
			if platform.Os != optRemoteOs || platform.Arch != optRemoteArch {
				continue
			}
		} else if optRemoteOs != "" {
			// 如果只指定了 os，只显示匹配 os 的平台
			if platform.Os != optRemoteOs {
				continue
			}
		} else if optRemoteArch != "" {
			// 如果只指定了 arch，只显示匹配 arch 的平台
			if platform.Arch != optRemoteArch {
				continue
			}
		}
		// 如果 os 和 arch 都未指定，显示所有平台（不进行过滤）

		// 调用 listPlatform 函数搜集单个平台信息
		platformData, err := listPlatform(packageName, platform.Os, platform.Arch, verbose)
		if err != nil {
			fmt.Printf("Warning: failed to get platform data for %s/%s: %v\n", platform.Os, platform.Arch, err)
			continue
		}
		dataList = append(dataList, platformData...)
	}
	return dataList, nil
}

/**
 *	搜集单个平台信息
 */
func listPlatform(packageName, os, arch string, verbose bool) ([]*orderedmap.OrderedMap, error) {
	// 为平台创建特定的配置
	platformCfg := utils.UpgradeConfig{
		PackageName: packageName,
		Os:          os,
		Arch:        arch,
	}
	platformCfg.Correct()

	// 获取该平台的远程版本列表
	versList, err := utils.GetRemoteVersions(platformCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to get remote versions for platform %s/%s: %v", os, arch, err)
	}

	// 格式化输出版本列表
	var dataList []*orderedmap.OrderedMap

	// 遍历该平台的所有版本
	for _, ver := range versList.Versions {
		// verbose模式：显示所有字段
		row := RemotePackage_Columns_Verbose{}
		row.PackageName = versList.PackageName
		row.Os = versList.Os
		row.Arch = versList.Arch
		row.Version = utils.PrintVersion(ver.VersionId)

		if verbose {
			row.Size = "*"
			row.Checksum = "*"
			row.ChecksumAlgo = "*"
			row.Build = "*"
			row.Description = "*"
			// 获取版本的详细元数据
			if ver.InfoUrl != "" {
				pkgInfo, err := getPackageDetailInfo(platformCfg.BaseUrl + ver.InfoUrl)
				if err == nil {
					row.Size = formatSize(pkgInfo.Size)
					row.Checksum = pkgInfo.Checksum
					row.ChecksumAlgo = pkgInfo.ChecksumAlgo
					row.Build = pkgInfo.Build
					row.Description = pkgInfo.Description
				}
			}
		}
		recordMap, _ := utils.StructToOrderedMap(row)
		dataList = append(dataList, recordMap)
	}
	return dataList, nil
}

// packageRemoteCmd represents the 'smc package remote' command
var packageRemoteCmd = &cobra.Command{
	Use:   "remote {package | -p package} [--os os] [--arch arch]",
	Short: "List remote packages",
	Long:  `Lists remote packages available for download`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 1 {
			optRemotePackageName = args[0]
		}
		return listPackages(optRemoteVerbose)
	},
}

const packageRemoteExample = `  # List all remote packages
  smc package remote
  # List specific remote package
  smc package remote aip
  # Show remote package details
  smc package remote -p aip -v
  # List packages for specific OS and architecture
  smc package remote -p aip --os linux --arch amd64`

var optRemotePackageName string
var optRemoteVerbose bool
var optRemoteOs string
var optRemoteArch string

func init() {
	packageCmd.AddCommand(packageRemoteCmd)
	packageRemoteCmd.Flags().SortFlags = false
	packageRemoteCmd.Example = packageRemoteExample
	packageRemoteCmd.Flags().StringVarP(&optRemotePackageName, "package", "p", "", "Package name")
	packageRemoteCmd.Flags().BoolVarP(&optRemoteVerbose, "verbose", "v", false, "Show details")
	packageRemoteCmd.Flags().StringVar(&optRemoteOs, "os", "", "Target operating system")
	packageRemoteCmd.Flags().StringVar(&optRemoteArch, "arch", "", "Target architecture")
}
