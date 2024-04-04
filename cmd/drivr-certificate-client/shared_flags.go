package main

import "github.com/urfave/cli/v2"

var (
	nameFlag = &cli.StringFlag{
		Name:     "name",
		Aliases:  []string{"n"},
		Usage:    "Name of the certificate",
		Required: true,
	}
	certificateOutfileFlag = &cli.StringFlag{
		Name:  "cert-outfile",
		Usage: "Certificate output file",
	}
	systemCodeFlag = &cli.StringFlag{
		Name:    "system-code",
		Aliases: []string{"s"},
		Usage:   "Code of the System to create the certificate for",
	}
	componentCodeFlag = &cli.StringFlag{
		Name:    "component-code",
		Aliases: []string{"c"},
		Usage:   "Code of the Component to create the certificate for",
	}
	drivrAPIURLFlag = &cli.StringFlag{
		Name:     "drivr-api",
		Usage:    "DRIVR API URL",
		Required: true,
		EnvVars:  []string{"DRIVR_API_URL"},
	}
	drivrAPIKeyFlag = &cli.StringFlag{
		Name:     "api-key",
		Usage:    "Static API key for authenticating requests.",
		EnvVars:  []string{"DRIVR_API_KEY"},
		Required: true,
		Action: func(c *cli.Context, value string) error {
			if c.String("api-key") == "" {
				return cli.Exit("api-key cannot be empty", 1)
			}
			return nil
		},
	}
	issuerFlag = &cli.StringFlag{
		Name:    "issuer",
		Value:   "default",
		Aliases: []string{"i"},
		Usage:   "Issuer of the certificate",
	}
)
