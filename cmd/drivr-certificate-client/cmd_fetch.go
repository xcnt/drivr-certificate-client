package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/google/uuid"
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

func fetchCommand() *cli.Command {
	return &cli.Command{
		Name:   "fetch",
		Usage:  "Fetch certificates from DRIVR",
		Before: checkAPIKey,
		Subcommands: []*cli.Command{
			fetchCertificateCommand(),
			fetchCertificateAutorityCommand(),
		},
	}
}

func fetchCertificateAutorityCommand() *cli.Command {
	return &cli.Command{
		Name:   "ca",
		Usage:  "Fetch a certificate authority",
		Action: fetchCertificateAutority,
		Flags: []cli.Flag{
			issuerFlag,
			drivrAPIURLFlag,
		},
	}
}

func fetchCertificateCommand() *cli.Command {
	return &cli.Command{
		Name:   "certificate",
		Usage:  "Fetch a certificate",
		Action: fetchCertificate,
		Flags: []cli.Flag{
			certificateUUIDFlag,
			drivrAPIURLFlag,
		},
	}
}

func getCaCert(ctx context.Context, issuer string, apiURL *url.URL, apiKey string) ([]byte, error) {
	api, err := api.NewDrivrAPI(apiURL, apiKey)
	if err != nil {
		logrus.WithError(err).Error("Failed to initialize DRIVR API Client")
		return nil, err
	}
	ca, err := api.FetchCertificateAuthority(ctx, issuer)
	if err != nil {
		logrus.WithError(err).Error("Failed to fetch certificate authority")
		return nil, err
	}

	return ca, nil
}

func fetchCertificateAutority(ctx *cli.Context) error {
	apiURL, err := url.Parse(ctx.String(drivrAPIURLFlag.Name))
	if err != nil {
		logrus.WithError(err).Error("Failed to parse GraphQL API URL")
		return err
	}
	ca, err := getCaCert(ctx.Context, ctx.String(issuerFlag.Name), apiURL, getAPIKey())
	if err != nil {
		logrus.WithError(err).Error("Failed to fetch CA certificate")
		return err
	}
	err = cert.WriteToPEMFile(cert.Certificate, ca, "ca.crt")
	if err != nil {
		logrus.WithError(err).Error("Failed writing CA certificate to file")
		return err
	}
	return nil
}

func fetchCertificate(ctx *cli.Context) error {
	apiURL, err := url.Parse(ctx.String(drivrAPIURLFlag.Name))
	if err != nil {
		logrus.WithError(err).Error("Failed to parse GraphQL API URL")
		return err
	}

	certificateUUIDstr := ctx.String(certificateUUIDFlag.Name)
	certificateUUID, err := uuid.Parse(certificateUUIDstr)
	if err != nil {
		logrus.WithField("certificate_uuid", certificateUUID).Error("Invalid certificate UUID")
		return errors.New("invalid certificate UUID")
	}

	drivrAPI, err := api.NewDrivrAPI(apiURL, getAPIKey())
	if err != nil {
		logrus.WithError(err).Error("Failed to initialize DRIVR API Client")
		return err
	}

	certificate, name, err := drivrAPI.FetchCertificate(ctx.Context, &certificateUUID)
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
