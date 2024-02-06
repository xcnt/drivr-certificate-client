package cert

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"

	"github.com/google/uuid"
)

func CreateCSR(privKey *rsa.PrivateKey) ([]byte, error) {
	subject := pkix.Name{
		CommonName: uuid.New().String(),
	}
	var csrTemplate = x509.CertificateRequest{
		Subject:            subject,
		SignatureAlgorithm: x509.SHA512WithRSA,
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
