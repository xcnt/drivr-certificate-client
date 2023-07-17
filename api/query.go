package api

import (
	"github.com/shurcooL/graphql"
)

type CreateCertificateMutation struct {
	CreateCertificate struct {
		Name     graphql.String
		Duration graphql.Int
		Csr      graphql.String
	} `graphql:"createCertificate(name: $name, duration: $duration, csr: $csr)"`
}

type FetchCertificateQuery struct {
	FetchCertificate struct {
		Name        graphql.String
		Certificate graphql.String
	} `graphql:"fetchCertificate(name: $name)"`
}
