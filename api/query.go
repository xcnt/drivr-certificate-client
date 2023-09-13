package api

import (
	"github.com/shurcooL/graphql"
)

type Certificate struct {
	Uuid graphql.String
}

type CertificateWithName struct {
	Name        graphql.String
	Certificate graphql.String
}

type CreateCertificateMutation struct {
	Certificate `graphql:"createCertificate(issuerUuid: $issuerUuid, name: $name, duration: $duration, csr: $csr)"`
}

type CreateCertificateWithEntityMutation struct {
	Certificate `graphql:"createCertificate(issuerUuid: $issuerUuid, name: $name, duration: $duration, csr: $csr, entityUuid: $entityUuid)"`
}

type FetchCertificateQuery struct {
	CertificateWithName `graphql:"certificate(uuid: $uuid)"`
}

type FetchIssuerUUIDQuery struct {
	Issuer struct {
		Items []struct {
			Uuid graphql.String
		}
	} `graphql:"issuers(where: {name: {_eq: $name}}, limit: 1)"`
}

type FetchCaQuery struct {
	CA struct {
		Items []struct {
			Ca graphql.String
		}
	} `graphql:"issuers(where: {name: {_eq: $name}}, limit: 1)"`
}
