package cert

import (
	"bytes"
	"encoding/pem"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
)

type PEMType string

const (
	RSAPrivateKey      PEMType = "RSA PRIVATE KEY"
	RSAPublicKey       PEMType = "RSA PUBLIC KEY"
	CertificateRequest PEMType = "CERTIFICATE REQUEST"
	Certificate        PEMType = "CERTIFICATE"
)

func WriteToPEMFile(keyType PEMType, keyBytes []byte, filename string) error {
	var buffer bytes.Buffer
	if err := pem.Encode(&buffer, &pem.Block{Type: string(keyType), Bytes: keyBytes}); err != nil {
		return err
	}

	var _, err = os.Stat(filename)
	if err == nil {
		logrus.WithField("outfile", filename).Error("file already exists")
		return fmt.Errorf("file %s already exists", filename)
	}

	f, err := os.Create(filename)
	if err != nil {
		logrus.WithError(err).WithField("filename", filename).Error("Failed to create PEM file")
		return err
	}
	defer f.Close()

	_, err = f.Write(buffer.Bytes())
	return err
}
