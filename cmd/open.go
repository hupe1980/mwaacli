package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/hupe1980/mwaacli/pkg/config"
	"github.com/hupe1980/mwaacli/pkg/mwaa"
	"github.com/spf13/cobra"
)

// newOpenCommand creates a new Cobra command for opening the MWAA web application in a browser.
func newOpenCommand(globalOpts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "open [environment]",
		Short: "Open the MWAA webapp in a browser",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.NewConfig(globalOpts.profile, globalOpts.region)
			if err != nil {
				return err
			}

			client, err := mwaa.NewClient(cfg)
			if err != nil {
				return err
			}

			mwaaEnvName := args[0]
			ctx := context.Background()

			if mwaaEnvName == "" {
				environments, err := client.ListEnvironments(ctx)
				if err != nil {
					return fmt.Errorf("failed to list environments: %w", err)
				}

				if len(environments) != 1 {
					return fmt.Errorf("environment name is required")
				}

				mwaaEnvName = environments[0]
			}

			webLoginTokenOutput, err := client.CreateWebLoginToken(ctx, mwaaEnvName)
			if err != nil {
				return err
			}

			webserverURL := fmt.Sprintf("https://%s/aws_mwaa/aws-console-sso?login=true#%s",
				aws.ToString(webLoginTokenOutput.WebServerHostname), aws.ToString(webLoginTokenOutput.WebToken))

			cmd.Printf("Opening webserver at: %s\n", webserverURL)

			return openBrowser(webserverURL)
		},
	}

	cmd.SetOut(os.Stdout)
	cmd.SetErr(os.Stderr)

	return cmd
}

// openBrowser attempts to open the given URL in the default web browser based on the operating system.
func openBrowser(url string) error {
	switch runtime.GOOS {
	case "linux":
		return exec.Command("xdg-open", url).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return fmt.Errorf("unsupported platform")
	}
}
