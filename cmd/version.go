package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	version  = "0.9.5"
	codename = "XrayR"
	intro    = "A Xray backend that supports many panels"
)

func init() {
	// 注册子命令：输出版本信息
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print current version of XrayR",
		Run: func(cmd *cobra.Command, args []string) {
			showVersion()
		},
	})
}

func showVersion() {
	// 统一版本输出格式
	fmt.Printf("%s %s (%s) \n", codename, version, intro)
}
