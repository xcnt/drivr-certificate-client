package main

import "github.com/urfave/cli/v2"

var APIKeyFlag = &cli.StringFlag{
	Name:    "api-key",
	Usage:   "Static API key for authenticating requests.",
	Value:   "",
	EnvVars: []string{"DRIVR_API_KEY"},
}
