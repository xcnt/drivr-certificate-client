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
		Name:  "mqtt-broker",
		Usage: "MQTT broker to connect to",
	}
	mqttBrokerPortFlag = &cli.IntFlag{
		Name:  "mqtt-broker-port",
		Usage: "MQTT broker port to connect to",
		Value: 8883,
	}
	caCertInfileFlag = &cli.StringFlag{
		Name:    "ca-cert",
		Aliases: []string{"a"},
		Usage:   "CA certificate file",
		Value:   "ca.crt",
	}
	topicFlag = &cli.StringFlag{
		Name:    "topic",
		Aliases: []string{"t"},
		Usage:   "Optional topic to subscribe to",
		Value:   "",
	}
)

func validateCommand() *cli.Command {
	return &cli.Command{
		Name:   "validate",
		Usage:  "Validate a certificate",
		Before: checkAPIKey,
		Action: validateCertificate,
		Flags: []cli.Flag{
			drivrAPIURLFlag,
			privateKeyInfileFlag,
			certificateInfileFlag,
			mqttBrokerFlag,
			mqttBrokerPortFlag,
			issuerFlag,
			topicFlag,
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
	privKeyFile := ctx.String(privateKeyInfileFlag.Name)
	certificateFile := ctx.String(certificateInfileFlag.Name)
	mqttBroker := ctx.String(mqttBrokerFlag.Name)
	mqttBrokerPort := ctx.Int(mqttBrokerPortFlag.Name)

	if certificateFile == "" {
		return fmt.Errorf("certificate file must be specified")
	}

	if privKeyFile == "" {
		return fmt.Errorf("private key file must be specified")
	}

	var cacert []byte

	if cacertfile == "" {
		issuer := ctx.String(issuerFlag.Name)
		apiURL, err := url.Parse(getAPIUrl(ctx))
		if err != nil {
			logrus.WithError(err).Error("Failed to parse GraphQL API URL")
			return err
		}
		if issuer == "" || apiURL == nil {
			return fmt.Errorf("either %s or %s and %s must be specified", caCertInfileFlag.Name, issuerFlag.Name, drivrAPIURLFlag.Name)
		}
		cacert, err = getCaCert(ctx.Context, issuer, apiURL, getAPIKey())
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
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("ssl://%s:%d", mqttBroker, mqttBrokerPort))

	tlsConfig, err := newTLSConfig(cacert, certificateFile, privKeyFile)
	if err != nil {
		return err
	}
	opts.SetTLSConfig(tlsConfig)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}

	fmt.Println("Successfully used certificate to connect to MQTT broker!")

	topic := ctx.String(topicFlag.Name)
	if topic != "" {
		token := client.Subscribe(topic, 0, func(client mqtt.Client, msg mqtt.Message) {
			fmt.Printf("Received message on topic %s: %s\n", msg.Topic(), string(msg.Payload()))
		})
		if token.Wait() && token.Error() != nil {
			return token.Error()
		}

		m := token.(*mqtt.SubscribeToken).Result()
		if m == nil {
			return fmt.Errorf("Failed subscribing to topic: %s: invalid broker response", topic)
		}

		if m[topic] == 0 {
			fmt.Printf("Subscribed to topic: %s\n", topic)
		} else {
			fmt.Printf("Failed subscribing to topic: %s: code: %d\n", topic, m[topic])
		}
	}

	return nil
}
