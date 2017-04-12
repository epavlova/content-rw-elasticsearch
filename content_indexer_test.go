package main

import (
	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gopkg.in/olivere/elastic.v2"
	"io/ioutil"
	"net/url"
	"testing"
	"time"
)

type esServiceMock struct {
	mock.Mock
}

func (service *esServiceMock) writeData(conceptType string, uuid string, payload interface{}) (*elastic.IndexResult, error) {
	args := service.Called(conceptType, uuid, payload)
	return args.Get(0).(*elastic.IndexResult), args.Error(1)
}
func (service *esServiceMock) readData(conceptType string, uuid string) (*elastic.GetResult, error) {
	args := service.Called()
	return args.Get(0).(*elastic.GetResult), args.Error(1)
}
func (service *esServiceMock) deleteData(conceptType string, uuid string) (*elastic.DeleteResult, error) {
	args := service.Called()
	return args.Get(0).(*elastic.DeleteResult), args.Error(1)
}

func (service *esServiceMock) setClient(client esClientI) {

}

func (service *esServiceMock) getClusterHealth() (*elastic.ClusterHealthResponse, error) {
	args := service.Called()
	return args.Get(0).(*elastic.ClusterHealthResponse), args.Error(1)
}

type elasticClientMock struct {
	mock.Mock
}

func (client elasticClientMock) ClusterHealth() *elastic.ClusterHealthService {
	args := client.Called()
	return args.Get(0).(*elastic.ClusterHealthService)
}

func (client elasticClientMock) Index() *elastic.IndexService {
	args := client.Called()
	return args.Get(0).(*elastic.IndexService)
}

func (client elasticClientMock) Get() *elastic.GetService {
	args := client.Called()
	return args.Get(0).(*elastic.GetService)
}

func (client elasticClientMock) Delete() *elastic.DeleteService {
	args := client.Called()
	return args.Get(0).(*elastic.DeleteService)
}

func (client elasticClientMock) PerformRequest(method, path string, params url.Values, body interface{}, ignoreErrors ...int) (*elastic.Response, error) {
	args := client.Called()
	return args.Get(0).(*elastic.Response), args.Error(1)
}

func TestStartClientError(t *testing.T) {
	assert := assert.New(t)

	accessConfig := esAccessConfig{
		accessKey:  "key",
		secretKey:  "secret",
		esEndpoint: "endpoint",
	}

	queueConfig := consumer.QueueConfig{
		Addrs:                []string{"address"},
		Group:                "group",
		Topic:                "topic",
		Queue:                "queue",
		ConcurrentProcessing: false,
	}

	newAmazonClient = func(config esAccessConfig) (esClientI, error) {
		return nil, elastic.ErrNoClient
	}

	indexer := contentIndexer{}

	indexer.start("index", "1984", accessConfig, queueConfig)

	assert.NotNil(indexer.esServiceInstance, "Elastic Service should be initialized")
	assert.Equal("index", (indexer.esServiceInstance).(*esService).indexName, "Wrong index")
	assert.Nil((indexer.esServiceInstance).(*esService).elasticClient, "Elastic client should not be initialized")
}

func TestStartClient(t *testing.T) {
	assert := assert.New(t)

	accessConfig := esAccessConfig{
		accessKey:  "key",
		secretKey:  "secret",
		esEndpoint: "endpoint",
	}

	queueConfig := consumer.QueueConfig{
		Addrs:                []string{"address"},
		Group:                "group",
		Topic:                "topic",
		Queue:                "queue",
		ConcurrentProcessing: false,
	}

	newAmazonClient = func(config esAccessConfig) (esClientI, error) {
		return &elasticClientMock{}, nil
	}

	indexer := contentIndexer{}

	indexer.start("index", "1985", accessConfig, queueConfig)

	time.Sleep(2000 * time.Millisecond)

	assert.NotNil(indexer.esServiceInstance, "Elastic Service should be initialized")
	assert.Equal("index", (indexer.esServiceInstance).(*esService).indexName, "Wrong index")
	assert.NotNil((indexer.esServiceInstance).(*esService).elasticClient, "Elastic client should be initialized")
}

func TestHandleMessage(t *testing.T) {
	assert := assert.New(t)

	inputJson, err := ioutil.ReadFile("exampleEnrichedContentModel.json")
	assert.NoError(err, "Unexpected error")

	serviceMock := &esServiceMock{}

	serviceMock.On("writeData", "FTCom", "aae9611e-f66c-4fe4-a6c6-2e2bdea69060", mock.Anything).Return(&elastic.IndexResult{}, nil)

	indexer := contentIndexer{esServiceInstance: serviceMock}
	indexer.handleMessage(consumer.Message{Body: string(inputJson)})

	serviceMock.AssertExpectations(t)

}
