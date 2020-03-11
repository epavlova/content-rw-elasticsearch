package service

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Financial-Times/content-rw-elasticsearch/content"
	"github.com/Financial-Times/content-rw-elasticsearch/es"
	"github.com/Financial-Times/content-rw-elasticsearch/service/concept"
	"github.com/Financial-Times/go-logger"
	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	"github.com/dchest/uniuri"
	"github.com/stretchr/stew/slice"
)

const (
	syntheticRequestPrefix   = "SYNTHETIC-REQ-MON"
	transactionIDHeader      = "X-Request-Id"
	blogsAuthority           = "http://api.ft.com/system/FT-LABS-WP"
	articleAuthority         = "http://api.ft.com/system/FTCOM-METHODE"
	videoAuthority           = "http://api.ft.com/system/NEXT-VIDEO-EDITOR"
	oldSparkAuthoriy         = "http://api.ft.com/system/cct"
	sparkAuthoriy            = "http://api.ft.com/system/spark"
	originHeader             = "Origin-System-Id"
	oldSparkOrigin           = "cct"
	sparkOrigin              = "spark"
	methodeOrigin            = "methode-web-pub"
	wordpressOrigin          = "wordpress"
	videoOrigin              = "next-video-editor"
	pacOrigin                = "http://cmdb.ft.com/systems/pac"
	contentTypeHeader        = "Content-Type"
	audioContentTypeHeader   = "ft-upp-audio"
	articleContentTypeHeader = "ft-upp-article"
)

// Empty type added for older content. Placeholders - which are subject of exclusion - have type Content.
var allowedTypes = []string{"Article", "Video", "MediaResource", "Audio", ""}

type MessageHandler struct {
	esService         es.ServiceI
	messageConsumer   consumer.MessageConsumer
	ConceptGetter     concept.ConceptGetter
	connectToESClient func(config es.AccessConfig, c *http.Client) (es.ClientI, error)
	baseApiUrl        string
	wg                sync.WaitGroup
	mu                sync.Mutex
}

func NewMessageHandler(service es.ServiceI, conceptGetter concept.ConceptGetter, client *http.Client, queueConfig consumer.QueueConfig, wg *sync.WaitGroup, connectToClient func(config es.AccessConfig, c *http.Client) (es.ClientI, error)) *MessageHandler {
	indexer := &MessageHandler{esService: service, ConceptGetter: conceptGetter, connectToESClient: connectToClient, wg: *wg}
	indexer.messageConsumer = consumer.NewConsumer(queueConfig, indexer.handleMessage, client)
	return indexer
}

func (handler *MessageHandler) Start(baseApiUrl string, accessConfig es.AccessConfig, httpClient *http.Client) {
	handler.baseApiUrl = baseApiUrl
	channel := make(chan es.ClientI)
	go func() {
		defer close(channel)
		for {
			ec, err := handler.connectToESClient(accessConfig, httpClient)
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
		defer handler.wg.Done()
		for ec := range channel {
			handler.mu.Lock()
			handler.wg.Add(1)
			handler.mu.Unlock()
			handler.esService.SetClient(ec)
			handler.startMessageConsumer()
		}
	}()
}

func (handler *MessageHandler) Stop() {
	handler.mu.Lock()
	if handler.messageConsumer != nil {
		handler.messageConsumer.Stop()
	}
	handler.mu.Unlock()

}

func (handler *MessageHandler) startMessageConsumer() {
	//this is a blocking method
	handler.messageConsumer.Start()
}

func (handler *MessageHandler) handleMessage(msg consumer.Message) {
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

	if combinedPostPublicationEvent.Content.BodyXML != "" && combinedPostPublicationEvent.Content.Body == "" {
		combinedPostPublicationEvent.Content.Body = combinedPostPublicationEvent.Content.BodyXML
		combinedPostPublicationEvent.Content.BodyXML = ""
	}

	if !slice.ContainsString(allowedTypes, combinedPostPublicationEvent.Content.Type) {
		logger.WithTransactionID(tid).Infof("Ignoring message of type %s", combinedPostPublicationEvent.Content.Type)
		return
	}

	uuid := combinedPostPublicationEvent.UUID
	logger.WithTransactionID(tid).WithUUID(uuid).Info("Processing combined post publication event")

	contentType := extractContentTypeFromMsg(msg, combinedPostPublicationEvent)
	if contentType == "" && msg.Headers[originHeader] != pacOrigin {
		logger.WithTransactionID(tid).WithUUID(uuid).Error("Failed to index content. Could not infer type of content")
		return
	}

	if combinedPostPublicationEvent.MarkedDeleted == "true" {
		_, err = handler.esService.DeleteData(ContentTypeMap[contentType].Collection, uuid)
		if err != nil {
			logger.WithTransactionID(tid).WithUUID(uuid).WithError(err).Error("Failed to delete indexed content")
			return
		}
		logger.WithMonitoringEvent("ContentDeleteElasticsearch", tid, contentType).WithUUID(uuid).Info("Successfully deleted")
		return
	}

	if combinedPostPublicationEvent.Content.UUID == "" || contentType == "" {
		logger.WithTransactionID(tid).WithUUID(combinedPostPublicationEvent.UUID).Info("Ignoring message with no content")
		return
	}

	payload := handler.ToIndexModel(combinedPostPublicationEvent, contentType, tid)

	_, err = handler.esService.WriteData(ContentTypeMap[contentType].Collection, uuid, payload)
	if err != nil {
		logger.WithTransactionID(tid).WithUUID(uuid).WithError(err).Error("Failed to index content")
		return
	}
	logger.WithMonitoringEvent("ContentWriteElasticsearch", tid, contentType).WithUUID(uuid).Info("Successfully saved")

}

func extractContentTypeFromMsg(msg consumer.Message, event content.EnrichedContent) string {

	typeHeader := msg.Headers[contentTypeHeader]
	if strings.Contains(typeHeader, audioContentTypeHeader) {
		return AudioType
	}
	if strings.Contains(typeHeader, articleContentTypeHeader) {
		return ArticleType
	}
	var contentType string

	for _, identifier := range event.Content.Identifiers {
		if strings.HasPrefix(identifier.Authority, blogsAuthority) {
			contentType = BlogType
		} else if strings.HasPrefix(identifier.Authority, articleAuthority) {
			contentType = ArticleType
		} else if strings.HasPrefix(identifier.Authority, videoAuthority) {
			contentType = VideoType
		} else if strings.HasPrefix(identifier.Authority, oldSparkAuthoriy) {
			contentType = ArticleType
		} else if strings.HasPrefix(identifier.Authority, sparkAuthoriy) {
			contentType = ArticleType
		}
	}
	if contentType != "" {
		return contentType
	}

	msgOrigin := msg.Headers[originHeader]
	originMap := map[string]string{
		methodeOrigin:   ArticleType,
		oldSparkOrigin:  ArticleType,
		sparkOrigin:     ArticleType,
		wordpressOrigin: BlogType,
		videoOrigin:     VideoType,
	}
	for origin, contentType := range originMap {
		if strings.Contains(msgOrigin, origin) {
			return contentType
		}
	}

	return ""
}
