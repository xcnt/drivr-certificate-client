package api

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/google/uuid"
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

func newClient(apiURL url.URL, apiToken string) (*graphql.Client, error) {
	var httpClient *http.Client

	if apiToken == "" {
		clientID, clientSecret, err := getOAuthCredentials()
		if err != nil {
			return nil, err
		}
		apiToken = oauthFlow(apiURL, clientID, clientSecret)
	}

	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: apiToken},
	)
	httpClient = oauth2.NewClient(context.Background(), src)

	if logrus.GetLevel() == logrus.DebugLevel {
		if httpClient == nil {
			httpClient = http.DefaultClient
		}
		injectLoggingTransport(httpClient)
	}

	graphQLURL := apiURL.JoinPath("graphql")

	logrus.WithField("graphql_url", graphQLURL.String()).Debug("init graphql client")
	client := graphql.NewClient(graphQLURL.String(), httpClient)
	return client, nil
}

type DrivrAPI struct {
	client *graphql.Client
}

func NewDrivrAPI(apiURL *url.URL, apiToken string) (*DrivrAPI, error) {
	client, err := newClient(*apiURL, apiToken)
	if err != nil {
		return nil, err
	}

	return &DrivrAPI{client: client}, nil
}

func (d *DrivrAPI) FetchCertificateAuthority(ctx context.Context, issuer string) ([]byte, error) {
	var query FetchCaQuery

	err := d.client.Query(ctx, &query, map[string]interface{}{
		"name": graphql.String(issuer),
	})
	if err != nil {
		logrus.WithField("issuer", issuer).WithError(err).Error("Failed to query CA")
		return nil, err
	}

	if query.CA.Items[0].Ca == "" {
		logrus.WithField("issuer", issuer).Error("No CA found for issuer")
		return nil, err
	}

	ca, err := base64.RawStdEncoding.DecodeString(string(query.CA.Items[0].Ca))
	if err != nil {
		logrus.WithError(err).Error("Failed to decode ca certificate")
		return nil, err
	}
	return ca, nil
}

func (d *DrivrAPI) FetchCertificate(ctx context.Context, uuid *uuid.UUID) ([]byte, string, error) {
	var query FetchCertificateQuery

	vars := map[string]interface{}{
		"uuid": NewGraphQLUUID(*uuid),
	}

	err := d.client.Query(ctx, &query, vars)
	if err != nil {
		logrus.WithField("uuid", uuid).WithError(err).Error("Failed to query certificate")
		return nil, "", err
	}

	name := string(query.CertificateWithName.Name)

	if query.CertificateWithName.Certificate == "" {
		logrus.WithField("certificate_uuid", uuid).Debug("Certificate not yet signed")
		return nil, name, errors.New("Certificate not yet signed")
	}

	certificate, err := base64.StdEncoding.DecodeString(string(query.CertificateWithName.Certificate))
	if err != nil {
		logrus.WithError(err).Error("Failed to decode certificate")
		return nil, name, err
	}
	return certificate, name, nil
}

func (d *DrivrAPI) FetchIssuerUUID(ctx context.Context, name string) (*uuid.UUID, error) {
	var query FetchIssuerUUIDQuery

	vars := map[string]interface{}{
		"name": graphql.String(name),
	}

	err := d.client.Query(ctx, &query, vars)
	if err != nil {
		logrus.WithField("issuer", name).WithError(err).Error("Failed to query issuer")
		return nil, err
	}

	if len(query.Issuer.Items) != 1 {
		logrus.WithField("issuer", name).Error("Issuer not found")
		return nil, errors.New("Issuer not found")
	}

	uuidStr := string(query.Issuer.Items[0].Uuid)
	uuid, err := uuid.Parse(uuidStr)
	if err != nil {
		logrus.WithField("issuer_uuid", uuidStr).WithError(err).Error("Failed to parse issuer UUID")
		return nil, err
	}

	return &uuid, nil
}

func (d *DrivrAPI) FetchDomainUUID(ctx context.Context) (*uuid.UUID, error) {
	var query FetchDomainUUIDQuery

	err := d.client.Query(ctx, &query, nil)
	if err != nil {
		logrus.WithError(err).Error("Failed to query domain")
		return nil, err
	}

	uuidStr := string(query.CurrentDomain.Uuid)
	uuid, err := uuid.Parse(uuidStr)
	if err != nil {
		logrus.WithField("domain_uuid", uuidStr).WithError(err).Error("Failed to parse domain UUID")
		return nil, err
	}

	return &uuid, nil
}

func (d *DrivrAPI) FetchSystemUUID(ctx context.Context, code string) (*uuid.UUID, error) {
	var query FetchSystemUUIDQuery

	err := d.client.Query(ctx, &query, map[string]interface{}{
		"code": graphql.String(code),
	})
	if err != nil {
		logrus.WithError(err).Error("Failed to query system")
		return nil, err
	}

	if len(query.Systems.Items) == 0 {
		logrus.WithField("system_code", code).Error("System not found")
		return nil, errors.New("System not found")
	}

	uuidStr := string(query.Systems.Items[0].Uuid)
	uuid, err := uuid.Parse(uuidStr)
	if err != nil {
		logrus.WithField("system_uuid", uuidStr).WithError(err).Error("Failed to parse system UUID")
		return nil, err
	}

	return &uuid, nil
}

func (d *DrivrAPI) FetchComponentUUID(ctx context.Context, code string) (*uuid.UUID, error) {
	var query FetchComponentUUIDQuery

	err := d.client.Query(ctx, &query, map[string]interface{}{
		"code": graphql.String(code),
	})
	if err != nil {
		logrus.WithError(err).Error("Failed to query component")
		return nil, err
	}

	if len(query.Components.Items) == 0 {
		logrus.WithField("component_code", code).Error("Component not found")
		return nil, errors.New("Component not found")
	}

	uuidStr := string(query.Components.Items[0].Uuid)
	uuid, err := uuid.Parse(uuidStr)
	if err != nil {
		logrus.WithField("component_uuid", uuidStr).WithError(err).Error("Failed to parse component UUID")
		return nil, err
	}

	return &uuid, nil
}

func (d *DrivrAPI) CreateCertificate(ctx context.Context, issuerUuid, entityUuid *uuid.UUID, name, csr, duration string) (*uuid.UUID, error) {
	var mutation CreateCertificateMutation

	vars := map[string]interface{}{
		"issuerUuid": NewGraphQLUUID(*issuerUuid),
		"name":       graphql.String(name),
		"duration":   NewTimespan(duration),
		"csr":        graphql.String(csr),
		"entityUuid": NewGraphQLUUID(*entityUuid),
	}

	err := d.client.Mutate(ctx, &mutation, vars)
	if err != nil {
		return nil, err
	}

	uuid, err := uuid.Parse(string(mutation.Certificate.Uuid))
	return &uuid, err
}
