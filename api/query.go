package api

import (
	"github.com/shurcooL/graphql"
)

type CreateCertificateMutation struct {
	CreateCertificate struct {
		Uuid graphql.String
	} `graphql:"createCertificate(issuerUuid: $issuerUuid, name: $name, duration: $duration, csr: $csr, entityUuid: $entityUuid)"`
}

type FetchCertificateQuery struct {
	FetchCertificate struct {
		Name        graphql.String
		Certificate graphql.String
	} `graphql:"certificate(uuid: $uuid)"`
}

type FetchIssuerUUIDQuery struct {
	FetchIssuer struct {
		Items []struct {
			Uuid graphql.String
		}
	} `graphql:"issuers(where: {name: {_eq: $name}}, limit: 1)"`
}

type FetchCaQuery struct {
	FetchCa struct {
		Items []struct {
			Ca graphql.String
		}
	} `graphql:"issuers(where: {name: {_eq: $name}}, limit: 1)"`
}
