/*
Copyright © 2022 zbc <zbc@sangfor.com.cn>
*/
package cert

import (
	"encoding/hex"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zgsm-ai/smc/internal/utils"
)

func signSoftware() error {
	_, md5str, err := utils.CalcFileMd5(optTarget)
	if err != nil {
		return err
	}
	priKey, err := os.ReadFile(optKeyFile)
	if err != nil {
		return err
	}
	data := utils.Sign(priKey, []byte(md5str))
	fmt.Printf("%s\n", hex.EncodeToString(data))
	return nil
}

// signCmd represents the 'smc sign' command
var signCmd = &cobra.Command{
	Use:   "sign",
	Short: "Sign a file to prevent tampering",
	Long:  `'smc sign' signs a file to prevent tampering`,

	Run: func(cmd *cobra.Command, args []string) {
		if err := signSoftware(); err != nil {
			fmt.Println(err)
		}
	},
}

var optKeyFile string
var optTarget string

func init() {
	certCmd.AddCommand(signCmd)

	signCmd.Example = `  # Sign the shenma file using private key shenma-private.key to get signature string
  smc cert sign -k shenma-private.key -t ./shenma`
	signCmd.Flags().SortFlags = false
	signCmd.Flags().StringVarP(&optKeyFile, "key", "k", "", "Private key file")
	signCmd.Flags().StringVarP(&optTarget, "target", "t", "", "Target file to sign")
	signCmd.MarkFlagRequired("target")
	signCmd.MarkFlagRequired("key")
}
