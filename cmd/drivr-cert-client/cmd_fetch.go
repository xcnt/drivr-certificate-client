package main

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"github.com/xcnt/drivr-certificate-client/api"
)

func fetchCommand() *cli.Command {
	return &cli.Command{
		Name:  "fetch",
		Usage: "Fetch certificates from DRIVR",
		Subcommands: []*cli.Command{
			fetchCertificateCommand(),
		},
	}
}

func fetchCertificateCommand() *cli.Command {
	return &cli.Command{
		Name:   "certificate",
		Usage:  "Fetch a certificate",
		Action: fetchCertificate,
		Flags: []cli.Flag{
			clientNameFlag,
			APIKeyFlag,
		},
	}
}

func fetchCertificate(ctx *cli.Context) error {
	client, err := api.NewClient(ctx.String(APIKeyFlag.Name))
	if err != nil {
		logrus.WithError(err).Error("Failed to initialize GraphQL client")
		return err
	}

	var query api.FetchCertificateQuery

	clientName := ctx.String(clientNameFlag.Name)
	err = client.Query(context.TODO(), &query, map[string]interface{}{
		"name": clientName,
	})
	if err != nil {
		logrus.WithField("client_name", clientName).WithError(err).Error("Failed to query certificate")
		return err
	}

	return nil
}
