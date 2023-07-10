package main

import (
	"github.com/urfave/cli/v2"
	"github.com/xcnt/drivr-certificate-client/cert"
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
	}
}

func createKeyPair(ctx *cli.Context) error {
	return cert.GenerateRSAKeyPair(2048, "key.pem")
}

func certificateCommand() *cli.Command {
	return &cli.Command{
		Name:   "certificate",
		Usage:  "Create a new certificate",
		Action: createCertificate,
	}
}

func createCertificate(ctx *cli.Context) error {
	return nil
}
