package main

import "github.com/urfave/cli/v2"

var (
	APIKeyFlag = &cli.StringFlag{
		Name:    "api-key",
		Usage:   "Static API key for authenticating requests.",
		Value:   "",
		EnvVars: []string{"DRIVR_API_KEY"},
	}
	clientNameFlag = &cli.StringFlag{
		Name:     "client-name",
		Aliases:  []string{"n"},
		Usage:    "Name of the client to create the certificate for",
		Required: true,
	}
	graphqlAPIFlag = &cli.StringFlag{
		Name:     "graphql-api",
		Usage:    "URL of the GraphQL API",
		Required: true,
		EnvVars:  []string{"DRIVR_GRAPHQL_API"},
	}
)
