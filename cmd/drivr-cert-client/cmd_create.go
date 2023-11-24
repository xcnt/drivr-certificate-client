package main

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"github.com/xcnt/drivr-certificate-client/api"
	"github.com/xcnt/drivr-certificate-client/cert"
)

const (
	PRIVATE_KEY_FILE  = "private.key"
	FETCH_DELAY_SEC   = 1
	FETCH_TIMEOUT_SEC = 120
)

var (
	keyBitsFlag = &cli.IntFlag{
		Name:    "key-bits",
		Aliases: []string{"b"},
		Usage:   "Number of bits for the key",
		Value:   2048,
	}
	privateKeyOutfileFlag = &cli.StringFlag{
		Name:    "private-key-outfile",
		Aliases: []string{"o"},
		Usage:   "Output file for the generated private key",
		Value:   PRIVATE_KEY_FILE,
	}
	privateKeyInfileFlag = &cli.StringFlag{
		Name:    "private-key-infile",
		Aliases: []string{"p"},
		Usage:   "Input file containing private key to sign certificate request",
		Value:   PRIVATE_KEY_FILE,
	}
	publicKeyOutfileFlag = &cli.StringFlag{
		Name:    "public-key-outfile",
		Aliases: []string{"u"},
		Usage:   "Output file for the generated public key",
	}
	certificateDurationFlag = &cli.StringFlag{
		Name:    "duration",
		Aliases: []string{"d"},
		Usage:   "Duration of the certificate in ISO 8601 format",
		Value:   "P365D",
	}
)

func createCommand() *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "Create a new certificate",
		Subcommands: []*cli.Command{
			keyPairCommand(),
			certificateCommand(),
		},
	}
}

func keyPairCommand() *cli.Command {
	return &cli.Command{
		Name:   "keypair",
		Usage:  "Create a new key pair",
		Action: createKeyPair,
		Flags: []cli.Flag{
			keyBitsFlag,
			privateKeyOutfileFlag,
			publicKeyOutfileFlag,
		},
	}
}

func createKeyPair(ctx *cli.Context) error {
	privateKeyOutfile := ctx.String(privateKeyOutfileFlag.Name)
	publicKeyOutfile := ctx.String(publicKeyOutfileFlag.Name)

	if privateKeyOutfile == publicKeyOutfile {
		return fmt.Errorf("Private key and public key output file cannot be the same")
	}

	return cert.GenerateRSAKeyPair(ctx.Int(keyBitsFlag.Name), privateKeyOutfile, publicKeyOutfile)
}

func certificateCommand() *cli.Command {
	return &cli.Command{
		Name:   "certificate",
		Usage:  "Create a new certificate",
		Action: createCertificate,
		Flags: []cli.Flag{
			privateKeyInfileFlag,
			drivrAPIKeyFlag,
			systemCodeFlag,
			drivrAPIURLFlag,
			certificateOutfileFlag,
			issuerFlag,
			certificateDurationFlag,
		},
	}
}

func createCertificate(ctx *cli.Context) error {
	systemCode := ctx.String(systemCodeFlag.Name)
	componentCode := ctx.String(componentCodeFlag.Name)
	duration := ctx.String(certificateDurationFlag.Name)
	issuer := ctx.String(issuerFlag.Name)

	if systemCode == "" && componentCode == "" {
		return errors.New("Either system code or component code must be specified")
	}

	if systemCode != "" && componentCode != "" {
		return errors.New("Either system code or component code must be specified, not both")
	}

	var err error

	apiURL, err := url.Parse(ctx.String(drivrAPIURLFlag.Name))
	if err != nil {
		logrus.WithError(err).Error("Failed to parse GraphQL API URL")
		return err
	}

	// load private key
	privateKeyFile := ctx.String(privateKeyInfileFlag.Name)

	if _, err := os.Stat(privateKeyFile); os.IsNotExist(err) {
		logrus.Info("Private key file does not exist - generating new key pair")

		if err := cert.GenerateRSAKeyPair(keyBitsFlag.Value, privateKeyFile, ""); err != nil {
			logrus.WithError(err).Error("Failed to generate key pair")
			return err
		}

	}

	logrus.WithField("filename", privateKeyFile).Debug("Loading private key")
	privKey, err := cert.LoadPrivateKey(privateKeyFile)
	if err != nil {
		logrus.WithError(err).Error("Failed to load private key")
		return err
	}

	logrus.Debug("Initializing DRIVR API Client")
	drivrAPI, err := api.NewDrivrAPI(apiURL, ctx.String(drivrAPIKeyFlag.Name))
	if err != nil {
		logrus.WithError(err).Error("Failed to create DRIVR API client")
		return err
	}

	domainUUID, err := drivrAPI.FetchDomainUUID(ctx.Context)
	if err != nil {
		logrus.WithError(err).Debug("Failed to fetch current domain")
		return err
	}

	var entityUUID *uuid.UUID
	if systemCode != "" {
		entityUUID, err = drivrAPI.FetchSystemUUID(ctx.Context, systemCode)
		if err != nil {
			logrus.WithField("code", systemCode).WithError(err).Debug("Failed to fetch system")
			return err
		}
	} else {
		entityUUID, err = drivrAPI.FetchComponentUUID(ctx.Context, componentCode)
		if err != nil {
			logrus.WithField("code", componentCode).WithError(err).Debug("Failed to fetch component")
			return err
		}
	}

	issuerUUID, err := drivrAPI.FetchIssuerUUID(ctx.Context, issuer)
	if err != nil {
		logrus.WithField("issuer", issuer).WithError(err).Debug("Failed to fetch issuer")
		return err
	}

	var code string
	if systemCode != "" {
		code = systemCode
	} else {
		code = componentCode
	}
	cn := fmt.Sprintf("%s@%s", code, domainUUID.String())
	logrus.WithField("common_name", cn).Debug("Generating CSR")
	csr, err := cert.CreateCSR(privKey, cn)
	if err != nil {
		logrus.WithError(err).Error("Failed to generate CSR")
		return err
	}

	base64CSR := base64.StdEncoding.EncodeToString(csr)

	logrus.WithFields(logrus.Fields{
		"issuerUuid": issuerUUID,
		"cn":         cn,
		"csr":        base64CSR,
		"duration":   duration,
		"entityUuid": entityUUID,
	}).Debug("Calling DRIVR API")

	var certificateUUID *uuid.UUID
	certificateUUID, err = drivrAPI.CreateCertificate(ctx.Context, issuerUUID, entityUUID, cn, base64CSR, duration)

	if err != nil {
		logrus.WithError(err).Error("Failed to request certificate creation")
		return err
	}
	logrus.WithField("certificate_uuid", certificateUUID.String()).Debug("Certificate requested")

	certificate, _, err := waitForCertificate(ctx.Context, drivrAPI, certificateUUID)
	if err != nil {
		logrus.WithError(err).Error("Failed to fetch certificate")
		return err
	}

	certificateOutfile := ctx.String(certificateOutfileFlag.Name)
	if certificateOutfile == "" {
		certificateOutfile = fmt.Sprintf("%s.crt", code)
	}

	return cert.WriteToPEMFile(cert.Certificate, certificate, certificateOutfile)
}

func waitForCertificate(ctx context.Context, api *api.DrivrAPI, certificateUUID *uuid.UUID) (certificate []byte, name string, err error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, FETCH_TIMEOUT_SEC*time.Second)
	defer cancel()
	certReady := make(chan interface{}, 1)

	go func() {
		for {
			certificate, name, err = api.FetchCertificate(timeoutCtx, certificateUUID)
			if err != nil {
				logrus.WithError(err).Debug("Failed to fetch certificate")
				time.Sleep(FETCH_DELAY_SEC * time.Second)
				continue
			}

			certReady <- nil
			return
		}
	}()

	select {
	case <-certReady:
	case <-timeoutCtx.Done():
		err = errors.New("timed out waiting for certificate")
	}

	return certificate, name, err
}
