package main

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"regexp"

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
	certificateUUIDFlag = &cli.StringFlag{
		Name:     "uuid",
		Usage:    "Certificate UUID",
		Required: true,
	}
)

var uUIDRegexp = regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-4[a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}$`)

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
		Action: fetchCertificateAction,
		Flags: []cli.Flag{
			certificateUUIDFlag,
			APIKeyFlag,
			graphqlAPIFlag,
		},
	}
}

func fetchCertificateAction(ctx *cli.Context) error {
	apiURL, err := url.Parse(ctx.String(graphqlAPIFlag.Name))
	if err != nil {
		logrus.WithError(err).Error("Failed to parse GraphQL API URL")
		return err
	}

	certificateUUID := ctx.String(certificateUUIDFlag.Name)
	if !uUIDRegexp.MatchString(certificateUUID) {
		logrus.WithField("certificate_uuid", certificateUUID).Error("Invalid certificate UUID")
		return errors.New("invalid certificate UUID")
	}

	return fetchCertificate(apiURL.String(), ctx.String(APIKeyFlag.Name), certificateUUID, ctx.String(certificateOutfileFlag.Name))
}

func fetchCertificate(apiURL, apiKey, certificateUUID, certificateOutfile string) error {
	var query api.FetchCertificateQuery

	client, err := api.NewClient(apiURL, apiKey)
	if err != nil {
		logrus.WithError(err).Error("Failed to initialize GraphQL client")
		return err
	}
	err = client.Query(context.TODO(), &query, map[string]interface{}{
		"uuid": certificateUUID,
	})
	if err != nil {
		logrus.WithField("certificate_uuid", certificateUUID).WithError(err).Error("Failed to query certificate")
		return err
	}

	if query.FetchCertificate.Certificate == "" {
		logrus.WithField("certificate_uuid", certificateUUID).Error("Certificate not yet signed")
		return err
	}

	certificate, err := base64.RawStdEncoding.DecodeString(string(query.FetchCertificate.Certificate))
	if err != nil {
		logrus.WithError(err).Error("Failed to decode certificate")
		return err
	}

	if certificateOutfile == "" {
		certificateOutfile = fmt.Sprintf("%s.crt", string(query.FetchCertificate.Name))
	}
	err = cert.WriteToPEMFile(cert.Certificate, certificate, certificateOutfile)
	if err != nil {
		logrus.WithField("filename", certificateOutfile).WithError(err).Error("Failed to write certificate to file")
		return err
	}

	return nil
}
