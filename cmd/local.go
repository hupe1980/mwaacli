package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
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
	cmd.AddCommand(newDiffCommand(globalOpts))

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
			cmd.Println(cyan("[INFO]"), "Setting up the AWS MWAA local runner...")

			installer, err := local.NewInstaller(version, func(o *local.InstallerOptions) {
				o.RepoURL = repoURL
			})
			if err != nil {
				return fmt.Errorf("failed to create installer: %w", err)
			}

			if err := installer.Run(); err != nil {
				return err
			}

			cmd.Println(green("[SUCCESS]"), "AWS MWAA local runner setup complete.")

			return nil
		},
	}

	cmd.Flags().StringVar(&version, "version", defaultVersion, "Specify the Airflow version for the AWS MWAA local runner")
	cmd.Flags().StringVar(&repoURL, "repo-url", local.MWAALocalRunnerRepoURL, "Specify the repository URL for the AWS MWAA local runner")

	return cmd
}

func newBuildImageCommand(_ *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "build-image",
		Short:         "Build the Docker image for the AWS MWAA local runner",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.Println(cyan("[INFO]"), "Building the Docker image for the AWS MWAA local runner...")

			runner, err := local.NewRunner()
			if err != nil {
				return fmt.Errorf("failed to create AWS MWAA local runner: %w", err)
			}
			defer runner.Close()

			ctx := context.Background()

			if err := runner.BuildImage(ctx); err != nil {
				return fmt.Errorf("failed to build Docker image: %w", err)
			}

			cmd.Println(green("[SUCCESS]"), "Docker image built successfully.")

			return nil
		},
	}

	return cmd
}

func newStartCommand(globalOpts *globalOptions) *cobra.Command {
	var (
		noBrowser  bool
		port       string
		resetDB    bool
		awsCreds   bool
		roleARN    string
		followLogs bool
		waitTime   time.Duration // Add wait time flag
	)

	cmd := &cobra.Command{
		Use:           "start",
		Short:         "Start the AWS MWAA local runner environment",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.Println(cyan("[INFO]"), "Starting the AWS MWAA local runner environment...")

			ctx := context.Background()

			runner, err := local.NewRunner()
			if err != nil {
				return fmt.Errorf("failed to create AWS MWAA local runner: %w", err)
			}
			defer runner.Close()

			if err := runner.BuildImage(ctx); err != nil {
				return fmt.Errorf("failed to build Docker image: %w", err)
			}

			cmd.Println(green("[SUCCESS]"), "Docker image built successfully.")

			envs := &local.Envs{}

			if awsCreds {
				creds, err := retrieveAWSCredentials(ctx, globalOpts.profile, globalOpts.region, roleARN)
				if err != nil {
					return err
				}

				envs.Credentials = creds
			}

			containerID, err := runner.Start(ctx, func(o *local.StartOptions) {
				o.Port = port
				o.ResetDB = resetDB
				o.Envs = envs
			})
			if err != nil {
				return fmt.Errorf("failed to start AWS MWAA local runner environment: %w", err)
			}

			webserverURL := fmt.Sprintf("http://localhost:%s", port)

			// Wait for the webserver to be ready
			if err := waitForWebserver(ctx, runner, webserverURL, waitTime); err != nil {
				cmd.Println(cyan("[INFO]"), "Webserver did not become healthy within the wait time. Stopping the containers...")

				// Attempt to stop the container gracefully
				stopErr := runner.Stop(ctx)
				if stopErr != nil {
					cmd.Println(cyan("[ERROR]"), "Failed to stop the containers:", stopErr)
				}

				return fmt.Errorf("application is not ready: %w", err)
			}

			if !noBrowser {
				cmd.Println(cyan("[INFO]"), "Opening the Airflow UI in the default web browser...")
				if err := util.OpenBrowser(webserverURL); err != nil {
					return fmt.Errorf("failed to open the Airflow UI: %w", err)
				}
			} else {
				cmd.Printf(green("[SUCCESS]"), "You can access the Airflow UI at %s\n", webserverURL)
			}

			if followLogs {
				cmd.Println(cyan("[INFO]"), "Following the logs of the Airflow webserver and scheduler...")

				logsCtx, cancel := context.WithCancel(ctx)
				defer cancel() // Ensure cancellation on exit

				// Handle OS signals for graceful shutdown
				sigChan := make(chan os.Signal, 1)
				signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
				go func() {
					<-sigChan
					cmd.Println(cyan("[INFO]"), "Received shutdown signal, stopping local runner...")
					cancel()
				}()

				// Run logs in a separate goroutine
				logsErr := make(chan error, 1)
				go func() {
					logsErr <- runner.Logs(logsCtx, containerID)
				}()

				select {
				case <-logsCtx.Done(): // Exit on context cancellation
					cmd.Println(cyan("[INFO]"), "Shutting down AWS MWAA local runner...")
					if err := runner.Stop(ctx); err != nil {
						return fmt.Errorf("failed to stop AWS MWAA local runner environment: %w", err)
					}

					cmd.Println(green("[SUCCESS]"), "AWS MWAA local runner environment stopped successfully.")
				case err := <-logsErr: // Capture log errors
					if err != nil {
						return fmt.Errorf("failed to follow logs: %w", err)
					}
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&noBrowser, "no-browser", false, "Do not open the Airflow UI in the default web browser after starting")
	cmd.Flags().BoolVar(&followLogs, "follow-logs", false, "Follow the logs of the Airflow webserver and scheduler")
	cmd.Flags().BoolVar(&resetDB, "reset-db", false, "Reset the Airflow database before starting")
	cmd.Flags().StringVar(&port, "port", "8080", "Specify the port for the Airflow webserver")
	cmd.Flags().BoolVar(&awsCreds, "aws-creds", false, "Start the AWS MWAA local runner with AWS credentials")
	cmd.Flags().StringVar(&roleARN, "role-arn", "", "Specify the IAM Role ARN to use for the AWS MWAA local runner")
	cmd.Flags().DurationVar(&waitTime, "wait", 5*time.Minute, "Amount of time to wait for the webserver to get healthy before timing out (e.g., 30s, 5m).")

	return cmd
}

func newStopCommand(_ *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "stop",
		Short:         "Stop the AWS MWAA local runner environment",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.Println(cyan("[INFO]"), "Stopping the AWS MWAA local runner environment...")

			runner, err := local.NewRunner()
			if err != nil {
				return fmt.Errorf("failed to create AWS MWAA local runner: %w", err)
			}

			ctx := context.Background()

			if err := runner.Stop(ctx); err != nil {
				return fmt.Errorf("failed to stop AWS MWAA local runner environment: %w", err)
			}

			cmd.Println(green("[SUCCESS]"), "AWS MWAA local runner environment stopped successfully.")

			return nil
		},
	}

	return cmd
}

func newTestRequirementsCommand(_ *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "test-requirements",
		Short:         "Test installing requirements in an ephemeral container instance",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.Println(cyan("[INFO]"), "Testing requirements installation in an ephemeral container...")

			runner, err := local.NewRunner()
			if err != nil {
				return fmt.Errorf("failed to create AWS MWAA local runner: %w", err)
			}
			defer runner.Close()

			ctx := context.Background()

			// Ensure image is built
			if err := runner.BuildImage(ctx); err != nil {
				return fmt.Errorf("failed to build Docker image: %w", err)
			}

			if err := runner.TestRequirements(ctx); err != nil {
				return fmt.Errorf("failed to test requirements installation: %w", err)
			}

			cmd.Println(green("[SUCCESS]"), "Requirements installed successfully in the test container.")

			return nil
		},
	}

	return cmd
}

func newPackageRequirementsCommand(_ *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "package-requirements",
		Short:         "Package Python requirements into a ZIP file for AWS MWAA",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.Println(cyan("[INFO]"), "Packaging Python requirements into a ZIP file...")

			runner, err := local.NewRunner()
			if err != nil {
				return fmt.Errorf("failed to create AWS MWAA local runner: %w", err)
			}
			defer runner.Close()

			ctx := context.Background()

			// Ensure image is built
			if err := runner.BuildImage(ctx); err != nil {
				return fmt.Errorf("failed to build Docker image: %w", err)
			}

			if err := runner.PackageRequirements(ctx); err != nil {
				return fmt.Errorf("failed to package requirements: %w", err)
			}

			cmd.Println(green("[SUCCESS]"), "Python requirements packaged successfully.")

			return nil
		},
	}

	return cmd
}

func newTestStartupScriptCommand(globalOpts *globalOptions) *cobra.Command {
	var (
		awsCreds bool
		roleARN  string
	)

	cmd := &cobra.Command{
		Use:           "test-startup-script",
		Short:         "Test executing the startup script in an ephemeral container",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.Println(cyan("[INFO]"), "Testing startup script execution in an ephemeral container...")

			runner, err := local.NewRunner()
			if err != nil {
				return fmt.Errorf("failed to create AWS MWAA local runner: %w", err)
			}
			defer runner.Close()

			ctx := context.Background()

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

			cmd.Println(green("[SUCCESS]"), "Startup script executed successfully in the test container.")

			return nil
		},
	}

	cmd.Flags().BoolVar(&awsCreds, "aws-creds", false, "Start the AWS MWAA local runner with AWS credentials")
	cmd.Flags().StringVar(&roleARN, "role-arn", "", "Specify the IAM Role ARN to use for the AWS MWAA local runner")

	return cmd
}

func newSyncCommand(globalOpts *globalOptions) *cobra.Command {
	var (
		awsCreds bool
		roleARN  string
	)

	cmd := &cobra.Command{
		Use:           "sync",
		Short:         "",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.Println(cyan("[INFO]"), "Syncing the Airflow configuration with the remote MWAA environment...")

			cfg, err := config.NewConfig(globalOpts.profile, globalOpts.region)
			if err != nil {
				return err
			}

			mwaaClient := mwaa.NewClient(cfg)

			ctx := context.Background()

			mwaaEnvName, err := getEnvironment(ctx, mwaaClient)
			if err != nil {
				return err
			}

			environment, err := mwaaClient.GetEnvironment(ctx, mwaaEnvName)
			if err != nil {
				return err
			}

			syncer := local.NewSyncer(cfg)

			// Extract bucket name from ARN
			bucketArn := aws.ToString(environment.SourceBucketArn)
			bucketName := strings.Split(bucketArn, ":")[5] // Extracts the bucket name

			if startupScriptPath := environment.StartupScriptS3Path; startupScriptPath != nil {
				cmd.Printf("Remote Startup Script: s3://%s/%s\n", bucketName, aws.ToString(startupScriptPath))
				if err := syncer.SyncStartupScript(ctx, &local.SyncStartupScriptInput{
					Bucket:  aws.String(bucketName),
					Key:     startupScriptPath,
					Version: environment.StartupScriptS3ObjectVersion,
				}); err != nil {
					return fmt.Errorf("failed to sync startup script: %w", err)
				}
				cmd.Println("Startup script synced successfully.")
			} else {
				cmd.Println("No remote startup script configured.")
			}

			if requirementsFile := environment.RequirementsS3Path; requirementsFile != nil {
				cmd.Printf("Remote Requirements File: s3://%s/%s\n", bucketName, aws.ToString(requirementsFile))
				if err := syncer.SyncRequirementsTXT(ctx, &local.SyncRequirementsTXTInput{
					Bucket:  aws.String(bucketName),
					Key:     requirementsFile,
					Version: environment.RequirementsS3ObjectVersion,
				}); err != nil {
					return fmt.Errorf("failed to sync requirements file: %w", err)
				}
				cmd.Println("Requirements file synced successfully.")
			} else {
				cmd.Println("No remote requirements file configured.")
			}

			if pluginsPath := environment.PluginsS3Path; pluginsPath != nil {
				cmd.Printf("Remote Plugins Path: s3://%s/%s\n", bucketName, aws.ToString(pluginsPath))
				// TODO
			} else {
				cmd.Println("No remote plugins path configured.")
			}

			cmd.Println("Syncing DAGs...")

			if dagS3Path := environment.DagS3Path; dagS3Path != nil {
				cmd.Printf("Remote DAGs Path: s3://%s/%s\n", bucketName, aws.ToString(dagS3Path))
				// TODO
			} else {
				cmd.Println("No remote DAGs path configured.")
			}

			cmd.Println(green("[SUCCESS]"), "Airflow configuration synced successfully.")

			return nil
		},
	}

	cmd.Flags().BoolVar(&awsCreds, "aws-creds", false, "Start the AWS MWAA local runner with AWS credentials")
	cmd.Flags().StringVar(&roleARN, "role-arn", "", "Specify the IAM Role ARN to use for the AWS MWAA local runner")

	return cmd
}

func newDiffCommand(globalOpts *globalOptions) *cobra.Command {
	var mwaaEnvName string

	cmd := &cobra.Command{
		Use:           "diff",
		Short:         "Compare local Airflow configuration with the remote MWAA configuration",
		Long:          "Fetch the remote Airflow configuration from the MWAA environment and compare it with the local configuration file.",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.Println(cyan("[INFO]"), "Comparing local Airflow configuration with the remote MWAA configuration...")

			cfg, err := config.NewConfig(globalOpts.profile, globalOpts.region)
			if err != nil {
				return fmt.Errorf("failed to load AWS configuration: %w", err)
			}

			ctx := context.Background()

			runner, err := local.NewRunner()
			if err != nil {
				return fmt.Errorf("failed to create AWS MWAA local runner: %w", err)
			}
			defer runner.Close()

			mwaaClient := mwaa.NewClient(cfg)

			// Get the MWAA environment name if not provided
			if mwaaEnvName == "" {
				mwaaEnvName, err = getEnvironment(ctx, mwaaClient)
				if err != nil {
					return fmt.Errorf("failed to get MWAA environment: %w", err)
				}
			}

			// Fetch the remote configuration
			environment, err := mwaaClient.GetEnvironment(ctx, mwaaEnvName)
			if err != nil {
				return fmt.Errorf("failed to get MWAA environment: %w", err)
			}

			diffs, err := runner.CompareAirflowConfigs(environment.AirflowConfigurationOptions)
			if err != nil {
				return fmt.Errorf("failed to compare configurations: %w", err)
			}

			cmd.Println(diffs.ToString())

			return nil
		},
	}

	cmd.Flags().StringVar(&mwaaEnvName, "env", "", "MWAA environment name")

	return cmd
}

func waitForWebserver(ctx context.Context, runner *local.Runner, webserverURL string, waitTime time.Duration) error {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Prefix = fmt.Sprintf("%s %s", cyan("[INFO]"), "Waiting for the Airflow webserver to be ready... ")
	s.Start()

	defer s.Stop()

	// Wait for the webserver to be ready
	err := runner.WaitForWebserverReady(ctx, fmt.Sprintf("%s/health", webserverURL), waitTime)
	if err != nil {
		return err
	}

	s.FinalMSG = fmt.Sprintf("%s %s", green("[SUCCESS]"), "AWS MWAA local runner environment started successfully!\n")

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
