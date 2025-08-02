package pkg

import (
	"encoding/json"
	"fmt"
	"runtime"
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

func listRemotePackages(verbose bool, os string, arch string) error {
	// 格式化输出版本列表
	var dataList []*orderedmap.OrderedMap
	if optRemotePackageName != "" {
		ret, err := listRemotePackage(optRemotePackageName, verbose, os, arch)
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
			ret, err := listRemotePackage(pkg, verbose, os, arch)
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
func listRemotePackage(packageName string, verbose bool, os string, arch string) ([]*orderedmap.OrderedMap, error) {
	// 创建升级配置
	cfg := utils.UpgradeConfig{
		PackageName: packageName,
		Os:          os,
		Arch:        arch,
	}
	cfg.Correct()

	// 获取远程版本列表
	versList, err := utils.GetRemoteVersions(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to get remote versions: %v", err)
	}

	// 格式化输出版本列表
	var dataList []*orderedmap.OrderedMap
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
				pkgInfo, err := getPackageDetailInfo(cfg.BaseUrl + ver.InfoUrl)
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
		return listRemotePackages(optRemoteVerbose, optRemoteOs, optRemoteArch)
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
	packageRemoteCmd.Flags().StringVar(&optRemoteOs, "os", runtime.GOOS, "Target operating system")
	packageRemoteCmd.Flags().StringVar(&optRemoteArch, "arch", runtime.GOARCH, "Target architecture")
}
