package api

import (
	"github.com/shurcooL/graphql"
)

type CreateCertificateMutation struct {
	CreateCertificate struct {
		name     graphql.String
		duration graphql.Int
		csr      graphql.String
	} `graphql:"createCreateCertificate(name: $name, duration: $duration, csr: $csr)"`
}

type FetchCertificateQuery struct {
	FetchCertificate struct {
		name graphql.String
	} `graphql:"fetchCertificate(name: $name)"`
}
