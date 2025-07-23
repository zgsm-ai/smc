/*
Copyright © 2022 zbc <zbc@sangfor.com.cn>
*/
package cmd

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zgsm-ai/smc/internal/utils"
)

/**
 *	Build package descriptor file for executable
 */
func makePackage() error {
	size, md5str, err := utils.CalcFileMd5(optFrom)
	if err != nil {
		return err
	}
	priKey, err := os.ReadFile(optKeyFile)
	if err != nil {
		return err
	}
	data := utils.Sign(priKey, []byte(md5str))

	pkgData := &utils.PackageInfo{}
	pkgData.Arch = optArch
	pkgData.Os = optOs
	dir, fname := filepath.Split(optFrom)
	fnames := strings.Split(fname, ".")
	fields := strings.Split(fnames[0], "-")
	pkgData.PackageName = fmt.Sprintf("%s-%s-%s", fields[0], optOs, optArch)
	pkgData.Size = size
	pkgData.Checksum = md5str
	pkgData.ChecksumAlgo = "md5"
	pkgData.Sign = hex.EncodeToString(data)
	pkgData.VersionId, err = utils.ParseVersion(optVersion)
	if err != nil {
		return fmt.Errorf("parse version error: %v", err)
	}
	bytes, err := json.MarshalIndent(pkgData, "", "  ")
	if err != nil {
		return err
	}
	outputFname := optOutput
	if outputFname == "" {
		outputFname = filepath.Join(dir, fmt.Sprintf("package-%s-%s-%s.json", optOs, optArch, optVersion))
	}
	return os.WriteFile(outputFname, bytes, 0666)
}

// packageCmd represents the 'smc package' command
var packageCmd = &cobra.Command{
	Use:   "package",
	Short: "Generate package descriptor file for executable",
	Long:  `smc package signs a file and generates package.json`,

	Run: func(cmd *cobra.Command, args []string) {
		if err := makePackage(); err != nil {
			fmt.Println(err)
		}
	},
}

var optOs string
var optArch string
var optVersion string
var optOutput string
var optFrom string

func init() {
	rootCmd.AddCommand(packageCmd)

	packageCmd.Example = `  # Sign shenma.exe with private key shenma-private.key and generate package descriptor package-windows-amd64-1.0.1120.json
	 smc package -f ./shenma.exe -k shenma-private.key -s windows -a amd64 -v 1.0.1120`
	packageCmd.Flags().SortFlags = false
	packageCmd.Flags().StringVarP(&optFrom, "from", "f", "", "Executable file to sign")
	packageCmd.Flags().StringVarP(&optKeyFile, "key", "k", "", "Private key file")
	packageCmd.Flags().StringVarP(&optOs, "os", "s", "windows", "Target operating system")
	packageCmd.Flags().StringVarP(&optArch, "arch", "a", "amd64", "Target hardware architecture")
	packageCmd.Flags().StringVarP(&optVersion, "version", "v", "1.0.0", "Executable version number")
	packageCmd.Flags().StringVarP(&optOutput, "output", "o", "", "Output .json file")
	packageCmd.MarkFlagRequired("from")
	packageCmd.MarkFlagRequired("key")
}
