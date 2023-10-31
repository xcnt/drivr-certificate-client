package main

import "github.com/urfave/cli/v2"

var (
	certificateOutfileFlag = &cli.StringFlag{
		Name:  "cert-outfile",
		Usage: "Certificate output file",
	}
	clientNameFlag = &cli.StringFlag{
		Name:     "client-name",
		Aliases:  []string{"n"},
		Usage:    "Name of the client to create the certificate for",
		Required: true,
	}
	drivrAPIURLFlag = &cli.StringFlag{
		Name:     "drivr-api",
		Usage:    "DRIVR API URL",
		Required: true,
		EnvVars:  []string{"DRIVR_API_URL"},
	}
	drivrAPIKeyFlag = &cli.StringFlag{
		Name:    "api-key",
		Usage:   "Static API key for authenticating requests.",
		Value:   "",
		EnvVars: []string{"DRIVR_API_KEY"},
	}
)
