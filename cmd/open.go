package cmd

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/hupe1980/mwaacli/pkg/config"
	"github.com/hupe1980/mwaacli/pkg/mwaa"
	"github.com/hupe1980/mwaacli/pkg/util"
	"github.com/spf13/cobra"
)

// newOpenCommand creates a new Cobra command for opening the MWAA web application in a browser.
func newOpenCommand(globalOpts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "open [environment]",
		Short:         "Open the MWAA webapp in a browser",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.NewConfig(globalOpts.profile, globalOpts.region)
			if err != nil {
				return err
			}

			client := mwaa.NewClient(cfg)

			ctx := context.Background()

			var mwaaEnvName string
			if len(args) > 0 {
				mwaaEnvName = args[0]
			}

			if mwaaEnvName == "" {
				mwaaEnvName, err = getEnvironment(ctx, client)
				if err != nil {
					return err
				}
			}

			webLoginTokenOutput, err := client.CreateWebLoginToken(ctx, mwaaEnvName)
			if err != nil {
				return err
			}

			webserverURL := fmt.Sprintf("https://%s/aws_mwaa/aws-console-sso?login=true#%s",
				aws.ToString(webLoginTokenOutput.WebServerHostname), aws.ToString(webLoginTokenOutput.WebToken))

			cmd.Printf(cyan("[INFO]"), "Opening webserver at: %s\n", webserverURL)

			return util.OpenBrowser(webserverURL)
		},
	}

	return cmd
}
