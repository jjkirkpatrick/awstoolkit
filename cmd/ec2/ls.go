package ec2

import (
	"fmt"

	"github.com/jjkirkpatrick/awsclihelper/internal"
	"github.com/spf13/cobra"
)

// statusCmd represents the status command
var exportCmd = &cobra.Command{
	Use:   "ls",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		c, _ := internal.NewClient()
		c.CmdHeader()
		test()

	},
}

func test() {
	fmt.Println("ec2 ls called")
}

func init() {
	ec2Cmd.AddCommand(exportCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// statusCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
