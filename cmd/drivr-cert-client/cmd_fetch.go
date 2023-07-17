package main

import (
	"context"
	"encoding/base64"
	"net/url"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"github.com/xcnt/drivr-certificate-client/api"
	"github.com/xcnt/drivr-certificate-client/cert"
)

var (
	certificateOutfileFlag = &cli.StringFlag{
		Name:  "cert-outfile",
		Usage: "Certificate output file",
	}
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
			graphqlAPIFlag,
		},
	}
}

func fetchCertificate(ctx *cli.Context) error {
	apiURL, err := url.Parse(ctx.String(graphqlAPIFlag.Name))
	if err != nil {
		logrus.WithError(err).Error("Failed to parse GraphQL API URL")
		return err
	}

	client, err := api.NewClient(apiURL.String(), ctx.String(APIKeyFlag.Name))
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

	if query.FetchCertificate.Certificate == "" {
		logrus.WithField("client_name", clientName).Error("Certificate not yet signed")
		return err
	}

	certificate, err := base64.RawStdEncoding.DecodeString(string(query.FetchCertificate.Certificate))
	if err != nil {
		logrus.WithError(err).Error("Failed to decode certificate")
		return err
	}

	certOutfile := ctx.String(certificateOutfileFlag.Name)
	if certOutfile == "" {
		certOutfile = clientName + ".crt"
	}
	err = cert.WriteToPEMFile(cert.Certificate, certificate, certOutfile)
	if err != nil {
		logrus.WithField("filename", certOutfile).WithError(err).Error("Failed to write certificate to file")
		return err
	}

	return nil
}
