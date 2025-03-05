package api

//go:generate go run github.com/Khan/genqlient

import (
	"context"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/Khan/genqlient/graphql"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vektah/gqlparser/v2/gqlerror"
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

func newClient(apiURL url.URL, apiToken string) (graphql.Client, error) {
	var httpClient *http.Client

	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: apiToken, TokenType: "bearer"},
	)
	httpClient = oauth2.NewClient(context.Background(), src)

	if logrus.GetLevel() == logrus.DebugLevel {
		if httpClient == nil {
			httpClient = http.DefaultClient
		}
		injectLoggingTransport(httpClient)
	}

	graphQLURL := &apiURL
	if !strings.HasSuffix(graphQLURL.Path, "/graphql") {
		graphQLURL = graphQLURL.JoinPath("graphql")
	}

	logrus.WithField("graphql_url", graphQLURL.String()).Debug("init graphql client")
	client := graphql.NewClient(graphQLURL.String(), httpClient)
	return client, nil
}

type DrivrAPI struct {
	client graphql.Client
}

func NewDrivrAPI(apiURL *url.URL, apiToken string) (*DrivrAPI, error) {
	client, err := newClient(*apiURL, apiToken)
	if err != nil {
		return nil, err
	}

	return &DrivrAPI{client: client}, nil
}

func (d *DrivrAPI) FetchCertificateAuthority(ctx context.Context, issuer string) ([]byte, error) {
	resp, err := fetchCAByName(ctx, d.client, issuer)
	if err != nil {
		logrus.WithField("issuer", issuer).WithError(err).Error("Failed to query CA")
		return nil, err
	}

	if resp.Issuers.Items[0].Ca == "" {
		err := errors.New("No CA found for issuer")
		logrus.WithError(err).WithField("issuer", issuer).Error("Failed to fetch CA certificate")
		return nil, err
	}

	ca := resp.Issuers.Items[0].Ca

	decodedCa, _ := pem.Decode([]byte(ca))
	if decodedCa == nil {
		logrus.WithField("issuer", issuer).Error("Failed to decode CA certificate")
		return nil, errors.New("Failed to decode CA certificate")
	}

	return decodedCa.Bytes, nil
}

func (d *DrivrAPI) FetchCertificate(ctx context.Context, uuid *uuid.UUID) ([]byte, string, error) {
	resp, err := fetchCertificate(ctx, d.client, *uuid)
	if err != nil {
		logrus.WithField("uuid", uuid).WithError(err).Error("Failed to query certificate")
		return nil, "", err
	}

	name := resp.Certificate.Name

	if resp.Certificate.Certificate == "" {
		logrus.WithField("certificate_uuid", uuid).Debug("Certificate not yet signed")
		return nil, name, errors.New("Certificate not yet signed")
	}

	certificate := resp.Certificate.Certificate

	decodedCert, _ := pem.Decode([]byte(certificate))
	if decodedCert == nil {
		logrus.WithField("certificate_uuid", uuid).Error("Failed to decode certificate")
		return nil, name, errors.New("Failed to decode certificate")
	}

	return decodedCert.Bytes, name, nil
}

func (d *DrivrAPI) FetchIssuerUUID(ctx context.Context, name string) (*uuid.UUID, error) {
	resp, err := fetchIssuerUUIDByName(ctx, d.client, name)
	if err != nil {
		logrus.WithField("issuer", name).WithError(err).Error("Failed to query issuer")
		return nil, err
	}

	if len(resp.Issuers.Items) != 1 {
		logrus.WithField("issuer", name).Error("Issuer not found")
		return nil, errors.New("Issuer not found")
	}

	uuid := resp.Issuers.Items[0].Uuid
	return &uuid, nil
}

func (d *DrivrAPI) FetchDomainUUID(ctx context.Context) (*uuid.UUID, error) {
	resp, err := fetchDomainUUID(ctx, d.client)
	if err != nil {
		logrus.WithError(err).Error("Failed to query domain")
		return nil, err
	}

	uuid := resp.CurrentDomain.Uuid
	return &uuid, nil
}

func (d *DrivrAPI) FetchSystemUUID(ctx context.Context, code string) (*uuid.UUID, error) {
	resp, err := fetchSystemUUIDByCode(ctx, d.client, code)
	if err != nil {
		logrus.WithError(err).Error("Failed to query system")
		return nil, err
	}

	if len(resp.Systems.Items) == 0 {
		logrus.WithField("system_code", code).Error("System not found")
		return nil, errors.New("System not found")
	}

	uuid := resp.Systems.Items[0].Uuid
	return &uuid, nil
}

func (d *DrivrAPI) FetchComponentUUID(ctx context.Context, code string) (*uuid.UUID, error) {
	resp, err := fetchComponentUUIDByCode(ctx, d.client, code)
	if err != nil {
		logrus.WithError(err).Error("Failed to query component")
		return nil, err
	}

	if len(resp.Components.Items) == 0 {
		logrus.WithField("component_code", code).Error("Component not found")
		return nil, errors.New("Component not found")
	}

	uuid := resp.Components.Items[0].Uuid
	return &uuid, nil
}

type CreateCertificateInput struct {
	IssuerUUID   uuid.UUID
	EntityUUID   uuid.UUID
	Name         string
	CSR          string
	Duration     string
	AddServerUse bool
}

func (i CreateCertificateInput) LogFields() logrus.Fields {
	return logrus.Fields{
		"issuerUuid": i.IssuerUUID.String(),
		"name":       i.Name,
		"csr":        i.CSR,
		"duration":   i.Duration,
		"entityUuid": i.EntityUUID.String(),
		"serverUse":  i.AddServerUse,
	}
}

func (d *DrivrAPI) CreateCertificate(ctx context.Context, input CreateCertificateInput) (*uuid.UUID, error) {

	usages := []CertificateUsage{CertificateUsageClientAuth}
	if input.AddServerUse {
		usages = append(usages, CertificateUsageServerAuth)
	}

	resp, err := createCertificate(ctx, d.client, input.IssuerUUID, input.Name, input.Duration, input.CSR, input.EntityUUID, usages)
	if err != nil {
		extensions := err.(gqlerror.List)[0].Extensions
		sanitizedErrorMsg := ""
		if errors, ok := extensions["errors"]; ok {
			for code, msg := range errors.(map[string]interface{}) {
				sanitizedErrorMsg = fmt.Sprintf("%s %v [%s].", sanitizedErrorMsg, msg, code)
			}
		}
		newErr := fmt.Errorf("failed to create certificate: %v", sanitizedErrorMsg)
		return nil, newErr
	}

	uuid := resp.CreateCertificate.Uuid
	return &uuid, err
}
