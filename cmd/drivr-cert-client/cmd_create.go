package main

import (
	"github.com/urfave/cli/v2"
	"github.com/xcnt/drivr-certificate-client/cert"
)

var (
	keyBitsFlag = &cli.IntFlag{
		Name:    "key-bits",
		Aliases: []string{"b"},
		Usage:   "Number of bits for the key",
		Value:   2048,
	}
	keyOutfileFlag = &cli.StringFlag{
		Name:    "key-outfile",
		Aliases: []string{"k"},
		Usage:   "Output file for the key",
		Value:   "key.pem",
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
			keyOutfileFlag,
		},
	}
}

func createKeyPair(ctx *cli.Context) error {
	return cert.GenerateRSAKeyPair(ctx.Int(keyBitsFlag.Name), ctx.String(keyOutfileFlag.Name))
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
