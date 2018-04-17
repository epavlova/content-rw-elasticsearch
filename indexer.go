package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Financial-Times/content-rw-elasticsearch/es"
	"github.com/Financial-Times/go-logger"
	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	"github.com/dchest/uniuri"
	"github.com/stretchr/stew/slice"
	"github.com/Financial-Times/content-rw-elasticsearch/content"
)

const (
	syntheticRequestPrefix = "SYNTHETIC-REQ-MON"
	transactionIDHeader    = "X-Request-Id"
	blogsAuthority         = "http://api.ft.com/system/FT-LABS-WP"
	articleAuthority       = "http://api.ft.com/system/FTCOM-METHODE"
	videoAuthority         = "http://api.ft.com/system/NEXT-VIDEO-EDITOR"
	originHeader           = "Origin-System-Id"
	methodeOrigin          = "methode-web-pub"
	wordpressOrigin        = "wordpress"
	videoOrigin            = "next-video-editor"
)

// Empty type added for older content. Placeholders - which are subject of exclusion - have type Content.
var allowedTypes = []string{"Article", "Video", "MediaResource", ""}

type Indexer struct {
	esService         es.ServiceI
	messageConsumer   consumer.MessageConsumer
	ConceptGetter     ConceptGetter
	connectToESClient func(config es.AccessConfig, c *http.Client) (es.ClientI, error)
	wg                sync.WaitGroup
	mu                sync.Mutex
}

func NewIndexer(service es.ServiceI, conceptGetter ConceptGetter, client *http.Client, queueConfig consumer.QueueConfig, wg *sync.WaitGroup, connectToClient func(config es.AccessConfig, c *http.Client) (es.ClientI, error)) *Indexer {
	indexer := &Indexer{esService: service, ConceptGetter: conceptGetter, connectToESClient: connectToClient, wg: *wg}
	indexer.messageConsumer = consumer.NewConsumer(queueConfig, indexer.handleMessage, client)
	return indexer
}

func (indexer *Indexer) Start(appSystemCode string, appName string, indexName string, port string, accessConfig es.AccessConfig, httpClient *http.Client) {
	channel := make(chan es.ClientI)
	go func() {
		defer close(channel)
		for {
			ec, err := indexer.connectToESClient(accessConfig, httpClient)
			if err == nil {
				logger.Info("Connected to Elasticsearch")
				channel <- ec
				return
			}
			logger.Error("Could not connect to Elasticsearch")
			time.Sleep(time.Minute)
		}
	}()

	go func() {
		defer indexer.wg.Done()
		for ec := range channel {
			indexer.mu.Lock()
			indexer.wg.Add(1)
			indexer.mu.Unlock()
			indexer.esService.SetClient(ec)
			indexer.startMessageConsumer()
		}
	}()
}

func (indexer *Indexer) Stop() {
	indexer.mu.Lock()
	if indexer.messageConsumer != nil {
		indexer.messageConsumer.Stop()
	}
	indexer.mu.Unlock()

}

func (indexer *Indexer) startMessageConsumer() {
	//this is a blocking method
	indexer.messageConsumer.Start()
}

func (indexer *Indexer) handleMessage(msg consumer.Message) {
	tid := msg.Headers[transactionIDHeader]
	if tid == "" {
		tid = "tid_" + uniuri.NewLen(10) + "_content-rw-elasticsearch"
		logger.WithTransactionID(tid).Info("Generated tid")
	}

	if strings.Contains(tid, syntheticRequestPrefix) {
		logger.WithTransactionID(tid).Info("Ignoring synthetic message")
		return
	}

	var combinedPostPublicationEvent content.EnrichedContent
	err := json.Unmarshal([]byte(msg.Body), &combinedPostPublicationEvent)
	if err != nil {
		logger.WithTransactionID(tid).WithError(err).Error("Cannot unmarshal message body")
		return
	}

	if !slice.ContainsString(allowedTypes, combinedPostPublicationEvent.Content.Type) {
		logger.WithTransactionID(tid).Infof("Ignoring message of type %s", combinedPostPublicationEvent.Content.Type)
		return
	}

	uuid := combinedPostPublicationEvent.UUID
	logger.WithTransactionID(tid).WithUUID(uuid).Info("Processing combined post publication event")

	var contentType string
	for _, identifier := range combinedPostPublicationEvent.Content.Identifiers {
		if strings.HasPrefix(identifier.Authority, blogsAuthority) {
			contentType = BlogType
		} else if strings.HasPrefix(identifier.Authority, articleAuthority) {
			contentType = ArticleType
		} else if strings.HasPrefix(identifier.Authority, videoAuthority) {
			contentType = VideoType
		}
	}

	if contentType == "" {
		origin := msg.Headers[originHeader]
		if strings.Contains(origin, methodeOrigin) {
			contentType = ArticleType
		} else if strings.Contains(origin, wordpressOrigin) {
			contentType = BlogType
		} else if strings.Contains(origin, videoOrigin) {
			contentType = VideoType
		} else {
			logger.WithTransactionID(tid).WithUUID(uuid).WithError(err).Error("Failed to index content. Could not infer type of content")
			return
		}
	}

	if combinedPostPublicationEvent.MarkedDeleted == "true" {
		_, err = indexer.esService.DeleteData(ContentTypeMap[contentType].Collection, uuid)
		if err != nil {
			logger.WithTransactionID(tid).WithUUID(uuid).WithError(err).Error("Failed to delete indexed content")
			return
		}
		logger.WithMonitoringEvent("ContentDeleteElasticsearch", tid, "").WithUUID(uuid).Info("Successfully deleted")
		return
	}

	if combinedPostPublicationEvent.Content.UUID == "" {
		logger.WithTransactionID(tid).WithUUID(combinedPostPublicationEvent.UUID).Info("Ignoring message with no content")
		return
	}

	payload := indexer.ToIndexModel(combinedPostPublicationEvent, contentType, tid)

	_, err = indexer.esService.WriteData(ContentTypeMap[contentType].Collection, uuid, payload)
	if err != nil {
		logger.WithTransactionID(tid).WithUUID(uuid).WithError(err).Error("Failed to index content")
		return
	}
	logger.WithMonitoringEvent("ContentWriteElasticsearch", tid, "").WithUUID(uuid).Info("Successfully saved")

}
