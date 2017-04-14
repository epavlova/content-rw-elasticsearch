package main

import (
	"errors"
	"gopkg.in/olivere/elastic.v2"
	"sync"
)

type esService struct {
	sync.RWMutex
	elasticClient esClientI
	indexName     string
}

type esServiceI interface {
	esHealthServiceI
	setClient(client esClientI)
	writeData(conceptType string, uuid string, payload interface{}) (*elastic.IndexResult, error)
	deleteData(conceptType string, uuid string) (*elastic.DeleteResult, error)
}

type esHealthServiceI interface {
	getClusterHealth() (*elastic.ClusterHealthResponse, error)
}

func newEsService(indexName string) esServiceI {
	return &esService{indexName: indexName}
}

func (service *esService) getClusterHealth() (*elastic.ClusterHealthResponse, error) {
	if service.elasticClient == nil {
		return nil, errors.New("Client could not be created, please check the application parameters/env variables, and restart the service.")
	}

	return service.elasticClient.ClusterHealth().Do()
}

func (service *esService) setClient(client esClientI) {
	service.Lock()
	defer service.Unlock()
	service.elasticClient = client
}

func (service *esService) writeData(conceptType string, uuid string, payload interface{}) (*elastic.IndexResult, error) {
	return service.elasticClient.Index().
		Index(service.indexName).
		Type(conceptType).
		Id(uuid).
		BodyJson(payload).
		Do()
}

func (service *esService) deleteData(conceptType string, uuid string) (*elastic.DeleteResult, error) {
	return service.elasticClient.Delete().
		Index(service.indexName).
		Type(conceptType).
		Id(uuid).
		Do()
}
