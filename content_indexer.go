package main

import (
	"encoding/json"
	"github.com/Financial-Times/go-logger"
	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	"github.com/dchest/uniuri"
	"net/http"
	"strings"
	"sync"
	"time"
	"github.com/Financial-Times/content-rw-elasticsearch/es"
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
	esServiceInstance es.ServiceI
	messageConsumer   consumer.MessageConsumer
	Client            *http.Client
	wg                sync.WaitGroup
	mu                sync.Mutex
}

func NewContentIndexer(service es.ServiceI, client *http.Client) *contentIndexer {
	return &contentIndexer{esServiceInstance: service, Client: client}
}

func (indexer *contentIndexer) start(appSystemCode string, appName string, indexName string, port string, accessConfig es.AccessConfig, queueConfig consumer.QueueConfig) {
	channel := make(chan es.ClientI)
	go func() {
		defer close(channel)
		for {
			ec, err := es.NewAmazonClient(accessConfig)
			if err == nil {
				logger.Info("Connected to Elasticsearch")
				channel <- ec
				return
			}
			logger.Error("Could not connect to Elasticsearch")
			time.Sleep(time.Minute)
		}
	}()

	indexer.wg.Add(1)
	go func() {
		defer indexer.wg.Done()
		for ec := range channel {
			indexer.esServiceInstance.SetClient(ec)
			indexer.startMessageConsumer(queueConfig)
		}
	}()
}

func (indexer *contentIndexer) stop() {
	indexer.mu.Lock()
	if indexer.messageConsumer != nil {
		indexer.messageConsumer.Stop()
	}
	indexer.mu.Unlock()

}

func (indexer *contentIndexer) startMessageConsumer(config consumer.QueueConfig) {
	indexer.mu.Lock()
	indexer.messageConsumer = consumer.NewConsumer(config, indexer.handleMessage, indexer.Client)
	indexer.mu.Unlock()

	//this is a blocking method
	indexer.messageConsumer.Start()
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
		_, err = indexer.esServiceInstance.DeleteData(contentTypeMap[contentType].collection, uuid)
		if err != nil {
			logger.WithTransactionID(tid).WithUUID(uuid).WithError(err).Error("Failed to index content")
			return
		}
		logger.WithMonitoringEvent("ContentDeleteElasticsearch", tid, "").WithUUID(uuid).Info("Successfully deleted")
	} else {
		payload := convertToESContentModel(combinedPostPublicationEvent, contentType, tid)

		_, err = indexer.esServiceInstance.WriteData(contentTypeMap[contentType].collection, uuid, payload)
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
