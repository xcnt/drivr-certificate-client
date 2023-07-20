package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"github.com/xcnt/drivr-certificate-client/api"
	"github.com/xcnt/drivr-certificate-client/cert"
)

const (
	PRIVATE_KEY_FILE = "private.key"
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
	return cert.GenerateRSAKeyPair(ctx.Int(keyBitsFlag.Name), ctx.String(privKeyOutfileFlag.Name), ctx.String(pubKeyOutfileFlag.Name))
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

	logrus.WithField("certificate_uuid", mutation.CreateCertificate.UUID).Debug("Certificate created")
	fmt.Printf("Certificate UUID: %s\n", mutation.CreateCertificate.UUID)

	return nil
}
