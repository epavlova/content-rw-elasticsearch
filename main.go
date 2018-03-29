package main

import (
	"github.com/Financial-Times/go-logger"
	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	"github.com/jawher/mow.cli"
	"os"
	"os/signal"
	"syscall"
	"time"
	"github.com/Financial-Times/content-rw-elasticsearch/es"
	"net/http"
	health "github.com/Financial-Times/go-fthealth/v1_1"
	status "github.com/Financial-Times/service-status-go/httphandlers"
	"sync"
	"net"
	"github.com/Financial-Times/content-rw-elasticsearch/content"
)

const (
	appNameDefaultValue = "content-rw-elasticsearch"
	healthPath          = "/__health"
	healthDetailsPath   = "/__health-details"
)

func init() {
	logger.InitDefaultLogger(appNameDefaultValue)
}

func main() {
	app := cli.App("content-rw-elasticsearch", "Service for loading contents into elasticsearch")

	appSystemCode := app.String(cli.StringOpt{
		Name:   "app-system-code",
		Value:  "content-rw-elasticsearch",
		Desc:   "System Code of the application",
		EnvVar: "APP_SYSTEM_CODE",
	})

	appName := app.String(cli.StringOpt{
		Name:   "app-name",
		Value:  appNameDefaultValue,
		Desc:   "Application name",
		EnvVar: "APP_NAME",
	})

	port := app.String(cli.StringOpt{
		Name:   "port",
		Value:  "8080",
		Desc:   "Port to listen on",
		EnvVar: "APP_PORT",
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
	kafkaProxyAddress := app.String(cli.StringOpt{
		Name:   "kafka-proxy-address",
		Value:  "http://localhost:8080",
		Desc:   "Addresses used by the queue consumer to connect to the queue",
		EnvVar: "KAFKA_PROXY_ADDR",
	})
	kafkaConsumerGroup := app.String(cli.StringOpt{
		Name:   "kafka-consumer-group",
		Value:  "default-consumer-group",
		Desc:   "Group used to read the messages from the queue",
		EnvVar: "KAFKA_CONSUMER_GROUP",
	})
	kafkaTopic := app.String(cli.StringOpt{
		Name:   "kafka-topic",
		Value:  "CombinedPostPublicationEvents",
		Desc:   "The topic to read the meassages from",
		EnvVar: "KAFKA_TOPIC",
	})
	kafkaHeader := app.String(cli.StringOpt{
		Name:   "kafka-header",
		Value:  "kafka",
		Desc:   "The header identifying the queue to read the messages from",
		EnvVar: "KAFKA_HEADER",
	})
	kafkaConcurrentProcessing := app.Bool(cli.BoolOpt{
		Name:   "kafka-concurrent-processing",
		Value:  false,
		Desc:   "Whether the consumer uses concurrent processing for the messages",
		EnvVar: "KAFKA_CONCURRENT_PROCESSING",
	})

	queueConfig := consumer.QueueConfig{
		Addrs:                []string{*kafkaProxyAddress},
		Group:                *kafkaConsumerGroup,
		Topic:                *kafkaTopic,
		Queue:                *kafkaHeader,
		ConcurrentProcessing: *kafkaConcurrentProcessing,
	}

	logger.Info("[Startup] Application is starting")

	app.Action = func() {

		accessConfig := es.AccessConfig{
			AccessKey: *accessKey,
			SecretKey: *secretKey,
			Endpoint:  *esEndpoint,
		}

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

		service := es.NewService(*indexName)
		mapper := es.NewContentMapper()
		var wg sync.WaitGroup
		indexer := content.NewContentIndexer(service, mapper, client, queueConfig, &wg, es.NewClient)

		indexer.Start(*appSystemCode, *appName, *indexName, *port, accessConfig)
		serveAdminEndpoints(service, *appSystemCode, *appName, *port, queueConfig)
		indexer.Stop()
		wg.Wait()
	}
	err := app.Run(os.Args)
	if err != nil {
		logger.WithError(err).WithTime(time.Now()).Fatal("App could not start")
		return
	}
	logger.Info("[Shutdown] Shutdown complete")
}

func serveAdminEndpoints(esService es.ServiceI, appSystemCode string, appName string, port string, queueConfig consumer.QueueConfig) {
	healthService := newHealthService(&queueConfig, esService)

	serveMux := http.NewServeMux()

	hc := health.HealthCheck{SystemCode: appSystemCode, Name: appName, Description: "Content Read Writer for Elasticsearch", Checks: healthService.checks}
	serveMux.HandleFunc(healthPath, health.Handler(hc))
	serveMux.HandleFunc(healthDetailsPath, healthService.HealthDetails)
	serveMux.HandleFunc(status.GTGPath, status.NewGoodToGoHandler(healthService.gtgCheck))
	serveMux.HandleFunc(status.BuildInfoPath, status.BuildInfoHandler)

	server := &http.Server{Addr: ":" + port, Handler: serveMux}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		if err := server.ListenAndServe(); err != nil {
			logger.WithError(err).Error("HTTP server is closing")
		}
		wg.Done()
	}()

	waitForSignal()
	logger.Info("[Shutdown] Application is shutting down")

	if err := server.Close(); err != nil {
		logger.WithError(err).Error("Unable to stop http server")
	}
	wg.Wait()
}

func waitForSignal() {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
}
