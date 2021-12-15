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
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	internal "github.com/jjkirkpatrick/awsclihelper/internal"
	"github.com/logrusorgru/aurora"
	"github.com/spf13/cobra"
)

// ecsCmd represents the ecs command
var ecsCmd = &cobra.Command{
	Use:   "ecs",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		c, _ := internal.NewClient()
		c.CmdHeader()
		ecsConnect(c)
	},
}

func getClusters(c *internal.Client) string {

	input := &ecs.ListClustersInput{}

	result, err := c.ECS.ListClusters(context.TODO(), input)

	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	for _, cluster := range result.ClusterArns {
		fmt.Println(cluster)
	}

	if len(result.ClusterArns) == 1 {
		return result.ClusterArns[0]
	} else if len(result.ClusterArns) < 1 {
		fmt.Println(aurora.Bold(aurora.BrightRed("No clusters found, please check profile and region")))
		os.Exit(0)
	}

	choice := ""
	prompt := &survey.Select{
		Message: "Which ECS Cluster is the task in?:",
		Options: result.ClusterArns,
	}

	survey.AskOne(prompt, &choice)

	return choice

}

func getTasks(e *internal.Client, clusterArn string) string {
	input := &ecs.ListTasksInput{
		Cluster: &clusterArn,
	}
	result, err := e.ECS.ListTasks(context.TODO(), input)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if len(result.TaskArns) < 1 {
		fmt.Println(aurora.Bold(aurora.BrightRed("No tasks found")))
		os.Exit(1)
	}

	describeTaskinput := &ecs.DescribeTasksInput{
		Cluster: &clusterArn,
		Tasks:   result.TaskArns,
	}
	describeTaskResult, err := e.ECS.DescribeTasks(context.TODO(), describeTaskinput)
	if err != nil {
		fmt.Println(aurora.Bold(aurora.BrightRed("Error describing tasks")))
	}

	validTasks := []string{}
	for _, task := range describeTaskResult.Tasks {
		if task.Containers[0].ManagedAgents != nil {
			validTasks = append(validTasks, *task.Containers[0].Name+" : "+*task.TaskArn)
		}
	}

	choice := ""
	prompt := &survey.Select{
		Message: "Which ECS Task would you like to connect to?:",
		Options: validTasks,
	}

	survey.AskOne(prompt, &choice)

	return choice

}

func ecsConnect(c *internal.Client) {
	clusterArn := getClusters(c)
	task := strings.Split(getTasks(c, clusterArn), " : ")[1]
	fmt.Println(aurora.Bold(aurora.BrightGreen("Connecting to")), aurora.BrightCyan(task))

	arg0 := "aws"
	arg1 := "ecs"
	arg2 := "execute-command"
	arg3 := "--task=" + task
	arg4 := "--cluster=" + clusterArn
	arg5 := "--command=/bin/bash"
	arg6 := "--interactive"
	arg7 := "--region=" + c.Region
	arg8 := "--profile=" + c.Profile

	if err := internal.RunCommand(arg0, arg1, arg2, arg3, arg4, arg5, arg6, arg7, arg8); err != nil {
		return
	}

	fmt.Println(aurora.Bold(aurora.BrightGreen("Disconnected from ")), aurora.BrightCyan(task))

}

func init() {
	connectCmd.AddCommand(ecsCmd)
}
