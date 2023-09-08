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
		Aliases: []string{"p"},
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
	entityUuidFlag = &cli.StringFlag{
		Name:     "entity-uuid",
		Aliases:  []string{"e"},
		Usage:    "UUID of the entity to create the certificate for",
		Required: true,
	}
	requiredIssuerFlag = &cli.StringFlag{
		Name:     "issuer",
		Aliases:  []string{"i"},
		Usage:    "Issuer of the certificate",
		Required: true,
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
			certificateOutfileFlag,
			requiredIssuerFlag,
			entityUuidFlag,
		},
	}
}

func fetchIssuerUUID(client *graphql.Client, issuer string) (uuid string, err error) {
	var query api.FetchIssuerUUIDQuery

	err = client.Query(context.TODO(), &query, map[string]interface{}{
		"name": graphql.String(issuer),
	})
	if err != nil {
		logrus.WithField("issuer", issuer).WithError(err).Error("Failed to query issuer")
		return "", err
	}

	if len(query.FetchIssuer.Items) != 1 {
		logrus.WithField("issuer", issuer).Error("Issuer not found")
		return "", err
	}

	return string(query.FetchIssuer.Items[0].Uuid), nil

}

func createCertificate(ctx *cli.Context) error {
	name := ctx.String(clientNameFlag.Name)
	duration := ctx.Int(certificateDurationFlag.Name)
	issuer := ctx.String(issuerFlag.Name)
	entityUUID := ctx.String(entityUuidFlag.Name)

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

	issuerUUID, err := fetchIssuerUUID(client, issuer)
	if err != nil {
		logrus.WithField("issuer", issuer).WithError(err).Debug("Failed to fetch issuer")
		return err
	}

	vars := map[string]interface{}{
		"issuerUuid": graphql.String(issuerUUID),
		"name":       graphql.String(name),
		"csr":        graphql.String(base64CSR),
		"duration":   graphql.Int(duration),
		"entityUuid": graphql.String(entityUUID),
	}

	logrus.WithFields(logrus.Fields{
		"issuerUuid": issuerUUID,
		"name":       name,
		"csr":        base64CSR,
		"duration":   duration,
		"entityUuid": entityUUID,
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
