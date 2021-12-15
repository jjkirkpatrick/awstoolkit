package route53

import (
	"github.com/jjkirkpatrick/awsclihelper/cmd"
	"github.com/spf13/cobra"
)

// pipelineCmd represents the pipeline command
var route53Cmd = &cobra.Command{
	Use:   "route53",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	cmd.RootCmd.AddCommand(route53Cmd)
}
