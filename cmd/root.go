package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "btptfexporter",
	Short: "Terraform exporter for BTP",
	Long: `btptfexporter is a utility to generate configuration for existing btp resources that are created manually and not managed by terraform. btptfexporter help to generate configuration which then can be used by terrraform to bring that resorce under terraform state.
	`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
