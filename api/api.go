package api

import (
	"context"

	"github.com/shurcooL/graphql"
	"golang.org/x/oauth2"
)

func NewClient(apiURL, apiToken string) (*graphql.Client, error) {
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: apiToken},
	)
	httpClient := oauth2.NewClient(context.Background(), src)

	client := graphql.NewClient(apiURL, httpClient)
	return client, nil
}
