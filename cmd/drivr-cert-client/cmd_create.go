package main

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/shurcooL/graphql"
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
	privKeyOutfileFlag = &cli.StringFlag{
		Name:    "privkey-outfile",
		Aliases: []string{"o"},
		Usage:   "Output file for the generated private key",
		Value:   PRIVATE_KEY_FILE,
	}
	privKeyInfileFlag = &cli.StringFlag{
		Name:    "privkey-infile",
		Aliases: []string{"i"},
		Usage:   "Input file containing private key to sign certificate request",
		Value:   PRIVATE_KEY_FILE,
	}
	pubKeyOutfileFlag = &cli.StringFlag{
		Name:    "pubkey-outfile",
		Aliases: []string{"u"},
		Usage:   "Output file for the generated public key",
	}
	certificateDurationFlag = &cli.IntFlag{
		Name:    "duration",
		Aliases: []string{"d"},
		Usage:   "Duration of the certificate in days",
		Value:   365,
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
			privKeyOutfileFlag,
			pubKeyOutfileFlag,
		},
	}
}

func createKeyPair(ctx *cli.Context) error {
	privateKeyOutfile := ctx.String(privKeyOutfileFlag.Name)
	publicKeyOutfile := ctx.String(pubKeyOutfileFlag.Name)

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
			privKeyInfileFlag,
			APIKeyFlag,
			clientNameFlag,
			graphqlAPIFlag,
		},
	}
}

func createCertificate(ctx *cli.Context) error {
	name := ctx.String(clientNameFlag.Name)
	duration := ctx.Int(certificateDurationFlag.Name)

	apiURL, err := url.Parse(ctx.String(graphqlAPIFlag.Name))
	if err != nil {
		logrus.WithError(err).Error("Failed to parse GraphQL API URL")
		return err
	}

	// load private key
	privateKeyFile := ctx.String(privKeyInfileFlag.Name)

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

	logrus.WithField("common_name", name).Debug("Generating CSR")
	csr, err := cert.CreateCSR(privKey, name)
	if err != nil {
		logrus.WithError(err).Error("Failed to generate CSR")
		return err
	}

	base64CSR := base64.StdEncoding.EncodeToString(csr)

	logrus.Debug("Initializing GraphQL client")
	client, err := api.NewClient(apiURL.String(), ctx.String(APIKeyFlag.Name))
	if err != nil {
		logrus.WithError(err).Error("Failed to create GraphQL client")
		return err
	}

	vars := map[string]interface{}{
		"name":     name,
		"csr":      base64CSR,
		"duration": duration,
	}

	logrus.WithFields(logrus.Fields{
		"name":     name,
		"csr":      base64CSR,
		"duration": duration,
	}).Debug("Calling GraphQL API")

	var mutation api.CreateCertificateMutation

	err = client.Mutate(context.TODO(), mutation, vars)
	if err != nil {
		logrus.WithError(err).Error("Failed to request certificate creation")
		return err
	}

	certificateUUID := mutation.CreateCertificate.UUID

	logrus.WithField("certificate_uuid", string(certificateUUID)).Debug("Certificate requested")

	certificate, name, err := waitForCertificate(client, string(certificateUUID))
	if err != nil {
		logrus.WithError(err).Error("Failed to fetch certificate")
		return err
	}

	certificateOutfile := ctx.String(certificateOutfileFlag.Name)
	if certificateOutfile == "" {
		certificateOutfile = fmt.Sprintf("%s.crt", name)
	}

	return cert.WriteToPEMFile(cert.Certificate, certificate, certificateOutfile)
}

func waitForCertificate(client *graphql.Client, certificateUUID string) (certificate []byte, name string, err error) {
	certReady := make(chan interface{}, 1)

	go func() {
		for {
			certificate, name, err = fetchCertificate(client, certificateUUID)
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
	case <-time.After(FETCH_TIMEOUT_SEC * time.Second):
		err = errors.New("timed out waiting for certificate")
	}

	return certificate, name, nil
}
