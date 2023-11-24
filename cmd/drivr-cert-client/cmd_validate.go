package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/url"
	"os"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var (
	certificateInfileFlag = &cli.StringFlag{
		Name:    "certificate-infile",
		Aliases: []string{"c"},
		Usage:   "Certificate file to validate",
	}
	mqttBrokerFlag = &cli.StringFlag{
		Name:    "mqtt-broker",
		Aliases: []string{"b"},
		Usage:   "MQTT broker to connect to",
	}
	mqttBrokerPortFlag = &cli.IntFlag{
		Name:    "mqtt-broker-port",
		Aliases: []string{"p"},
		Usage:   "MQTT broker port to connect to",
		Value:   8883,
	}
	caCertInfileFlag = &cli.StringFlag{
		Name:    "ca-cert",
		Aliases: []string{"a"},
		Usage:   "CA certificate file",
	}
	issuerFlag = &cli.StringFlag{
		Name:    "issuer",
		Aliases: []string{"i"},
		Usage:   "Issuer of the certificate",
	}
)

func validateCommand() *cli.Command {
	return &cli.Command{
		Name:   "validate",
		Usage:  "Validate a certificate",
		Action: validateCertificate,
		Flags: []cli.Flag{
			drivrAPIKeyFlag,
			drivrAPIURLFlag,
			privateKeyInfileFlag,
			certificateInfileFlag,
			mqttBrokerFlag,
			mqttBrokerPortFlag,
		},
	}
}

func loadCAFromFile(filename string) ([]byte, error) {
	ca, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return ca, nil
}

func newTLSConfig(caCert []byte, clientCert, clientPrivateKey string) (*tls.Config, error) {
	certpool := x509.NewCertPool()
	certpool.AppendCertsFromPEM(caCert)

	clientKeyPair, err := tls.LoadX509KeyPair(clientCert, clientPrivateKey)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		RootCAs:            certpool,
		ClientAuth:         tls.NoClientCert,
		ClientCAs:          nil,
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{clientKeyPair},
	}, nil
}

func validateCertificate(ctx *cli.Context) error {
	cacertfile := ctx.String(caCertInfileFlag.Name)

	var cacert []byte

	if cacertfile == "" {
		issuer := ctx.String(issuerFlag.Name)
		apiURL, err := url.Parse(ctx.String(drivrAPIURLFlag.Name))
		if err != nil {
			logrus.WithError(err).Error("Failed to parse GraphQL API URL")
			return err
		}
		if issuer == "" || apiURL == nil {
			return fmt.Errorf("either %s or %s and %s must be specified", caCertInfileFlag.Name, issuerFlag.Name, drivrAPIURLFlag.Name)
		}
		cacert, err = getCaCert(ctx.Context, issuer, apiURL, ctx.String(drivrAPIKeyFlag.Name))
		if err != nil {
			return err
		}
	} else {
		var err error
		cacert, err = loadCAFromFile(cacertfile)
		if err != nil {
			return err
		}
	}

	privKeyFile := ctx.String(privateKeyInfileFlag.Name)
	certificateFile := ctx.String(certificateInfileFlag.Name)
	mqttBroker := ctx.String(mqttBrokerFlag.Name)
	mqttBrokerPort := ctx.Int(mqttBrokerPortFlag.Name)

	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", mqttBroker, mqttBrokerPort))

	tlsConfig, err := newTLSConfig(cacert, certificateFile, privKeyFile)
	if err != nil {
		return err
	}
	opts.SetTLSConfig(tlsConfig)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}
