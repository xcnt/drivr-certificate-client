package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

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
	issuerFlag = &cli.StringFlag{
		Name:    "issuer",
		Value:   "default",
		Aliases: []string{"i"},
		Usage:   "Issuer of the certificate",
	}
)

const drivrAPIKeyEnv = "DRIVR_API_KEY"

func getAPIKey() string {
	// read the API key from the environment
	return os.Getenv(drivrAPIKeyEnv)
}

func checkAPIKey(ctx *cli.Context) error {
	if getAPIKey() == "" {
		return fmt.Errorf("%s environment variable not set", drivrAPIKeyEnv)
	}
	return nil
}
