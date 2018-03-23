package main

import (
	"encoding/json"
	health "github.com/Financial-Times/go-fthealth/v1_1"
	"github.com/Financial-Times/go-logger"
	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	status "github.com/Financial-Times/service-status-go/httphandlers"
	"github.com/dchest/uniuri"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	healthPath             = "/__health"
	healthDetailsPath      = "/__health-details"
	syntheticRequestPrefix = "SYNTHETIC-REQ-MON"
	transactionIDHeader    = "X-Request-Id"
	blogsAuthority         = "http://api.ft.com/system/FT-LABS-WP"
	articleAuthority       = "http://api.ft.com/system/FTCOM-METHODE"
	videoAuthority         = "http://api.ft.com/system/NEXT-VIDEO-EDITOR"
	originHeader           = "Origin-System-Id"
	methodeOrigin          = "methode-web-pub"
	wordpressOrigin        = "wordpress"
	videoOrigin            = "next-video-editor"
	blogType               = "blog"
	articleType            = "article"
	videoType              = "video"
)

// Empty type added for older content. Placeholders - which are subject of exclusion - have type Content.
var allowedTypes = []string{"Article", "Video", "MediaResource", ""}

type contentIndexer struct {
	esServiceInstance esServiceI
	server            *http.Server
	messageConsumer   consumer.MessageConsumer
	wg                sync.WaitGroup
	mu                sync.Mutex
}

func (indexer *contentIndexer) start(appSystemCode string, appName string, indexName string, port string, accessConfig esAccessConfig, queueConfig consumer.QueueConfig) {
	channel := make(chan esClientI)
	go func() {
		defer close(channel)
		for {
			ec, err := newAmazonClient(accessConfig)
			if err == nil {
				logger.Info("Connected to Elasticsearch")
				channel <- ec
				return
			}
			logger.Error("Could not connect to Elasticsearch")
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

	indexer.serveAdminEndpoints(appSystemCode, appName, port, queueConfig)
}

func (indexer *contentIndexer) stop() {
	go func() {
		indexer.mu.Lock()
		if indexer.messageConsumer != nil {
			indexer.messageConsumer.Stop()
		}
		indexer.mu.Unlock()
	}()
	if err := indexer.server.Close(); err != nil {
		logger.WithError(err).Error("Unable to stop http server")
	}
	indexer.wg.Wait()
}

func (indexer *contentIndexer) serveAdminEndpoints(appSystemCode string, appName string, port string, queueConfig consumer.QueueConfig) {
	healthService := newHealthService(&queueConfig, indexer.esServiceInstance)

	serveMux := http.NewServeMux()

	hc := health.HealthCheck{SystemCode: appSystemCode, Name: appName, Description: "Content Read Writer for Elasticsearch", Checks: healthService.checks}
	serveMux.HandleFunc(healthPath, health.Handler(hc))
	serveMux.HandleFunc(healthDetailsPath, healthService.HealthDetails)
	serveMux.HandleFunc(status.GTGPath, status.NewGoodToGoHandler(healthService.gtgCheck))
	serveMux.HandleFunc(status.BuildInfoPath, status.BuildInfoHandler)

	indexer.server = &http.Server{Addr: ":" + port, Handler: serveMux}

	indexer.wg.Add(1)
	go func() {
		if err := indexer.server.ListenAndServe(); err != nil {
			logger.WithError(err).Error("HTTP server is closing")
		}
		indexer.wg.Done()
	}()
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
	indexer.mu.Lock()
	indexer.messageConsumer = consumer.NewConsumer(config, indexer.handleMessage, client)
	indexer.mu.Unlock()

	indexer.wg.Add(1)

	indexer.messageConsumer.Start()
	indexer.wg.Done()
}

func (indexer *contentIndexer) handleMessage(msg consumer.Message) {

	tid := msg.Headers[transactionIDHeader]
	if tid == "" {
		tid = "tid_" + uniuri.NewLen(10) + "_content-rw-elasticsearch"
		logger.WithTransactionID(tid).Info("Generated tid")
	}

	if strings.Contains(tid, syntheticRequestPrefix) {
		logger.WithTransactionID(tid).Info("Ignoring synthetic message")
		return
	}

	var combinedPostPublicationEvent enrichedContentModel
	err := json.Unmarshal([]byte(msg.Body), &combinedPostPublicationEvent)
	if err != nil {
		logger.WithTransactionID(tid).WithError(err).Error("Cannot unmarshal message body")
		return
	}

	if !contains(allowedTypes, combinedPostPublicationEvent.Content.Type) {
		logger.WithTransactionID(tid).Infof("Ignoring message of type %s", combinedPostPublicationEvent.Content.Type)
		return
	}

	if combinedPostPublicationEvent.Content.UUID == "" {
		logger.WithTransactionID(tid).WithUUID(combinedPostPublicationEvent.UUID).Info("Ignoring message with no content for UUID")
		return
	}

	uuid := combinedPostPublicationEvent.UUID
	logger.WithTransactionID(tid).WithUUID(uuid).Info("Processing combined post publication event")

	var contentType string
	for _, identifier := range combinedPostPublicationEvent.Content.Identifiers {
		if strings.HasPrefix(identifier.Authority, blogsAuthority) {
			contentType = blogType
		} else if strings.HasPrefix(identifier.Authority, articleAuthority) {
			contentType = articleType
		} else if strings.HasPrefix(identifier.Authority, videoAuthority) {
			contentType = videoType
		}
	}

	if contentType == "" {
		origin := msg.Headers[originHeader]
		if strings.Contains(origin, methodeOrigin) {
			contentType = articleType
		} else if strings.Contains(origin, wordpressOrigin) {
			contentType = blogType
		} else if strings.Contains(origin, videoOrigin) {
			contentType = videoType
		} else {
			logger.WithTransactionID(tid).WithUUID(uuid).WithError(err).Error("Failed to index content. Could not infer type of content")
			return
		}
	}

	if combinedPostPublicationEvent.Content.MarkedDeleted {
		_, err = indexer.esServiceInstance.deleteData(contentTypeMap[contentType].collection, uuid)
		if err != nil {
			logger.WithTransactionID(tid).WithUUID(uuid).WithError(err).Error("Failed to index content")
			return
		}
		logger.WithMonitoringEvent("ContentDeleteElasticsearch", tid, "").WithUUID(uuid).Info("Successfully deleted")
	} else {
		payload := convertToESContentModel(combinedPostPublicationEvent, contentType, tid)

		_, err = indexer.esServiceInstance.writeData(contentTypeMap[contentType].collection, uuid, payload)
		if err != nil {
			logger.WithTransactionID(tid).WithUUID(uuid).WithError(err).Error("Failed to index content")
			return
		}
		logger.WithMonitoringEvent("ContentWriteElasticsearch", tid, "").WithUUID(uuid).Info("Successfully saved")
	}
}

func contains(list []string, elem string) bool {
	for _, a := range list {
		if a == elem {
			return true
		}
	}
	return false
}
