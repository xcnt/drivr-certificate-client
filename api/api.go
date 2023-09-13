package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"

	"github.com/shurcooL/graphql"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

type loggingTransport struct {
	wrapped http.RoundTripper
}

func (s *loggingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	reqBytes, _ := httputil.DumpRequestOut(r, true)
	fmt.Printf("%s\n", reqBytes)
	fmt.Println("==============")
	resp, err := s.wrapped.RoundTrip(r)
	// err is returned after dumping the response

	respBytes, _ := httputil.DumpResponse(resp, true)
	fmt.Printf("%s\n", respBytes)
	fmt.Println("==============")

	logrus.WithField("request", string(reqBytes)).WithField("response", string(respBytes)).Debug("sending request to graphql server")

	return resp, err
}

func injectLoggingTransport(c *http.Client) {
	if c.Transport == nil {
		c.Transport = http.DefaultTransport
	}
	c.Transport = &loggingTransport{c.Transport}
}

func NewClient(apiURL, apiToken string) (*graphql.Client, error) {
	var httpClient *http.Client

	if apiToken != "" {
		src := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: apiToken},
		)
		httpClient = oauth2.NewClient(context.Background(), src)
	}

	if logrus.GetLevel() == logrus.DebugLevel {
		if httpClient == nil {
			httpClient = http.DefaultClient
		}
		injectLoggingTransport(httpClient)
	}

	client := graphql.NewClient(apiURL, httpClient)
	return client, nil
}
