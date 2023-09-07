package api

import (
	"github.com/shurcooL/graphql"
)

type CreateCertificateMutation struct {
	CreateCertificate struct {
		UUID graphql.String
	} `graphql:"createCertificate(name: $name, duration: $duration, csr: $csr)"`
}

type FetchCertificateQuery struct {
	FetchCertificate struct {
		Name        graphql.String
		Certificate graphql.String
	} `graphql:"certificate(uuid: $uuid)"`
}
