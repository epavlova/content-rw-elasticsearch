package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	health "github.com/Financial-Times/go-fthealth/v1_1"
	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	"github.com/Financial-Times/service-status-go/gtg"
	"github.com/Financial-Times/go-logger"
)

type healthService struct {
	esHealthService  esHealthServiceI
	consumerInstance consumer.MessageConsumer
	httpClient       *http.Client
	checks           []health.Check
}

func newHealthService(config *consumer.QueueConfig, esHealthService esHealthServiceI) *healthService {
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConnsPerHost:   20,
			TLSHandshakeTimeout:   3 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
	consumerInstance := consumer.NewConsumer(*config, func(m consumer.Message) {}, client)
	service := &healthService{
		esHealthService:  esHealthService,
		consumerInstance: consumerInstance,
		httpClient:       client,
	}
	service.checks = []health.Check{
		service.clusterIsHealthyCheck(),
		service.connectivityHealthyCheck(),
		service.schemaHealthyCheck(),
		service.checkKafkaProxyConnectivity()}
	return service
}

func (service *healthService) clusterIsHealthyCheck() health.Check {
	return health.Check{
		BusinessImpact:   "Full or partial degradation in serving requests from Elasticsearch",
		Name:             "Check Elasticsearch cluster health",
		PanicGuide:       "https://dewey.ft.com/content-rw-elasticsearch.html",
		Severity:         1,
		TechnicalSummary: "Elasticsearch cluster is not healthy. Details on /__health-details",
		Checker:          service.healthChecker,
	}
}

func (service *healthService) healthChecker() (string, error) {
	output, err := service.esHealthService.getClusterHealth()
	if err != nil {
		return "Cluster is not healthy: ", err
	} else if output.Status != "green" {
		return "Cluster is not healthy", fmt.Errorf("Cluster is %v", output.Status)
	} else {
		return "Cluster is healthy", nil
	}
}

func (service *healthService) connectivityHealthyCheck() health.Check {
	return health.Check{
		BusinessImpact:   "Could not connect to Elasticsearch",
		Name:             "Check connectivity to the Elasticsearch cluster",
		PanicGuide:       "https://dewey.ft.com/content-rw-elasticsearch.html",
		Severity:         1,
		TechnicalSummary: "Connection to Elasticsearch cluster could not be created. Please check your AWS credentials.",
		Checker:          service.connectivityChecker,
	}
}

func (service *healthService) connectivityChecker() (string, error) {
	_, err := service.esHealthService.getClusterHealth()
	if err != nil {
		return "Could not connect to elasticsearch", err
	}

	return "Successfully connected to the cluster", nil
}

func (service *healthService) schemaHealthyCheck() health.Check {
	return health.Check{
		BusinessImpact:   "Search results may be inconsistent",
		Name:             "Check Elasticsearch mapping",
		PanicGuide:       "https://dewey.ft.com/content-rw-elasticsearch.html",
		Severity:         1,
		TechnicalSummary: "Elasticsearch mapping does not match expected mapping. Please check index against the reference https://github.com/Financial-Times/content-rw-elasticsearch/blob/master/runtime/referenceSchema.json",
		Checker:          service.schemaChecker,
	}
}

func (service *healthService) schemaChecker() (string, error) {
	output, err := service.esHealthService.getSchemaHealth()
	if err != nil {
		return "Could not get schema: ", err
	} else if output != "ok" {
		return "Schema is not healthy", fmt.Errorf("Schema is %v", output)
	} else {
		return "Schema is healthy", nil
	}
}

func (service *healthService) checkKafkaProxyConnectivity() health.Check {
	return health.Check{
		BusinessImpact:   "CombinedPostPublication messages can't be read from the queue. Indexing for search won't work.",
		Name:             "Check kafka-proxy connectivity.",
		PanicGuide:       "https://dewey.ft.com/content-rw-elasticsearch.html",
		Severity:         1,
		TechnicalSummary: "Messages couldn't be read from the queue. Check if kafka-proxy is reachable.",
		Checker:          service.consumerInstance.ConnectivityCheck,
	}
}

func (service *healthService) gtgCheck() gtg.Status {
	for _, check := range service.checks {
		if _, err := check.Checker(); err != nil {
			return gtg.Status{GoodToGo: false, Message: err.Error()}
		}
	}
	return gtg.Status{GoodToGo: true}
}

//HealthDetails returns the response from elasticsearch service /__health endpoint - describing the cluster health
func (service *healthService) HealthDetails(writer http.ResponseWriter, req *http.Request) {

	writer.Header().Set("Content-Type", "application/json")

	output, err := service.esHealthService.getClusterHealth()
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	var response []byte
	response, err = json.Marshal(*output)
	if err != nil {
		response = []byte(err.Error())
	}

	_, err = writer.Write(response)
	if err != nil {
		logger.WithError(err).Error(err.Error())
	}
}
