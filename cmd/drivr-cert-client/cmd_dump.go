package main

import (
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"github.com/xcnt/drivr-certificate-client/cert"
)

const (
	PUBLIC_KEY_FILE = "public.key"
)

var (
	dumpPubKeyOutfileFlag = &cli.StringFlag{
		Name:    "pubkey-outfile",
		Aliases: []string{"u"},
		Usage:   "Output file for the generated public key",
		Value:   PUBLIC_KEY_FILE,
	}
)

func dumpCommand() *cli.Command {
	return &cli.Command{
		Name:  "dump",
		Usage: "dump public key",
		Flags: []cli.Flag{
			dumpPubKeyOutfileFlag,
			privateKeyInfileFlag,
		},
		Action: func(c *cli.Context) error {
			privKey, err := cert.LoadPrivateKey(c.String(privateKeyInfileFlag.Name))
			if err != nil {
				logrus.WithError(err).Error("failed to load private key")
				return err
			}
			return cert.DumpPublicKey(privKey, c.String(dumpPubKeyOutfileFlag.Name))
		},
	}
}
