package main

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"regexp"

	"github.com/shurcooL/graphql"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"github.com/xcnt/drivr-certificate-client/api"
	"github.com/xcnt/drivr-certificate-client/cert"
)

var (
	certificateUUIDFlag = &cli.StringFlag{
		Name:     "uuid",
		Usage:    "Certificate UUID",
		Required: true,
	}
)

var uuidRegexp = regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-4[a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}$`)

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
	if !uuidRegexp.MatchString(certificateUUID) {
		logrus.WithField("certificate_uuid", certificateUUID).Error("Invalid certificate UUID")
		return errors.New("invalid certificate UUID")
	}

	client, err := api.NewClient(apiURL.String(), ctx.String(APIKeyFlag.Name))
	if err != nil {
		logrus.WithError(err).Error("Failed to initialize GraphQL client")
		return err
	}
	certificate, name, err := fetchCertificate(client, certificateUUID)
	if err != nil {
		logrus.WithError(err).Error("Failed to fetch certificate")
		return err
	}

	certificateOutfile := ctx.String(certificateOutfileFlag.Name)
	if certificateOutfile == "" {
		certificateOutfile = fmt.Sprintf("%s.crt", name)
	}
	if err = cert.WriteToPEMFile(cert.Certificate, certificate, certificateOutfile); err != nil {
		logrus.WithField("filename", certificateOutfile).WithError(err).Error("Failed to write certificate to file")
		return err
	}

	return nil
}

func fetchCertificate(client *graphql.Client, certificateUUID string) (certificate []byte, name string, err error) {
	var query api.FetchCertificateQuery

	err = client.Query(context.TODO(), &query, map[string]interface{}{
		"uuid": api.UUID(certificateUUID),
	})
	if err != nil {
		logrus.WithField("certificate_uuid", certificateUUID).WithError(err).Error("Failed to query certificate")
		return nil, "", err
	}

	if query.FetchCertificate.Certificate == "" {
		logrus.WithField("certificate_uuid", certificateUUID).Error("Certificate not yet signed")
		return nil, "", err
	}

	name = string(query.FetchCertificate.Name)

	certificate, err = base64.RawStdEncoding.DecodeString(string(query.FetchCertificate.Certificate))
	if err != nil {
		logrus.WithError(err).Error("Failed to decode certificate")
		return nil, "", err
	}
	return certificate, name, nil

}
