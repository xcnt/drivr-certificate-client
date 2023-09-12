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
	entityUuidFlag = &cli.StringFlag{
		Name:    "entity-uuid",
		Aliases: []string{"e"},
		Usage:   "UUID of the entity to create the certificate for",
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
			APIKeyFlag,
			clientNameFlag,
			graphqlAPIFlag,
			certificateOutfileFlag,
			requiredIssuerFlag,
			entityUuidFlag,
			certificateDurationFlag,
		},
	}
}

func fetchIssuerUUID(ctx context.Context, client *graphql.Client, issuer string) (*uuid.UUID, error) {
	var query api.FetchIssuerUUIDQuery

	err := client.Query(ctx, &query, map[string]interface{}{
		"name": graphql.String(issuer),
	})
	if err != nil {
		logrus.WithField("issuer", issuer).WithError(err).Error("Failed to query issuer")
		return nil, err
	}

	if len(query.FetchIssuer.Items) != 1 {
		logrus.WithField("issuer", issuer).Error("Issuer not found")
		return nil, errors.New("Issuer not found")
	}

	uuidStr := string(query.FetchIssuer.Items[0].Uuid)
	uuid, err := uuid.Parse(uuidStr)
	if err != nil {
		logrus.WithField("issuer_uuid", uuidStr).WithError(err).Error("Failed to parse issuer UUID")
		return nil, err
	}

	return &uuid, nil
}

func createCertificate(ctx *cli.Context) error {
	name := ctx.String(clientNameFlag.Name)
	duration := ctx.String(certificateDurationFlag.Name)
	issuer := ctx.String(issuerFlag.Name)
	entityUUIDstr := ctx.String(entityUuidFlag.Name)

	var err error
	var entityUUID uuid.UUID

	if entityUUIDstr != "" {
		entityUUID, err = uuid.Parse(entityUUIDstr)
		if err != nil {
			logrus.WithError(err).Error("Failed to parse entity UUID")
			return err
		}
	}

	apiURL, err := url.Parse(ctx.String(graphqlAPIFlag.Name))
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

	issuerUUID, err := fetchIssuerUUID(ctx.Context, client, issuer)
	if err != nil {
		logrus.WithField("issuer", issuer).WithError(err).Debug("Failed to fetch issuer")
		return err
	}

	// help the graphql client to determine the correct type
	gqlEntityUUID := api.UUID(entityUUID.String())
	gqlIssuerUUID := api.UUID(issuerUUID.String())

	gqlDuration := api.Timespan(duration)

	vars := map[string]interface{}{
		"issuerUuid": gqlIssuerUUID,
		"name":       graphql.String(name),
		"csr":        graphql.String(base64CSR),
		"duration":   gqlDuration,
	}

	if entityUUID == uuid.Nil {
		vars["entityUuid"] = map[string]interface{}{
			"entityUuid": gqlEntityUUID,
		}
	}

	logrus.WithFields(logrus.Fields{
		"issuerUuid": issuerUUID,
		"name":       name,
		"csr":        base64CSR,
		"duration":   duration,
		"entityUuid": entityUUID,
	}).Debug("Calling GraphQL API")

	var mutation api.CreateCertificateMutation

	err = client.Mutate(ctx.Context, &mutation, vars)
	if err != nil {
		logrus.WithError(err).Error("Failed to request certificate creation")
		return err
	}

	certificateUUID := mutation.CreateCertificate.Uuid

	logrus.WithField("certificate_uuid", string(certificateUUID)).Debug("Certificate requested")

	certificate, name, err := waitForCertificate(ctx.Context, client, string(certificateUUID))
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

func waitForCertificate(ctx context.Context, client *graphql.Client, certificateUUID string) (certificate []byte, name string, err error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, FETCH_TIMEOUT_SEC*time.Second)
	defer cancel()
	certReady := make(chan interface{}, 1)

	go func() {
		for {
			certificate, name, err = fetchCertificate(timeoutCtx, client, certificateUUID)
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
