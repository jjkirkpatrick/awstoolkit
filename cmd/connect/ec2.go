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
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/AlecAivazis/survey/v2"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/logrusorgru/aurora"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type ExecCommand struct {
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
		e := createExecCommand()
		connect(e)
	},
}

func createExecCommand() *ExecCommand {
	ssmClient, _ := createClient("ssm")
	_, ec2Client := createClient("ec2")
	e := &ExecCommand{
		region:    viper.GetString("region"),
		profile:   viper.GetString("profile"),
		ssmClient: ssmClient,
		ec2Client: ec2Client,
	}
	return e
}

func validateRegion(region string) bool {
	if region == "" {
		return false
	}
	return true
}

func validateProfile(profile string) bool {
	if profile == "" {
		return false
	}
	return true
}

func createClient(clientType string) (*ssm.Client, *ec2.Client) {
	region := viper.GetString("region")
	profile := viper.GetString("profile")

	// validate region and profile are set, else exit
	if !validateRegion(region) {
		fmt.Println(aurora.Bold(aurora.BrightRed("Region is required, but not set see -h flag")))
	}
	if !validateProfile(profile) {
		fmt.Println(aurora.Bold(aurora.BrightRed("Profile is required but not set see -h flag")))
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithSharedConfigProfile(profile),
	)

	if err != nil {
		log.Fatal(err)
	}

	if clientType == "ssm" {
		return ssm.NewFromConfig(cfg), nil
	} else if clientType == "ec2" {
		return nil, ec2.NewFromConfig(cfg)
	}
	return nil, nil
}

func getManagedInstances(e *ExecCommand) []string {
	result, err := e.ssmClient.DescribeInstanceInformation(context.TODO(), &ssm.DescribeInstanceInformationInput{})
	if err != nil {
		log.Fatal(err)
	}

	instanceIDs := []string{}

	for _, instance := range result.InstanceInformationList {
		instanceIDs = append(instanceIDs, *instance.InstanceId)
	}

	instanceInfo, err := e.ec2Client.DescribeInstances(context.TODO(), &ec2.DescribeInstancesInput{
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

func connect(e *ExecCommand) {
	fmt.Println(aurora.Bold(aurora.BrightGreen("EC2 Connect. Running with Profile ")), aurora.BrightCyan(viper.GetString("profile")), aurora.BrightGreen("and Region "), aurora.BrightCyan(viper.GetString("region")))
	managedInstances := getManagedInstances(e)

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
	arg6 := "--region=" + e.region
	arg7 := "--profile=" + e.profile

	if err := runCommand(arg0, arg1, arg2, arg3, arg4, arg5, arg6, arg7); err != nil {
		return
	}

	fmt.Println(aurora.Bold(aurora.BrightGreen("Disconnected from ")), aurora.BrightCyan(choice))

}

func runCommand(process string, args ...string) error {
	cmd := exec.Command(process, args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin

	// Capture any SIGINTs and discard them
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGINT)
	go func() {
		for {
			select {
			case <-sigs:
			}
		}
	}()
	defer close(sigs)

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func init() {
	connectCmd.AddCommand(ec2Cmd)
}
