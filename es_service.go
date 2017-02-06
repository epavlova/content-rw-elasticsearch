package main

import (
	"gopkg.in/olivere/elastic.v2"
)

type esService struct {
	elasticClient *elastic.Client
	indexName     string
}

type esServiceI interface {
	writeData(conceptType string, uuid string, payload interface{}) (*elastic.IndexResult, error)
	readData(conceptType string, uuid string) (*elastic.GetResult, error)
	deleteData(conceptType string, uuid string) (*elastic.DeleteResult, error)
}

func newEsService(client *elastic.Client, indexName string) *esService {
	return &esService{elasticClient: client, indexName: indexName}
}

func (service esService) writeData(conceptType string, uuid string, payload interface{}) (*elastic.IndexResult, error) {
	return service.elasticClient.Index().
		Index(service.indexName).
		Type(conceptType).
		Id(uuid).
		BodyJson(payload).
		Do()
}

func (service esService) readData(conceptType string, uuid string) (*elastic.GetResult, error) {
	resp, err := service.elasticClient.Get().
		Index(service.indexName).
		Type(conceptType).
		Id(uuid).
		IgnoreErrorsOnGeneratedFields(false).
		Do()

	return resp, err

}

func (service esService) deleteData(conceptType string, uuid string) (*elastic.DeleteResult, error) {
	resp, err := service.elasticClient.Delete().
		Index(service.indexName).
		Type(conceptType).
		Id(uuid).
		Do()

	return resp, err

}