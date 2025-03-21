package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/briandowns/spinner"
	"github.com/hupe1980/mwaacli/pkg/config"
	"github.com/hupe1980/mwaacli/pkg/local"
	"github.com/hupe1980/mwaacli/pkg/mwaa"
	"github.com/hupe1980/mwaacli/pkg/util"
	"github.com/spf13/cobra"
)

const (
	defaultVersion = "v2.10.3"
)

func newLocalCommand(globalOpts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "local",
		Short: "Setup and control the AWS MWAA local runner",
		Long:  `Manage the AWS MWAA local runner, including setup, starting, stopping, and checking the status.`,
	}

	cmd.AddCommand(newInitCommand(globalOpts))
	cmd.AddCommand(newBuildImageCommand(globalOpts))
	cmd.AddCommand(newStartCommand(globalOpts))
	cmd.AddCommand(newStopCommand(globalOpts))
	cmd.AddCommand(newTestRequirementsCommand(globalOpts))
	cmd.AddCommand(newPackageRequirementsCommand(globalOpts))
	cmd.AddCommand(newTestStartupScriptCommand(globalOpts))
	cmd.AddCommand(newSyncCommand(globalOpts))

	return cmd
}

func newInitCommand(_ *globalOptions) *cobra.Command {
	var (
		version string
		repoURL string
	)

	cmd := &cobra.Command{
		Use:           "init",
		Short:         "Initialize the AWS MWAA local runner",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.Println("Setting up the AWS MWAA local runner...")

			installer, err := local.NewInstaller(version, func(o *local.InstallerOptions) {
				o.RepoURL = repoURL
			})
			if err != nil {
				return fmt.Errorf("failed to create installer: %w", err)
			}

			if err := installer.Run(); err != nil {
				return err
			}

			cmd.Println("AWS MWAA local runner setup complete.")

			return nil
		},
	}

	cmd.Flags().StringVar(&version, "version", defaultVersion, "Specify the Airflow version for the AWS MWAA local runner")
	cmd.Flags().StringVar(&repoURL, "repo-url", local.MWAALocalRunnerRepoURL, "Specify the repository URL for the AWS MWAA local runner")

	return cmd
}

func newBuildImageCommand(_ *globalOptions) *cobra.Command {
	var version string

	cmd := &cobra.Command{
		Use:           "build-image",
		Short:         "Build the Docker image for the AWS MWAA local runner",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.Println("Building the Docker image for the AWS MWAA local runner...")

			runner, err := local.NewRunner(version)
			if err != nil {
				return fmt.Errorf("failed to create AWS MWAA local runner: %w", err)
			}
			defer runner.Close()

			ctx := context.Background()

			if err := runner.BuildImage(ctx); err != nil {
				return fmt.Errorf("failed to build Docker image: %w", err)
			}

			cmd.Println("Docker image built successfully.")

			return nil
		},
	}

	cmd.Flags().StringVar(&version, "version", defaultVersion, "Specify the Airflow version for the AWS MWAA local runner")

	return cmd
}

func newStartCommand(globalOpts *globalOptions) *cobra.Command {
	var (
		version string
		open    bool
		port    string
		resetDB bool
		//syncConfig bool
		awsCreds bool
		roleARN  string
	)

	cmd := &cobra.Command{
		Use:           "start",
		Short:         "Start the AWS MWAA local runner environment",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.Println("Starting the AWS MWAA local runner environment...")

			ctx := context.Background()

			runner, err := local.NewRunner(version)
			if err != nil {
				return fmt.Errorf("failed to create AWS MWAA local runner: %w", err)
			}
			defer runner.Close()

			if err := runner.BuildImage(ctx); err != nil {
				return fmt.Errorf("failed to build Docker image: %w", err)
			}

			cmd.Println("Docker image built successfully.")

			var credentials *local.AWSCredentials
			if awsCreds {
				creds, err := retrieveAWSCredentials(ctx, globalOpts.profile, globalOpts.region, roleARN)
				if err != nil {
					return err
				}

				credentials = creds
			}

			if err := runner.Start(ctx, func(o *local.StartOptions) {
				o.Port = port
				o.ResetDB = resetDB
				o.Envs = &local.Envs{
					Credentials: credentials,
				}
			}); err != nil {
				return fmt.Errorf("failed to start AWS MWAA local runner environment: %w", err)
			}

			webserverURL := fmt.Sprintf("http://localhost:%s", port)

			// Wait for the webserver to be ready
			//cmd.Println("Waiting for the Airflow webserver to be ready...")
			if err := waitForWebserver(ctx, runner, webserverURL); err != nil {
				return fmt.Errorf("application is not ready: %w", err)
			}

			if open {
				cmd.Println("Opening the Airflow UI in the default web browser...")
				if err := util.OpenBrowser(webserverURL); err != nil {
					return fmt.Errorf("failed to open the Airflow UI: %w", err)
				}
			} else {
				cmd.Printf("You can access the Airflow UI at %s\n", webserverURL)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&version, "version", defaultVersion, "Specify the Airflow version for the AWS MWAA local runner")
	cmd.Flags().BoolVar(&open, "open", false, "Open the Airflow UI in the default web browser after starting")
	cmd.Flags().BoolVar(&resetDB, "reset-db", false, "Reset the Airflow database before starting")
	//cmd.Flags().BoolVar(&syncConfig, "sync-config", false, "Sync the Airflow configuration before starting")
	cmd.Flags().StringVar(&port, "port", "8080", "Specify the port for the Airflow webserver")
	cmd.Flags().BoolVar(&awsCreds, "aws-creds", false, "Start the AWS MWAA local runner with AWS credentials")
	cmd.Flags().StringVar(&roleARN, "role-arn", "", "Specify the IAM Role ARN to use for the AWS MWAA local runner")

	return cmd
}

func newStopCommand(_ *globalOptions) *cobra.Command {
	var version string

	cmd := &cobra.Command{
		Use:           "stop",
		Short:         "Stop the AWS MWAA local runner environment",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.Println("Stopping the AWS MWAA local runner environment...")
			runner, err := local.NewRunner(version)
			if err != nil {
				return fmt.Errorf("failed to create AWS MWAA local runner: %w", err)
			}

			ctx := context.Background()

			if err := runner.Stop(ctx); err != nil {
				return fmt.Errorf("failed to stop AWS MWAA local runner environment: %w", err)
			}

			cmd.Println("AWS MWAA local runner environment stopped successfully.")

			return nil
		},
	}

	cmd.Flags().StringVar(&version, "version", defaultVersion, "Specify the Airflow version for the AWS MWAA local runner")

	return cmd
}

func newTestRequirementsCommand(_ *globalOptions) *cobra.Command {
	var version string

	cmd := &cobra.Command{
		Use:           "test-requirements",
		Short:         "Test installing requirements in an ephemeral container instance",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.Println("Testing requirements installation in an ephemeral container...")

			ctx := context.Background()

			runner, err := local.NewRunner(version)
			if err != nil {
				return fmt.Errorf("failed to create AWS MWAA local runner: %w", err)
			}
			defer runner.Close()

			// Ensure image is built
			if err := runner.BuildImage(ctx); err != nil {
				return fmt.Errorf("failed to build Docker image: %w", err)
			}

			if err := runner.TestRequirements(ctx); err != nil {
				return fmt.Errorf("failed to test requirements installation: %w", err)
			}

			cmd.Println("Requirements installed successfully in the test container.")

			return nil
		},
	}

	cmd.Flags().StringVar(&version, "version", defaultVersion, "Specify the Airflow version for testing requirements")

	return cmd
}

func newPackageRequirementsCommand(_ *globalOptions) *cobra.Command {
	var version string

	cmd := &cobra.Command{
		Use:           "package-requirements",
		Short:         "Package Python requirements into a ZIP file for AWS MWAA",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.Println("Packaging Python requirements into a ZIP file...")

			ctx := context.Background()

			runner, err := local.NewRunner(version)
			if err != nil {
				return fmt.Errorf("failed to create AWS MWAA local runner: %w", err)
			}
			defer runner.Close()

			// Ensure image is built
			if err := runner.BuildImage(ctx); err != nil {
				return fmt.Errorf("failed to build Docker image: %w", err)
			}

			if err := runner.PackageRequirements(ctx); err != nil {
				return fmt.Errorf("failed to package requirements: %w", err)
			}

			cmd.Println("Python requirements packaged successfully.")

			return nil
		},
	}

	cmd.Flags().StringVar(&version, "version", defaultVersion, "Specify the Airflow version for packaging requirements")

	return cmd
}

func newTestStartupScriptCommand(globalOpts *globalOptions) *cobra.Command {
	var (
		version  string
		awsCreds bool
		roleARN  string
	)

	cmd := &cobra.Command{
		Use:           "test-startup-script",
		Short:         "Test executing the startup script in an ephemeral container",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.Println("Testing startup script execution in an ephemeral container...")

			ctx := context.Background()
			runner, err := local.NewRunner(version)
			if err != nil {
				return fmt.Errorf("failed to create AWS MWAA local runner: %w", err)
			}
			defer runner.Close()

			// Ensure image is built
			if err := runner.BuildImage(ctx); err != nil {
				return fmt.Errorf("failed to build Docker image: %w", err)
			}

			var credentials *local.AWSCredentials
			if awsCreds {
				creds, err := retrieveAWSCredentials(ctx, globalOpts.profile, globalOpts.region, roleARN)
				if err != nil {
					return err
				}

				credentials = creds
			}

			if err := runner.TestStartupScript(ctx, func(o *local.TestStartupScriptOptions) {
				o.Envs = &local.Envs{
					Credentials: credentials,
				}
			}); err != nil {
				return fmt.Errorf("failed to execute startup script: %w", err)
			}

			cmd.Println("Startup script executed successfully in the test container.")

			return nil
		},
	}

	cmd.Flags().StringVar(&version, "version", defaultVersion, "Specify the Airflow version for testing the startup script")
	cmd.Flags().BoolVar(&awsCreds, "aws-creds", false, "Start the AWS MWAA local runner with AWS credentials")
	cmd.Flags().StringVar(&roleARN, "role-arn", "", "Specify the IAM Role ARN to use for the AWS MWAA local runner")

	return cmd
}

func newSyncCommand(globalOpts *globalOptions) *cobra.Command {
	var (
		version     string
		awsCreds    bool
		roleARN     string
		mwaaEnvName string
	)

	cmd := &cobra.Command{
		Use:           "sync",
		Short:         "",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.NewConfig(globalOpts.profile, globalOpts.region)
			if err != nil {
				return err
			}

			mwaaClient := mwaa.NewClient(cfg)

			ctx := context.Background()
			if mwaaEnvName == "" {
				mwaaEnvName, err = getEnvironment(ctx, mwaaClient)
				if err != nil {
					return err
				}
			}

			environment, err := mwaaClient.GetEnvironment(ctx, mwaaEnvName)
			if err != nil {
				return err
			}

			// Extract bucket name from ARN
			bucketArn := aws.ToString(environment.SourceBucketArn)
			bucketName := strings.Split(bucketArn, ":")[5] // Extracts the bucket name

			//s3Client := s3.NewClient(cfg)

			if pluginsPath := environment.PluginsS3Path; pluginsPath != nil {
				cmd.Printf("Remote Plugins Path: s3://%s/%s\n", bucketName, aws.ToString(pluginsPath))
			} else {
				cmd.Println("No remote plugins path configured.")
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&version, "version", defaultVersion, "Specify the Airflow version for testing the startup script")
	cmd.Flags().BoolVar(&awsCreds, "aws-creds", false, "Start the AWS MWAA local runner with AWS credentials")
	cmd.Flags().StringVar(&roleARN, "role-arn", "", "Specify the IAM Role ARN to use for the AWS MWAA local runner")

	cmd.Flags().StringVar(&mwaaEnvName, "env", "", "MWAA environment name")

	return cmd
}

func waitForWebserver(ctx context.Context, runner *local.Runner, webserverURL string) error {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Prefix = "Waiting for the Airflow webserver to be ready... "
	s.Start()

	defer s.Stop()

	// Wait for the webserver to be ready
	err := runner.WaitForWebserverReady(ctx, fmt.Sprintf("%s/health", webserverURL))
	if err != nil {
		return fmt.Errorf("application is not ready: %w", err)
	}

	s.FinalMSG = "AWS MWAA local runner environment started successfully!\n"

	return nil
}

// retrieveAWSCredentials retrieves AWS credentials based on the provided profile, region, and optional role ARN.
func retrieveAWSCredentials(ctx context.Context, profile, region, roleARN string) (*local.AWSCredentials, error) {
	// Load AWS configuration
	cfg, err := config.NewConfig(profile, region)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS configuration: %w", err)
	}

	// If a role ARN is provided, assume the role
	if roleARN != "" {
		if err := util.IsValidARN(roleARN); err != nil {
			return nil, fmt.Errorf("invalid role ARN: %w", err)
		}

		stsClient := sts.NewFromConfig(cfg.AWSConfig)

		output, err := stsClient.AssumeRole(ctx, &sts.AssumeRoleInput{
			RoleArn:         aws.String(roleARN),
			RoleSessionName: aws.String("mwaacli"),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to assume role: %w", err)
		}

		return &local.AWSCredentials{
			Credentials: aws.Credentials{
				AccessKeyID:     aws.ToString(output.Credentials.AccessKeyId),
				SecretAccessKey: aws.ToString(output.Credentials.SecretAccessKey),
				SessionToken:    aws.ToString(output.Credentials.SessionToken),
			},
			Region: cfg.Region,
		}, nil
	}

	// Otherwise, retrieve default credentials
	creds, err := cfg.AWSConfig.Credentials.Retrieve(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve AWS credentials: %w", err)
	}

	return &local.AWSCredentials{
		Credentials: creds,
		Region:      cfg.Region,
	}, nil
}
