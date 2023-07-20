package cert

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"

	"github.com/sirupsen/logrus"
)

// GenerateRSAKeyPair generates a new RSA key pair of the given bit size.
func GenerateRSAKeyPair(bits int, privOutfile, pubOutfile string) error {
	privKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		logrus.WithError(err).Error("Failed to generate key")
		return err
	}
	logrus.WithField("name", privOutfile).Debug("Generate private RSA key")
	if err := WriteToPEMFile(RSAPrivateKey, x509.MarshalPKCS1PrivateKey(privKey), privOutfile); err != nil {
		logrus.WithError(err).Error("Failed to write private key to file")
		return err
	}

	if pubOutfile == "" {
		return nil
	}

	return DumpPublicKey(privKey, pubOutfile)
}

func DumpPublicKey(privateKey *rsa.PrivateKey, filename string) error {
	pubKey := &privateKey.PublicKey
	logrus.WithField("name", filename).Debug("Generate public RSA key")
	if err := WriteToPEMFile(RSAPublicKey, x509.MarshalPKCS1PublicKey(pubKey), filename); err != nil {
		logrus.WithError(err).Error("Failed to write public key to file")
		return err
	}

	return nil
}

func LoadPrivateKey(filename string) (*rsa.PrivateKey, error) {
	PEMBytes, err := os.ReadFile(filename)
	if err != nil {
		logrus.WithError(err).Error("Failed to read private key from file")
		return nil, err
	}

	keyBytes, _ := pem.Decode(PEMBytes)
	if keyBytes == nil {
		logrus.WithField("keyfile", filename).Error("Failed to decode private key")
		return nil, errors.New("Failed to decode private key")
	}

	privKey, err := x509.ParsePKCS1PrivateKey(keyBytes.Bytes)
	if err != nil {
		logrus.WithError(err).Error("Failed to parse private key")
		return nil, err
	}

	return privKey, nil
}
