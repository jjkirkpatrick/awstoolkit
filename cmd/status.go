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
package cmd

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/codepipeline"
	"github.com/logrusorgru/aurora"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		//get region and profile from config or cli
		Region := cmd.Flag("region").Value.String()
		Profile := cmd.Flag("profile").Value.String()
		fmt.Println(aurora.Bold(aurora.Cyan("AWS CodePipeline Status")))

		// va;idate region and profile are set, else exit
		if !validateRegion(Region) {
			fmt.Println(aurora.Bold(aurora.BrightRed("Region is required, but not set see -h flag")))
			return
		}
		if !validateProfile(Profile) {
			fmt.Println(aurora.Bold(aurora.BrightRed("Profile is required but not set see -h flag")))
			return
		}

		header(Region, Profile)
		status(args, Region, Profile)
	},
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

func header(region string, profile string) {
	fmt.Println("Running command against Region ", aurora.Magenta(region))
	fmt.Println("Running command against Profile ", aurora.Bold(aurora.Cyan(profile)))
}

func status(args []string, region, profile string) {
	pipeline := getPipelineToMonitor(args, region, profile)

	fmt.Println("Pipeline: " + pipeline)

}

func getPipelineToMonitor(args []string, region string, profile string) string {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithSharedConfigProfile(profile),
	)

	if err != nil {
		log.Fatal(err)
	}

	// Create an Amazon S3 service client
	client := codepipeline.NewFromConfig(cfg)

	// Get the first page of results for ListObjectsV2 for a bucket
	output, err := client.ListPipelines(context.TODO(), &codepipeline.ListPipelinesInput{MaxResults: aws.Int32(100)})

	if err != nil {
		fmt.Println(aurora.Bold(aurora.BrightRed("Unable to get list of Pipelines, Please check the profile is correct, and that you are authenticated.")))
		os.Exit(1)
	}

	var pipelines []string

	log.Println("first page results:")
	for _, object := range output.Pipelines {
		pipelines = append(pipelines, *object.Name)
	}

	prompt := promptui.Select{
		Label: "Pipeline",
		Items: pipelines,
	}

	_, result, err := prompt.Run()

	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return ""
	}

	return result
}

func init() {
	pipelineCmd.AddCommand(statusCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// statusCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
