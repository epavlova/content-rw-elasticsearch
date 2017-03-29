package main

import (
	"encoding/json"
	"github.com/Financial-Times/go-fthealth/v1a"
	"github.com/Financial-Times/http-handlers-go/httphandlers"
	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/jawher/mow.cli"
	"github.com/kr/pretty"
	"github.com/rcrowley/go-metrics"
	"gopkg.in/olivere/elastic.v2"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
)

var esServiceInstance esServiceI

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

	sourceAddresses := app.Strings(cli.StringsOpt{
		Name:   "source-addresses",
		Value:  []string{},
		Desc:   "Addresses used by the queue consumer to connect to the queue",
		EnvVar: "SRC_ADDR",
	})
	sourceGroup := app.String(cli.StringOpt{
		Name:   "source-group",
		Value:  "",
		Desc:   "Group used to read the messages from the queue",
		EnvVar: "SRC_GROUP",
	})
	sourceTopic := app.String(cli.StringOpt{
		Name:   "source-topic",
		Value:  "",
		Desc:   "The topic to read the meassages from",
		EnvVar: "SRC_TOPIC",
	})
	sourceQueue := app.String(cli.StringOpt{
		Name:   "source-queue",
		Value:  "",
		Desc:   "The header identifying the queue to read the messages from",
		EnvVar: "SRC_QUEUE",
	})
	sourceConcurrentProcessing := app.Bool(cli.BoolOpt{
		Name:   "source-concurrent-processing",
		Value:  false,
		Desc:   "Whether the consumer uses concurrent processing for the messages",
		EnvVar: "SRC_CONCURRENT_PROCESSING",
	})

	queueConfig := consumer.QueueConfig{
		Addrs:                *sourceAddresses,
		Group:                *sourceGroup,
		Topic:                *sourceTopic,
		Queue:                *sourceQueue,
		ConcurrentProcessing: *sourceConcurrentProcessing,
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
		esServiceInstance = newEsService(elasticClient, *indexName)

		contentWriter := newESWriter(&esServiceInstance)

		//create health service
		var esHealthService esHealthServiceI = newEsHealthService(elasticClient)
		healthService := newHealthService(&esHealthService)

		routeRequests(port, contentWriter, healthService)

		readMessages(queueConfig)
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

	//todo add Kafka check
	http.HandleFunc("/__health", v1a.Handler("Amazon Elasticsearch Service Healthcheck", "Checks for AES", healthService.connectivityHealthyCheck(), healthService.clusterIsHealthyCheck()))
	http.HandleFunc("/__health-details", healthService.HealthDetails)
	http.HandleFunc("/__gtg", healthService.GoodToGo)
	//todo __build-info

	http.Handle("/", monitoringRouter)

	if err := http.ListenAndServe(":"+*port, nil); err != nil {
		log.Fatalf("Unable to start: %v", err)
	}
}

func readMessages(config consumer.QueueConfig) {
	messageConsumer := consumer.NewConsumer(config, handleMessage, http.Client{})
	log.Printf("[Startup] Consumer: %# v", pretty.Formatter(messageConsumer))

	var consumerWaitGroup sync.WaitGroup
	consumerWaitGroup.Add(1)

	go func() {
		messageConsumer.Start()
		consumerWaitGroup.Done()
	}()

	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	messageConsumer.Stop()
	consumerWaitGroup.Wait()
}

func handleMessage(msg consumer.Message) {
	tid := msg.Headers["X-Request-Id"]

	var combinedPostPublicationEvent enrichedContentModel
	err := json.Unmarshal([]byte(msg.Body), &combinedPostPublicationEvent)
	if err != nil {
		log.Errorf("[%s] Cannot unmarshal message body:[%v]", tid, err.Error())
		return
	}

	uuid := combinedPostPublicationEvent.Content.UUID
	log.Printf("[%s] Processing combined post publication event for uuid [%s]", tid, uuid)

	var contentType string

	for _, identifier := range combinedPostPublicationEvent.Content.Identifiers {
		if strings.HasPrefix(identifier.Authority, "http://api.ft.com/system/FT-LABS-WP") {
			contentType = "blogPost"
		} else if strings.HasPrefix(identifier.Authority, "http://api.ft.com/system/FTCOM-METHODE") {
			contentType = "article"
		} else if strings.HasPrefix(identifier.Authority, "http://api.ft.com/system/BRIGHTCOVE") {
			contentType = "video"
		}
	}

	if contentType == "" {
		log.Errorf("Failed to index content with UUID %s. Could not infer type of content.", uuid)
		return
	}

	if combinedPostPublicationEvent.Content.MarkedDeleted {
		_, err = esServiceInstance.deleteData(contentTypeMap[contentType].collection, uuid)
		if err != nil {
			log.Errorf(err.Error())
			return
		}
	} else {
		payload := convertToESContentModel(combinedPostPublicationEvent, contentType)

		_, err = esServiceInstance.writeData(contentTypeMap[contentType].collection, uuid, payload)
		if err != nil {
			log.Errorf(err.Error())
			return
		}
	}
}
