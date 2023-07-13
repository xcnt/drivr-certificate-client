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
func GenerateRSAKeyPair(bits int, privOutfile, pubOutfile string) error {
	privKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		logrus.WithError(err).Error("Failed to generate key")
		return err
	}
	logrus.WithField("name", privOutfile).Info("Generate private RSA key")
	if err := writeKeyToFile("RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(privKey), privOutfile); err != nil {
		logrus.WithError(err).Error("Failed to write private key to file")
		return err
	}

	pubKey := &privKey.PublicKey
	logrus.WithField("name", pubOutfile).Info("Generate public RSA key")
	if err := writeKeyToFile("RSA PUBLIC KEY", x509.MarshalPKCS1PublicKey(pubKey), pubOutfile); err != nil {
		logrus.WithError(err).Error("Failed to write public key to file")
		return err
	}

	return nil
}

func writeKeyToFile(keyType string, keyBytes []byte, fileName string) error {
	var buffer bytes.Buffer
	if err := pem.Encode(&buffer, &pem.Block{Type: keyType, Bytes: keyBytes}); err != nil {
		return err
	}

	var _, err = os.Stat(fileName)
	if err == nil {
		logrus.WithField("outfile", fileName).Error("Key file already exists")
		return fmt.Errorf("Key file %s already exists", fileName)
	}

	f, err := os.Create(fileName)
	if err != nil {
		logrus.WithError(err).WithField("filename", fileName).Error("Failed to create key file")
		return err
	}
	defer f.Close()

	_, err = f.Write(buffer.Bytes())
	return err
}
