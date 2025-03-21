package main

import (
	"context"
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
	serverNameFlag = &cli.StringFlag{
		Name:    "server-name",
		Aliases: []string{"sn"},
		Usage:   "The server name for the certificate. If this is set a server certificate will be requested with the name as the common name. DRIVR only signs certificates which end in .local for local API authentication",
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
		Before: combinedCheckFuncs(checkSystemComponentCode, checkAPIKey),
		Action: createCertificate,
		Flags: []cli.Flag{
			nameFlag,
			privateKeyInfileFlag,
			systemCodeFlag,
			componentCodeFlag,
			drivrAPIURLFlag,
			certificateOutfileFlag,
			issuerFlag,
			certificateDurationFlag,
			serverNameFlag,
		},
	}
}

func createCertificate(ctx *cli.Context) error {
	name := ctx.String(nameFlag.Name)
	systemCode := ctx.String(systemCodeFlag.Name)
	componentCode := ctx.String(componentCodeFlag.Name)
	duration := ctx.String(certificateDurationFlag.Name)
	issuer := ctx.String(issuerFlag.Name)
	serverName := ctx.String(serverNameFlag.Name)
	addServerUse := len(serverName) > 0

	certificateOutfile := ctx.String(certificateOutfileFlag.Name)
	if certificateOutfile == "" {
		certificateOutfile = fmt.Sprintf("%s.crt", name)
	}

	var err error

	if _, err := os.Stat(certificateOutfile); err == nil {
		return fmt.Errorf("output file %s already exists", certificateOutfile)
	}

	apiURL, err := url.Parse(getAPIUrl(ctx))
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
	drivrAPI, err := api.NewDrivrAPI(apiURL, getAPIKey())
	if err != nil {
		logrus.WithError(err).Error("Failed to create DRIVR API client")
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

	if entityUUID == nil {
		return errors.New("entity UUID is nil")
	}

	if issuerUUID == nil {
		return errors.New("issuer UUID is nil")
	}

	csr, err := cert.CreateCSR(privKey, &cert.CSRData{ServerName: serverName})
	if err != nil {
		logrus.WithError(err).Error("Failed to generate CSR")
		return err
	}

	var certificateUUID *uuid.UUID
	certificateInput := api.CreateCertificateInput{
		Name:         name,
		CSR:          string(csr),
		Duration:     duration,
		EntityUUID:   *entityUUID,
		IssuerUUID:   *issuerUUID,
		AddServerUse: addServerUse,
	}

	logrus.WithFields(certificateInput.LogFields()).Debug("Calling DRIVR API")

	certificateUUID, err = drivrAPI.CreateCertificate(ctx.Context, certificateInput)

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

	logrus.Debugf("Writing certificate to %s", certificateOutfile)
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
