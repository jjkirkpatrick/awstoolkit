/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package connect

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	internal "github.com/jjkirkpatrick/awsclihelper/internal"
	"github.com/logrusorgru/aurora"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type execCommand struct {
	ssmClient *ssm.Client
	ec2Client *ec2.Client
	args      []string
	region    string
	profile   string
}

// ec2Cmd represents the ec2 command
var ec2Cmd = &cobra.Command{
	Use:   "ec2",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		c, _ := internal.NewClient()
		c.CmdHeader()
		connect(c)
	},
}

func getManagedInstances(c *internal.Client) []string {
	result, err := c.SSM.DescribeInstanceInformation(context.TODO(), &ssm.DescribeInstanceInformationInput{})
	if err != nil {
		log.Fatal(err)
	}

	instanceIDs := []string{}

	for _, instance := range result.InstanceInformationList {
		instanceIDs = append(instanceIDs, *instance.InstanceId)
	}

	instanceInfo, err := c.EC2.DescribeInstances(context.TODO(), &ec2.DescribeInstancesInput{
		InstanceIds: instanceIDs,
	})
	if err != nil {
		log.Fatal(err)
	}
	managedInstances := []string{}
	for _, reservation := range instanceInfo.Reservations {
		for _, instance := range reservation.Instances {
			for _, tag := range instance.Tags {
				if *tag.Key == "Name" {
					managedInstances = append(managedInstances, *instance.InstanceId+" : "+*tag.Value)
				}
			}
		}
	}
	return managedInstances
}

func connect(c *internal.Client) {
	fmt.Println(aurora.Bold(aurora.BrightGreen("EC2 Connect. Running with Profile ")), aurora.BrightCyan(viper.GetString("profile")), aurora.BrightGreen("and Region "), aurora.BrightCyan(viper.GetString("region")))
	managedInstances := getManagedInstances(c)

	choice := ""
	prompt := &survey.Select{
		Message: "Choose a pipeline:",
		Options: managedInstances,
	}

	survey.AskOne(prompt, &choice)

	fmt.Println(aurora.Bold(aurora.BrightGreen("Connecting to ")), aurora.BrightCyan(choice))
	instanceID := strings.Split(choice, " : ")[0]
	arg0 := "aws"
	arg1 := "ssm"
	arg2 := "start-session"
	arg3 := "--target=" + instanceID
	arg4 := "--document-name=AWS-StartInteractiveCommand"
	arg5 := "--parameters=command=[\"sudo -i -u root\"]"
	arg6 := "--region=" + c.Region
	arg7 := "--profile=" + c.Profile

	if err := internal.RunCommand(arg0, arg1, arg2, arg3, arg4, arg5, arg6, arg7); err != nil {
		return
	}

	fmt.Println(aurora.Bold(aurora.BrightGreen("Disconnected from ")), aurora.BrightCyan(choice))

}

func init() {
	connectCmd.AddCommand(ec2Cmd)
}
