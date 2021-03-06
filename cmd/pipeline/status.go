package pipeline

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/codepipeline"
	"github.com/aws/aws-sdk-go-v2/service/codepipeline/types"
	"github.com/inancgumus/screen"
	"github.com/jjkirkpatrick/awsclihelper/internal"
	"github.com/logrusorgru/aurora"
	"github.com/spf13/cobra"
)

type pipelineStatus struct {
	pipeline            string
	latestExecution     []types.PipelineExecutionSummary
	pipelineStageStates []types.StageState
}

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
		e := &pipelineStatus{}
		c, _ := internal.NewClient()
		c.CmdHeader()
		status(e, c)

	},
}

func header(region string, profile string) {
	fmt.Println("Running command against Region ", aurora.Cyan(region))
	fmt.Println("Running command against Profile ", aurora.Bold(aurora.Cyan(profile)))
}

func status(e *pipelineStatus, c *internal.Client) {
	pipeline := getPipelineToMonitor(c)

	if pipeline == "" {
		fmt.Println(aurora.Bold(aurora.BrightRed("Error getting Pipeline.")))
		return
	}

	e.pipeline = pipeline

	screen.Clear()
	screen.MoveTopLeft()
	fmt.Println("Monitoring Pipeline ", aurora.Bold(aurora.Cyan(pipeline)))
	getPipelineExecutions(e, c, true)
	getPipelineState(e, c, true)
	stage(e, c)

	return
}

func getPipelineToMonitor(c *internal.Client) string {

	// Get the first page of results for ListObjectsV2 for a bucket
	output, err := c.PIPELINE.ListPipelines(context.TODO(), &codepipeline.ListPipelinesInput{MaxResults: aws.Int32(100)})

	if err != nil {
		fmt.Println(aurora.Bold(aurora.BrightRed("Unable to get list of Pipelines, Please check the profile is correct, and that you are authenticated.")))
		os.Exit(1)
	}

	var pipelines []string

	for _, object := range output.Pipelines {
		pipelines = append(pipelines, *object.Name)
	}

	if len(pipelines) == 0 {
		fmt.Printf("No Pipelines found in Region %s with Profile %s \n", aurora.Green(c.Region), aurora.Green(c.Profile))
		fmt.Println(aurora.Bold(aurora.BrightRed("No Pipelines found, Please check the profile is correct, and that you are authenticated.")))
		os.Exit(1)
	}

	choice := ""
	prompt := &survey.Select{
		Message: "Choose a pipeline:",
		Options: pipelines,
	}

	survey.AskOne(prompt, &choice)

	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return ""
	}

	return choice
}

func getPipelineExecutions(e *pipelineStatus, c *internal.Client, writeToScreen bool) {
	if writeToScreen {
		screen.Clear()
		screen.MoveTopLeft()
		fmt.Println("Fetching data from AWS", aurora.Bold(aurora.Cyan("profile")))
	}

	output, err := c.PIPELINE.ListPipelineExecutions(context.TODO(), &codepipeline.ListPipelineExecutionsInput{
		PipelineName: aws.String(e.pipeline),
		MaxResults:   aws.Int32(1),
	})

	if err != nil {
		if writeToScreen {
			fmt.Println(aurora.Bold(aurora.BrightRed("Unable to get list of Pipelines, Please check the profile is correct, and that you are authenticated.")))
		}
		os.Exit(1)
	}

	status := output.PipelineExecutionSummaries[0].Status
	e.latestExecution = output.PipelineExecutionSummaries
	counter := 0
	for status != types.PipelineExecutionStatusInProgress {
		output, err := c.PIPELINE.ListPipelineExecutions(context.TODO(), &codepipeline.ListPipelineExecutionsInput{
			PipelineName: aws.String(e.pipeline),
			MaxResults:   aws.Int32(1),
		})

		if err != nil {
			if writeToScreen {
				fmt.Println(aurora.Bold(aurora.BrightRed("Unable to get list of Pipelines, Please check the profile is correct, and that you are authenticated.")))
			}
			os.Exit(1)
		}
		e.latestExecution = output.PipelineExecutionSummaries
		status = e.latestExecution[0].Status
		screen.MoveTopLeft()
		counter++
		if writeToScreen {
			fmt.Println(aurora.Sprintf(aurora.BrightYellow("Warning: No active AWS CodePipeline builds detected, polling for in progress build")))
		}
		time.Sleep(time.Second * 5)
	}

}

func getPipelineState(e *pipelineStatus, c *internal.Client, writeToScreen bool) {
	output, err := c.PIPELINE.GetPipelineState(context.TODO(), &codepipeline.GetPipelineStateInput{
		Name: aws.String(e.pipeline),
	})

	if err != nil {
		if writeToScreen {
			fmt.Println(aurora.Bold(aurora.BrightRed("Unable to get list of Pipelines, Please check the profile is correct, and that you are authenticated.")))
		}
		os.Exit(1)
	}

	e.pipelineStageStates = output.StageStates
	if writeToScreen {
		fmt.Println("Current Pipeline State: ", aurora.Bold(aurora.Cyan(*output.StageStates[0].StageName)))
	}
}

func stage(e *pipelineStatus, c *internal.Client) {
	fmt.Print("\033[?25l")
	screen.Clear()
	screen.MoveTopLeft()

	currentStatus := getCurrentPipelineState(e, c)
	for e.latestExecution[0].Status == types.PipelineExecutionStatusInProgress {
		screen.Clear()
		screen.MoveTopLeft()
		fmt.Println("Monitoring Pipeline: ", aurora.Bold(aurora.Cyan(e.pipeline)))
		fmt.Println("Current Pipeline State: ", aurora.Bold(aurora.Cyan(currentStatus)))

		//print pipeline status
		for _, stage := range e.pipelineStageStates {
			if stage.LatestExecution != nil && string(*stage.LatestExecution.PipelineExecutionId) == string(*e.latestExecution[0].PipelineExecutionId) && stage.LatestExecution.Status == "Succeeded" {
				fmt.Println(aurora.Sprintf(aurora.BrightGreen("Stage %s has completed %s"), string(*stage.StageName), stage.LatestExecution.Status))
			} else if stage.LatestExecution != nil && string(*stage.LatestExecution.PipelineExecutionId) == string(*e.latestExecution[0].PipelineExecutionId) && stage.LatestExecution.Status == "InProgress" {
				fmt.Println(aurora.Sprintf(aurora.BrightYellow("Stage %s is in progress"), string(*stage.StageName)))
			} else if stage.LatestExecution != nil && string(*stage.LatestExecution.PipelineExecutionId) == string(*e.latestExecution[0].PipelineExecutionId) && stage.LatestExecution.Status == "Failed" {
				fmt.Println(aurora.Sprintf(aurora.BrightRed("Stage %s has failed"), string(*stage.StageName)))
			} else {
				fmt.Println(aurora.Sprintf(aurora.BrightYellow("Stage %s not yet ran"), string(*stage.StageName)))
				continue
			}

			for _, action := range stage.ActionStates {
				if action.LatestExecution != nil && action.LatestExecution.Status == "Succeeded" {
					fmt.Println(aurora.Sprintf(aurora.BrightBlue("	Action %s has completed %s"), *action.ActionName, action.LatestExecution.Status))
				} else if action.LatestExecution != nil && action.LatestExecution.Status == "InProgress" {
					fmt.Println(aurora.Sprintf(aurora.BrightMagenta("	Action %s is in progress"), *action.ActionName))
					if *action.ActionName == "ApproveChangeSet" {
						manualApproval(e, c, *action.ActionName, *stage.StageName, *action.LatestExecution.Token)
					}
				} else if action.LatestExecution != nil && action.LatestExecution.Status == "Failed" {
					fmt.Println(aurora.Sprintf(aurora.BrightRed("	Action %s has failed"), *action.ActionName))
				} else {
					fmt.Println(aurora.Sprintf(aurora.BrightYellow("	Action %s not yet ran"), *action.ActionName))
				}
			}
		}

		currentStatus = getCurrentPipelineState(e, c)
		// if currentStatus != "InProgress"

		if currentStatus != types.PipelineExecutionStatusInProgress {
			pipelineComplete(currentStatus)
		}
		//print current status
		getPipelineState(e, c, false)
		time.Sleep(time.Second * 5)
	}

}

func getCurrentPipelineState(e *pipelineStatus, c *internal.Client) types.PipelineExecutionStatus {
	output, err := c.PIPELINE.GetPipelineExecution(context.TODO(), &codepipeline.GetPipelineExecutionInput{
		PipelineName:        &e.pipeline,
		PipelineExecutionId: e.latestExecution[0].PipelineExecutionId,
	})

	if err != nil {
		fmt.Println(aurora.Bold(aurora.BrightRed(err)))
	}

	return output.PipelineExecution.Status

}

func manualApproval(e *pipelineStatus, c *internal.Client, actionName string, stageName string, token string) {
	confirmation := true
	prompt := &survey.Confirm{
		Message: "Would you like to approve the change set",
		Default: true,
	}
	survey.AskOne(prompt, &confirmation)

	message := ""
	summery := &survey.Input{
		Message: "Change set summery",
	}
	survey.AskOne(summery, &message)

	approval := ""
	if confirmation {
		approval = "Approved"
	} else {
		approval = "Rejected"
	}

	// PutapprovalRequest
	_, err := c.PIPELINE.PutApprovalResult(context.TODO(), &codepipeline.PutApprovalResultInput{
		PipelineName: aws.String(e.pipeline),
		StageName:    &stageName,
		ActionName:   &actionName,
		Token:        &token,
		Result: &types.ApprovalResult{
			Status:  types.ApprovalStatus(approval),
			Summary: &message,
		},
	})

	if err != nil {
		fmt.Println(aurora.Sprintf(aurora.BrightRed("Unable to approve change set, Please check the profile is correct, and that you are authenticated.")))
		os.Exit(1)
	}

}

func pipelineComplete(status types.PipelineExecutionStatus) {
	screen.Clear()
	screen.MoveTopLeft()
	if status == types.PipelineExecutionStatusSucceeded {
		fmt.Println(aurora.Sprintf(aurora.BrightGreen("Pipeline has completed successfully")))
		os.Exit(0)
	} else if status == types.PipelineExecutionStatusFailed {
		fmt.Println(aurora.Sprintf(aurora.BrightRed("Pipeline has failed")))
		os.Exit(0)
	} else if status == types.PipelineExecutionStatusStopped {
		fmt.Println(aurora.Sprintf(aurora.BrightRed("Pipeline has been stopped")))
		os.Exit(0)
	}

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
