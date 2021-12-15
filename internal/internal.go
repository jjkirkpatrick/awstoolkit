package internal

import (
	"context"
	"errors"
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
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/logrusorgru/aurora"
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
	R53      *route53.Client
}

func NewClient() (*Client, error) {
	config := newConfig()
	Profile, _ := getProfile()
	client := &Client{
		config:   config,
		Region:   viper.GetString("region"),
		Profile:  Profile,
		EC2:      ec2.NewFromConfig(*config),
		ECS:      ecs.NewFromConfig(*config),
		SSM:      ssm.NewFromConfig(*config),
		STS:      sts.NewFromConfig(*config),
		PIPELINE: codepipeline.NewFromConfig(*config),
		R53:      route53.NewFromConfig(*config),
	}
	return client, nil
}

func newConfig() *aws.Config {
	if !validateRegion(viper.GetString("region")) {
		fmt.Println("Invalid region")
		os.Exit(0)
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(viper.GetString("region")),
	)

	profile, err := getProfile()
	if err == nil {
		config.WithSharedConfigProfile(profile)
	}

	if !testCredentials(&cfg) {
		fmt.Println(aurora.BrightRed("Credentials are Invalid, please check your credentials use --profile to specify profile"))
		os.Exit(0)
	}

	return &cfg
}

func testCredentials(cfg *aws.Config) bool {
	client := sts.NewFromConfig(*cfg)
	_, err := client.GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})
	if err != nil {
		return false
	}
	return true
}

func getProfile() (string, error) {
	profile := viper.GetString("profile")
	if profile == "" {
		//read env var for AWS_DEFAULT_PROFILE
		profile = os.Getenv("AWS_DEFAULT_PROFILE")
		viper.Set("Profile", profile)
	}

	if profile != "" {
		return profile, nil
	} else {
		return "", errors.New("No profile found")
	}

}

func validateRegion(region string) bool {
	reg, _ := regexp.Compile("^(us|eu|ap|sa|ca)\\-\\w+\\-\\d+$")
	regChina, _ := regexp.Compile("^cn\\-\\w+\\-\\d+$")
	regUsGov, _ := regexp.Compile("^us\\-gov\\-\\w+\\-\\d+$")

	return reg.MatchString(region) || regChina.MatchString(region) || regUsGov.MatchString(region)
}

func (c *Client) CmdHeader() {

	if c.Profile != "" {
		fmt.Println(aurora.Bold(aurora.BrightGreen("Running with Profile ")), aurora.BrightCyan(viper.GetString("profile")), aurora.BrightGreen("and Region "), aurora.BrightCyan(viper.GetString("region")))
	} else {
		fmt.Println(aurora.Bold(aurora.BrightGreen("Running with")), aurora.BrightCyan("Default Credentials"), aurora.BrightGreen("and Region "), aurora.BrightCyan(viper.GetString("region")))
	}

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
