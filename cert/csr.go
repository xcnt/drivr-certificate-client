package cert

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"

	"github.com/google/uuid"
)

type CSRData struct {
	ServerName string
}

func CreateCSR(privKey *rsa.PrivateKey, csrData *CSRData) ([]byte, error) {
	subject := pkix.Name{
		CommonName: uuid.New().String(),
	}
	dnsNames := []string{}
	if csrData != nil && csrData.ServerName != "" {
		dnsNames = append(dnsNames, csrData.ServerName)
	}
	var csrTemplate = x509.CertificateRequest{
		Subject:            subject,
		SignatureAlgorithm: x509.SHA512WithRSA,
		DNSNames:           dnsNames,
	}

	csrCertificate, err := x509.CreateCertificateRequest(rand.Reader, &csrTemplate, privKey)
	if err != nil {
		return nil, err
	}
	csr := pem.EncodeToMemory(&pem.Block{
		Type: string(CertificateRequest), Bytes: csrCertificate,
	})
	return csr, nil
}
