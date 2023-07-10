package cert

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
)

// GenerateRSAKeyPair generates a new RSA key pair of the given bit size.
func GenerateRSAKeyPair(bits int, outfile string) error {
	key, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		logrus.WithError(err).Error("Failed to generate key")
		return err
	}
	if err := writePrivateKeyToFile(key, outfile); err != nil {
		logrus.WithError(err).Error("Failed to write private key to file")
		return err
	}
	return nil
}

func writePrivateKeyToFile(keys *rsa.PrivateKey, fileName string) error {
	var buffer bytes.Buffer
	if err := pem.Encode(&buffer, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(keys)}); err != nil {
		return err
	}

	var _, err = os.Stat(fileName)
	if err == nil {
		logrus.WithField("outfile", fileName).Error("Key file already exists")
		return fmt.Errorf("Key file %s already exists", fileName)
	}

	f, err := os.Create(fileName)
	if err != nil {
		logrus.WithError(err).Error("Failed to create key file")
		return err
	}
	defer f.Close()

	_, err = f.Write(buffer.Bytes())
	return err
}
