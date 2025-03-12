package config

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// Config holds AWS configuration details including credentials, region, and account information.
type Config struct {
	// Account is the AWS account ID number associated with the credentials.
	Account string

	// Profile is the AWS shared configuration profile being used.
	Profile string

	// Region specifies the AWS region for API requests.
	Region string

	// AWSConfig provides service configuration for AWS SDK clients.
	AWSConfig aws.Config
}

// NewConfig initializes a new AWS configuration instance with the given profile and region.
// It loads credentials, configures the region, and retrieves the AWS account ID.
func NewConfig(profile string, region string) (*Config, error) {
	awsCfg, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithRegion(region),
		config.WithSharedConfigProfile(profile),
		config.WithAssumeRoleCredentialOptions(func(aro *stscreds.AssumeRoleOptions) {
			aro.TokenProvider = stscreds.StdinTokenProvider
		}),
	)
	if err != nil {
		return nil, err
	}

	// Create an STS client to retrieve the AWS account ID.
	client := sts.NewFromConfig(awsCfg)

	output, err := client.GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, err
	}

	return &Config{
		Account:   aws.ToString(output.Account),
		Profile:   profile,
		Region:    awsCfg.Region,
		AWSConfig: awsCfg,
	}, nil
}
