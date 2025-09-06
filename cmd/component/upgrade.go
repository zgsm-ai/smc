package component

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/zgsm-ai/smc/cmd/common"
	"github.com/zgsm-ai/smc/internal/env"
	"github.com/zgsm-ai/smc/internal/utils"
)

func upgradePackage() error {
	var err error
	cfg := utils.UpgradeConfig{}
	cfg.BaseUrl = env.BaseUrl + "/costrict"
	cfg.PackageName = optUpgradePackageName
	if optPublicKey != "" {
		// 读取公钥文件内容
		keyData, err := os.ReadFile(optPublicKey)
		if err != nil {
			return fmt.Errorf("failed to read public key file: %v", err)
		}
		cfg.PublicKey = string(keyData)
	}
	cfg.Correct()

	var newVer *utils.VersionNumber
	if optUpgradeVersion != "" {
		var ver utils.VersionNumber
		ver, err = utils.ParseVersion(optUpgradeVersion)
		if err != nil {
			return err
		}
		newVer = &ver
	} else {
		newVer = nil
	}

	pkg, upgraded, err := utils.GetPackage(cfg, newVer)
	if err != nil {
		fmt.Printf("The '%s' upgrade failed: %v", cfg.PackageName, err)
		return err
	}
	if !upgraded {
		fmt.Printf("The '%s' version '%s' is up to date\n",
			cfg.PackageName, utils.PrintVersion(pkg.VersionId))
		return nil
	}
	if err := utils.ActivatePackage(cfg, pkg.VersionId); err != nil {
		if optUpgradePackageName == "smc" || optUpgradeSelf {
			// 当package选项未设置时，默认升级smc自身
			return upgradeSelf(cfg, pkg.VersionId)
		}
		fmt.Printf("The '%s' activate '%s' failed: %v",
			cfg.PackageName, utils.PrintVersion(pkg.VersionId), err)
		return err
	}
	fmt.Printf("The '%s' is upgraded to version %s\n", cfg.PackageName, utils.PrintVersion(pkg.VersionId))
	return nil
}

func upgradeSelf(cfg utils.UpgradeConfig, newVer utils.VersionNumber) error {
	cacheDir := filepath.Join(cfg.PackageDir, utils.PrintVersion(newVer))

	tmpFname := filepath.Join(cacheDir, "smc")
	targetFile := filepath.Join(cfg.InstallDir, "smc")
	if runtime.GOOS == "windows" {
		targetFile += ".exe"
		tmpFname += ".exe"
	}

	// 直接执行升级命令，不保存脚本文件
	fmt.Println("启动升级命令...")
	if runtime.GOOS == "windows" {
		// Windows: 使用 start 命令在新窗口中执行升级命令
		cmd := exec.Command("cmd", "/C", fmt.Sprintf("start /min cmd /C \"echo 升级smc... && timeout /t 3 /nobreak > nul && copy /Y \"%s\" \"%s\" && echo 升级完成\"",
			tmpFname, targetFile))
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("启动升级命令失败: %v", err)
		}
	} else {
		// Unix/Linux: 使用 nohup 在后台执行升级命令
		cmd := exec.Command("sh", "-c", fmt.Sprintf("nohup sh -c 'echo \"升级smc...\" && sleep 3 && cp -f \"%s\" \"%s\" && echo \"升级完成\"' > /dev/null 2>&1 &",
			tmpFname, targetFile))
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("启动升级命令失败: %v", err)
		}
	}

	fmt.Printf("smc正在后台升级到版本 %s\n", utils.PrintVersion(newVer))

	// 立即退出当前程序
	os.Exit(0)
	return nil // 这行代码不会执行，只是为了语法完整
}

var upgradeCmd = &cobra.Command{
	Use:   "upgrade {package-name | -p package-name}",
	Short: "Upgrade package",
	Long:  `Upgrade package`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 1 {
			optUpgradePackageName = args[0]
		}
		if err := common.InitCommonEnv(); err != nil {
			return err
		}
		if optUpgradePackageName == "" {
			optUpgradePackageName = "smc"
		}
		return upgradePackage()
	},
}

const upgradeExample = `  # upgrade package
  smc component upgrade -p codebase-syncer -v 1.0.0
  smc component upgrade -p codebase-syncer
  smc component upgrade -p codebase-syncer --public /path/to/public.key
  # upgrade smc itself
  smc component upgrade
  smc component upgrade --self`

var optUpgradeVersion string
var optUpgradePackageName string
var optPublicKey string
var optUpgradeSelf bool

func init() {
	componentCmd.AddCommand(upgradeCmd)
	upgradeCmd.Flags().SortFlags = false
	upgradeCmd.Example = upgradeExample

	upgradeCmd.Flags().StringVarP(&optUpgradePackageName, "package", "p", "", "package name")
	upgradeCmd.Flags().StringVarP(&optUpgradeVersion, "version", "v", "", "package version")
	upgradeCmd.Flags().StringVar(&optPublicKey, "public", "", "public key file for package verification")
	upgradeCmd.Flags().BoolVar(&optUpgradeSelf, "self", false, "upgrade smc itself")
}
