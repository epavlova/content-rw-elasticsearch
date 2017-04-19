package main

import (
	"encoding/json"
	health "github.com/Financial-Times/go-fthealth/v1_1"
	"github.com/Financial-Times/http-handlers-go/httphandlers"
	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	status "github.com/Financial-Times/service-status-go/httphandlers"
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/kr/pretty"
	"github.com/rcrowley/go-metrics"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

const syntheticRequestPrefix = "SYNTHETIC-REQ-MON"

type contentIndexer struct {
	esServiceInstance esServiceI
}

func waitForSignal() {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
}

func (indexer *contentIndexer) start(appSystemCode string, indexName string, port string, accessConfig esAccessConfig, queueConfig consumer.QueueConfig) {
	channel := make(chan esClientI)
	go func() {
		defer close(channel)
		for {
			ec, err := newAmazonClient(accessConfig)
			if err == nil {
				log.Infof("connected to ElasticSearch")
				channel <- ec
				return
			}
			log.Errorf("could not connect to ElasticSearch: %s", err.Error())
			time.Sleep(time.Minute)
		}
	}()

	//create writer service
	indexer.esServiceInstance = newEsService(indexName)

	go func() {
		for ec := range channel {
			indexer.esServiceInstance.setClient(ec)
			indexer.startMessageConsumer(queueConfig)
		}
	}()

	go func() {
		indexer.serveAdminEndpoints(appSystemCode, port, queueConfig)
	}()
}

func (indexer *contentIndexer) serveAdminEndpoints(appSystemCode string, port string, queueConfig consumer.QueueConfig) {
	healthService := newHealthService(indexer.esServiceInstance, queueConfig.Topic, queueConfig.Addrs[0])
	var monitoringRouter http.Handler = mux.NewRouter()
	monitoringRouter = httphandlers.TransactionAwareRequestLoggingHandler(log.StandardLogger(), monitoringRouter)
	monitoringRouter = httphandlers.HTTPMetricsHandler(metrics.DefaultRegistry, monitoringRouter)

	serveMux := http.NewServeMux()

	hc := health.HealthCheck{SystemCode: appSystemCode, Name: appSystemCode, Description: "Content Read Writer for Elasticsearch", Checks: healthService.checks}
	serveMux.HandleFunc("/__health", health.Handler(hc))
	serveMux.HandleFunc("/__health-details", healthService.HealthDetails)
	serveMux.HandleFunc(status.GTGPath, status.NewGoodToGoHandler(healthService.gtgCheck))
	serveMux.HandleFunc(status.BuildInfoPath, status.BuildInfoHandler)

	serveMux.Handle("/", monitoringRouter)

	if err := http.ListenAndServe(":"+port, serveMux); err != nil {
		log.Fatalf("Unable to start: %v", err)
	}
}

func (indexer *contentIndexer) startMessageConsumer(config consumer.QueueConfig) {
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
	messageConsumer := consumer.NewConsumer(config, indexer.handleMessage, client)
	log.Printf("[Startup] Consumer: %# v", pretty.Formatter(messageConsumer))

	var consumerWaitGroup sync.WaitGroup
	consumerWaitGroup.Add(1)

	go func() {
		messageConsumer.Start()
		consumerWaitGroup.Done()
	}()

	waitForSignal()
	messageConsumer.Stop()
	consumerWaitGroup.Wait()
}

func (indexer *contentIndexer) handleMessage(msg consumer.Message) {

	tid := msg.Headers["X-Request-Id"]

	if strings.Contains(tid, syntheticRequestPrefix) {
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
