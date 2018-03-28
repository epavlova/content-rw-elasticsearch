package es

import (
	awsauth "github.com/smartystreets/go-aws-auth"
	"gopkg.in/olivere/elastic.v2"
	"net/http"
	"github.com/Financial-Times/go-logger"
)

type ClientI interface {
	ClusterHealth() *elastic.ClusterHealthService
	Index() *elastic.IndexService
	Get() *elastic.GetService
	Delete() *elastic.DeleteService
	IndexGet() *elastic.IndicesGetService
}

type AccessConfig struct {
	AccessKey string
	SecretKey string
	Endpoint  string
}

type AWSSigningTransport struct {
	HTTPClient  *http.Client
	Credentials awsauth.Credentials
}

// RoundTrip implementation
func (a AWSSigningTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return a.HTTPClient.Do(awsauth.Sign4(req, a.Credentials))
}

var NewAmazonClient = func(config AccessConfig) (ClientI, error) {

	signingTransport := AWSSigningTransport{
		Credentials: awsauth.Credentials{
			AccessKeyID:     config.AccessKey,
			SecretAccessKey: config.SecretKey,
		},
		HTTPClient: http.DefaultClient,
	}
	signingClient := &http.Client{Transport: http.RoundTripper(signingTransport)}

	return elastic.NewClient(
		elastic.SetURL(config.Endpoint),
		elastic.SetScheme("https"),
		elastic.SetHttpClient(signingClient),
		elastic.SetSniff(false), //needs to be disabled due to EAS behavior. Healthcheck still operates as normal.
		elastic.SetErrorLog(logger.Logger()),
	)
}
