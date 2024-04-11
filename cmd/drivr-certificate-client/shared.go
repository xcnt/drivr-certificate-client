package main

import (
	"errors"
	"fmt"
	"os"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"golang.org/x/term"
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

var apikey string

func getAPIKey() string {
	// read the API key from the environment
	if apikey != "" {
		return apikey
	}

	if envAPIKey := os.Getenv(drivrAPIKeyEnv); envAPIKey != "" {
		apikey = envAPIKey
		return apikey
	}

	fmt.Println("Enter the API key:")
	keyBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		logrus.WithError(err).Error("Failed to read API key")
		return ""
	}
	apikey = string(keyBytes)
	return apikey
}

func combinedCheckFuncs(checks ...func(*cli.Context) error) func(*cli.Context) error {
	return func(ctx *cli.Context) error {
		for _, check := range checks {
			if err := check(ctx); err != nil {
				return err
			}
		}
		return nil
	}
}

func checkAPIKey(ctx *cli.Context) error {
	if getAPIKey() == "" {
		return errors.New("API key is required")
	}
	return nil
}

func checkSystemComponentCode(ctx *cli.Context) error {
	systemCode := ctx.String(systemCodeFlag.Name)
	componentCode := ctx.String(componentCodeFlag.Name)

	if systemCode == "" && componentCode == "" {
		return errors.New("Either system code or component code must be specified")
	}

	if systemCode != "" && componentCode != "" {
		return errors.New("Either system code or component code must be specified, not both")
	}

	return nil
}
