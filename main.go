package main

import (
	"github.com/jawher/mow.cli"
	"gopkg.in/olivere/elastic.v2"
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/Financial-Times/http-handlers-go/httphandlers"
	"github.com/rcrowley/go-metrics"
	"github.com/Financial-Times/go-fthealth/v1a"
	"net/http"
	"os"
)

func main() {
	app := cli.App("content-rw-es", "Service for loading contents into elasticsearch")
	port := app.String(cli.StringOpt{
		Name:   "port",
		Value:  "8080",
		Desc:   "Port to listen on",
		EnvVar: "PORT",
	})
	accessKey := app.String(cli.StringOpt{
		Name:   "aws-access-key",
		Desc:   "AWS ACCES KEY",
		EnvVar: "AWS_ACCESS_KEY_ID",
	})
	secretKey := app.String(cli.StringOpt{
		Name:   "aws-secret-access-key",
		Desc:   "AWS SECRET ACCES KEY",
		EnvVar: "AWS_SECRET_ACCESS_KEY",
	})
	esEndpoint := app.String(cli.StringOpt{
		Name:   "elasticsearch-sapi-endpoint",
		Value:  "http://localhost:9200",
		Desc:   "AES endpoint",
		EnvVar: "ELASTICSEARCH_SAPI_ENDPOINT",
	})
	indexName := app.String(cli.StringOpt{
		Name:   "index-name",
		Value:  "ft",
		Desc:   "The name of the elaticsearch index",
		EnvVar: "ELASTICSEARCH_SAPI_INDEX",
	})

	accessConfig := esAccessConfig{
		accessKey:  *accessKey,
		secretKey:  *secretKey,
		esEndpoint: *esEndpoint,
	}

	log.SetLevel(log.InfoLevel)
	log.Infof("[Startup] Content RW Elasticsearch is starting ")

	app.Action = func() {
		var elasticClient *elastic.Client
		var err error

		elasticClient, err = newAmazonClient(accessConfig)

		if err != nil {
			log.Fatalf("Creating elasticsearch client failed with error=[%v]\n", err)
		}

		//create writer service
		var esService esServiceI = newEsService(elasticClient, *indexName)

		contentWriter := newESWriter(&esService)

		//create health service
		var esHealthService esHealthServiceI = newEsHealthService(elasticClient)
		healthService := newHealthService(&esHealthService)

		routeRequests(port, contentWriter, healthService)
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Errorf("App could not start, error=[%s]\n", err)
		return
	}
}

func routeRequests(port *string, contentWriter *contentWriter, healthService *healthService) {
	servicesRouter := mux.NewRouter()
	servicesRouter.HandleFunc("/{content-type}/{id}", contentWriter.writeData).Methods("PUT")
	servicesRouter.HandleFunc("/{content-type}/{id}", contentWriter.readData).Methods("GET")
	servicesRouter.HandleFunc("/{content-type}/{id}", contentWriter.deleteData).Methods("DELETE")

	var monitoringRouter http.Handler = servicesRouter
	monitoringRouter = httphandlers.TransactionAwareRequestLoggingHandler(log.StandardLogger(), monitoringRouter)
	monitoringRouter = httphandlers.HTTPMetricsHandler(metrics.DefaultRegistry, monitoringRouter)

	http.HandleFunc("/__health", v1a.Handler("Amazon Elasticsearch Service Healthcheck", "Checks for AES", healthService.connectivityHealthyCheck(), healthService.clusterIsHealthyCheck()))
	http.HandleFunc("/__health-details", healthService.HealthDetails)
	http.HandleFunc("/__gtg", healthService.GoodToGo)

	http.Handle("/", monitoringRouter)

	if err := http.ListenAndServe(":" + *port, nil); err != nil {
		log.Fatalf("Unable to start: %v", err)
	}
}

