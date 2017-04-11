package main

import (
	"encoding/json"
	"github.com/Financial-Times/go-fthealth/v1a"
	"github.com/Financial-Times/http-handlers-go/httphandlers"
	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/kr/pretty"
	"github.com/rcrowley/go-metrics"
	"gopkg.in/olivere/elastic.v2"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

const SYNTHETIC_REQUEST_PREFIX = "SYNTHETIC-REQ-MON"

type contentIndexer struct {
	esServiceInstance esService
}

func (indexer contentIndexer) start(indexName string, port string, accessConfig esAccessConfig, queueConfig consumer.QueueConfig) {
	channel := make(chan *elastic.Client)
	go func() {
		defer close(channel)
		for {
			ec, err := newAmazonClient(accessConfig)
			if err == nil {
				log.Infof("connected to ElasticSearch")
				channel <- ec
				return
			} else {
				log.Errorf("could not connect to ElasticSearch: %s", err.Error())
				time.Sleep(time.Minute)
			}
		}
	}()

	//create writer service
	esServiceInstance := newEsService(indexName)

	go func() {
		for ec := range channel {
			esServiceInstance.elasticClient = ec
			indexer.startMessageConsumer(queueConfig)
		}
	}()

	indexer.serveAdminEndpoints(port)
}

func (indexer contentIndexer) serveAdminEndpoints(port string) {
	healthService := newHealthService(indexer.esServiceInstance)
	var monitoringRouter http.Handler = mux.NewRouter()
	monitoringRouter = httphandlers.TransactionAwareRequestLoggingHandler(log.StandardLogger(), monitoringRouter)
	monitoringRouter = httphandlers.HTTPMetricsHandler(metrics.DefaultRegistry, monitoringRouter)

	//todo add Kafka check
	http.HandleFunc("/__health", v1a.Handler("Amazon Elasticsearch Service Healthcheck", "Checks for AES", healthService.connectivityHealthyCheck(), healthService.clusterIsHealthyCheck()))
	http.HandleFunc("/__health-details", healthService.HealthDetails)
	http.HandleFunc("/__gtg", healthService.GoodToGo)
	//todo __build-info

	http.Handle("/", monitoringRouter)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Unable to start: %v", err)
	}
}

func (indexer contentIndexer) startMessageConsumer(config consumer.QueueConfig) {
	messageConsumer := consumer.NewConsumer(config, indexer.handleMessage, &http.Client{})
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

func (indexer contentIndexer) handleMessage(msg consumer.Message) {

	tid := msg.Headers["X-Request-Id"]

	if strings.Contains(tid, SYNTHETIC_REQUEST_PREFIX) {
		log.Infof("[%s] Ignoring synthetic message", tid)
		return
	}

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
		origin := msg.Headers["Origin-System-Id"]
		if strings.Contains(origin, "methode-web-pub") {
			contentType = "article"
		} else if strings.Contains(origin, "wordpress") {
			contentType = "blogPost"
		} else if strings.Contains(origin, "brightcove") {
			contentType = "video"
		} else {
			log.Errorf("Failed to index content with UUID %s. Could not infer type of content.", uuid)
			return
		}
	}

	if combinedPostPublicationEvent.Content.MarkedDeleted {
		_, err = indexer.esServiceInstance.deleteData(contentTypeMap[contentType].collection, uuid)
		if err != nil {
			log.Errorf(err.Error())
			return
		}
	} else {
		payload := convertToESContentModel(combinedPostPublicationEvent, contentType)

		_, err = indexer.esServiceInstance.writeData(contentTypeMap[contentType].collection, uuid, payload)
		if err != nil {
			log.Errorf(err.Error())
			return
		}
	}
}
