package main

import (
	"github.com/urfave/cli/v2"
	"github.com/xcnt/drivr-certificate-client/cert"
)

const (
	PRIVATE_KEY_FILE = "private.key"
	PUBLIC_KEY_FILE  = "public.key"
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
		Value:   PUBLIC_KEY_FILE,
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
	}
}

func createCertificate(ctx *cli.Context) error {
	return nil
}
