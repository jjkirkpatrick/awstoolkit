package internal

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"syscall"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/codepipeline"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/spf13/viper"
)

type Client struct {
	config   *aws.Config
	Profile  string
	Region   string
	EC2      *ec2.Client
	ECS      *ecs.Client
	SSM      *ssm.Client
	STS      *sts.Client
	PIPELINE *codepipeline.Client
}

func NewClient() (*Client, error) {
	config := newConfig()
	client := &Client{
		config:   config,
		Profile:  getProfile(),
		Region:   viper.GetString("region"),
		EC2:      ec2.NewFromConfig(*config),
		ECS:      ecs.NewFromConfig(*config),
		SSM:      ssm.NewFromConfig(*config),
		STS:      sts.NewFromConfig(*config),
		PIPELINE: codepipeline.NewFromConfig(*config),
	}
	return client, nil
}

func newConfig() *aws.Config {

	if !validateRegion(viper.GetString("region")) {
		fmt.Println("Invalid region")
		os.Exit(0)
	}

	profile := getProfile()

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(viper.GetString("region")),
		config.WithSharedConfigProfile(profile),
	)
	if err != nil {
		panic(err)
	}

	return &cfg
}

func getProfile() string {
	profile := viper.GetString("profile")
	if profile == "" {
		//read env var for AWS_DEFAULT_PROFILE
		profile = os.Getenv("AWS_DEFAULT_PROFILE")
	}

	if profile != "" {
		return profile
	}

	fmt.Println("No valid profile found. Either set AWS_DEFAULT_PROFILE, use --profile or add profile to config file")

	os.Exit(0)
	return ""
}

func validateRegion(region string) bool {
	reg, _ := regexp.Compile("^(us|eu|ap|sa|ca)\\-\\w+\\-\\d+$")
	regChina, _ := regexp.Compile("^cn\\-\\w+\\-\\d+$")
	regUsGov, _ := regexp.Compile("^us\\-gov\\-\\w+\\-\\d+$")

	return reg.MatchString(region) || regChina.MatchString(region) || regUsGov.MatchString(region)
}

func RunCommand(process string, args ...string) error {
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
