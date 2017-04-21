package main

import (
	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	log "github.com/Sirupsen/logrus"
	"github.com/jawher/mow.cli"
	"os"
)

func main() {
	app := cli.App("content-rw-es", "Service for loading contents into elasticsearch")

	appSystemCode := app.String(cli.StringOpt{
		Name:   "app-system-code",
		Value:  "content-rw-elasticsearch",
		Desc:   "System Code of the application",
		EnvVar: "APP_SYSTEM_CODE",
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

	accessConfig := esAccessConfig{
		accessKey:  *accessKey,
		secretKey:  *secretKey,
		esEndpoint: *esEndpoint,
	}

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

	log.SetLevel(log.InfoLevel)
	log.Infof("[Startup] Content RW Elasticsearch is starting ")

	app.Action = func() {
		indexer := contentIndexer{}
		indexer.start(*appSystemCode, *indexName, *port, accessConfig, queueConfig)
		waitForSignal()
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Errorf("App could not start, error=[%s]\n", err)
		return
	}
}
