package main

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Financial-Times/content-rw-elasticsearch/es"
	logTest "github.com/Financial-Times/go-logger/test"
	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gopkg.in/olivere/elastic.v2"
)

type esServiceMock struct {
	mock.Mock
}

func (*esServiceMock) GetSchemaHealth() (string, error) {
	panic("implement me")
}

func (service *esServiceMock) WriteData(conceptType string, uuid string, payload interface{}) (*elastic.IndexResult, error) {
	args := service.Called(conceptType, uuid, payload)
	return args.Get(0).(*elastic.IndexResult), args.Error(1)
}

func (service *esServiceMock) DeleteData(conceptType string, uuid string) (*elastic.DeleteResult, error) {
	args := service.Called(conceptType, uuid)
	return args.Get(0).(*elastic.DeleteResult), args.Error(1)
}

func (service *esServiceMock) SetClient(client es.ClientI) {

}

func (service *esServiceMock) GetClusterHealth() (*elastic.ClusterHealthResponse, error) {
	args := service.Called()
	return args.Get(0).(*elastic.ClusterHealthResponse), args.Error(1)
}

type elasticClientMock struct {
	mock.Mock
}

func (client elasticClientMock) IndexGet() *elastic.IndicesGetService {
	args := client.Called()
	return args.Get(0).(*elastic.IndicesGetService)
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

type concordanceApiMock struct {
	mock.Mock
}

func (m *concordanceApiMock) GetConcepts(tid string, ids []string) (map[string]*ConceptModel, error) {
	args := m.Called(tid, ids)
	return args.Get(0).(map[string]*ConceptModel), args.Error(1)
}

func TestStartClient(t *testing.T) {
	assert := assert.New(t)

	accessConfig := es.AccessConfig{
		AccessKey: "key",
		SecretKey: "secret",
		Endpoint:  "endpoint",
	}

	queueConfig := consumer.QueueConfig{
		Addrs:                []string{"address"},
		Group:                "group",
		Topic:                "topic",
		Queue:                "queue",
		ConcurrentProcessing: false,
	}

	var NewClient = func(config es.AccessConfig, c *http.Client) (es.ClientI, error) {
		return &elasticClientMock{}, nil
	}
	concordanceApiMock := new(concordanceApiMock)

	var wg sync.WaitGroup
	indexer := NewIndexer(es.NewService("index"), concordanceApiMock, http.DefaultClient, queueConfig, &wg, NewClient)

	indexer.Start("app", "name", "index", "1984", accessConfig, http.DefaultClient)
	defer indexer.Stop()

	time.Sleep(100 * time.Millisecond)

	assert.NotNil(indexer.esService, "Elastic Service should be initialized")
	assert.Equal("index", (indexer.esService).(*es.Service).IndexName, "Wrong index")
	(indexer.esService).(*es.Service).Lock()
	assert.NotNil((indexer.esService).(*es.Service).ElasticClient, "Elastic client should be initialized")
	(indexer.esService).(*es.Service).Unlock()
}

func TestStartClientError(t *testing.T) {
	assert := assert.New(t)

	hook := logTest.NewTestHook("content-rw-elasticsearch")

	accessConfig := es.AccessConfig{
		AccessKey: "key",
		SecretKey: "secret",
		Endpoint:  "endpoint",
	}

	queueConfig := consumer.QueueConfig{
		Addrs:                []string{"address"},
		Group:                "group",
		Topic:                "topic",
		Queue:                "queue",
		ConcurrentProcessing: false,
	}

	var NewClient = func(config es.AccessConfig, c *http.Client) (es.ClientI, error) {
		return nil, elastic.ErrNoClient
	}

	concordanceApiMock := new(concordanceApiMock)

	var wg sync.WaitGroup
	indexer := NewIndexer(es.NewService("index"), concordanceApiMock, http.DefaultClient, queueConfig, &wg, NewClient)

	indexer.Start("app", "name", "index", "1984", accessConfig, http.DefaultClient)
	defer indexer.Stop()

	time.Sleep(100 * time.Millisecond)

	require.NotNil(t, hook.LastEntry())
	assert.Equal("error", hook.LastEntry().Level.String(), "Wrong log")
	assert.NotNil(indexer.esService, "Elastic Service should be initialized")
	assert.Equal("index", (indexer.esService).(*es.Service).IndexName, "Wrong index")
	assert.Nil((indexer.esService).(*es.Service).ElasticClient, "Elastic client should not be initialized")
}

func TestHandleWriteMessage(t *testing.T) {
	assert := assert.New(t)

	inputJSON, err := ioutil.ReadFile("testdata/exampleEnrichedContentModel.json")
	assert.NoError(err, "Unexpected error")

	serviceMock := &esServiceMock{}
	serviceMock.On("WriteData", "FTCom", "aae9611e-f66c-4fe4-a6c6-2e2bdea69060", mock.Anything).Return(&elastic.IndexResult{}, nil)
	concordanceApiMock := new(concordanceApiMock)
	concordanceApiMock.On("GetConcepts", mock.AnythingOfType("string"), mock.AnythingOfType("[]string")).Return(map[string]*ConceptModel{}, nil)

	indexer := Indexer{esService: serviceMock, ConceptGetter: concordanceApiMock}
	indexer.handleMessage(consumer.Message{Body: string(inputJSON)})

	serviceMock.AssertExpectations(t)
	concordanceApiMock.AssertExpectations(t)
}

func TestHandleWriteMessageBlog(t *testing.T) {
	assert := assert.New(t)

	inputJSON, err := ioutil.ReadFile("testdata/exampleEnrichedContentModel.json")
	assert.NoError(err, "Unexpected error")
	input := strings.Replace(string(inputJSON), "FTCOM-METHODE", "FT-LABS-WP1234", 1)

	serviceMock := &esServiceMock{}
	serviceMock.On("WriteData", "FTBlogs", "aae9611e-f66c-4fe4-a6c6-2e2bdea69060", mock.Anything).Return(&elastic.IndexResult{}, nil)
	concordanceApiMock := new(concordanceApiMock)
	concordanceApiMock.On("GetConcepts", mock.AnythingOfType("string"), mock.AnythingOfType("[]string")).Return(map[string]*ConceptModel{}, nil)

	indexer := Indexer{esService: serviceMock, ConceptGetter: concordanceApiMock}
	indexer.handleMessage(consumer.Message{Body: input})

	serviceMock.AssertExpectations(t)
	concordanceApiMock.AssertExpectations(t)
}

func TestHandleWriteMessageBlogWithHeader(t *testing.T) {
	assert := assert.New(t)

	inputJSON, err := ioutil.ReadFile("testdata/exampleEnrichedContentModel.json")
	assert.NoError(err, "Unexpected error")
	input := strings.Replace(string(inputJSON), "FTCOM-METHODE", "invalid", 1)

	serviceMock := &esServiceMock{}
	serviceMock.On("WriteData", "FTBlogs", "aae9611e-f66c-4fe4-a6c6-2e2bdea69060", mock.Anything).Return(&elastic.IndexResult{}, nil)
	concordanceApiMock := new(concordanceApiMock)
	concordanceApiMock.On("GetConcepts", mock.AnythingOfType("string"), mock.AnythingOfType("[]string")).Return(map[string]*ConceptModel{}, nil)

	indexer := Indexer{esService: serviceMock, ConceptGetter: concordanceApiMock}
	indexer.handleMessage(consumer.Message{Body: input, Headers: map[string]string{"Origin-System-Id": "wordpress"}})

	serviceMock.AssertExpectations(t)
	concordanceApiMock.AssertExpectations(t)
}

func TestHandleWriteMessageVideo(t *testing.T) {
	assert := assert.New(t)

	inputJSON, err := ioutil.ReadFile("testdata/exampleEnrichedContentModel.json")
	assert.NoError(err, "Unexpected error")
	input := strings.Replace(string(inputJSON), "FTCOM-METHODE", "NEXT-VIDEO-EDITOR", 1)

	serviceMock := &esServiceMock{}
	serviceMock.On("WriteData", "FTVideos", "aae9611e-f66c-4fe4-a6c6-2e2bdea69060", mock.Anything).Return(&elastic.IndexResult{}, nil)
	concordanceApiMock := new(concordanceApiMock)
	concordanceApiMock.On("GetConcepts", mock.AnythingOfType("string"), mock.AnythingOfType("[]string")).Return(map[string]*ConceptModel{}, nil)

	indexer := Indexer{esService: serviceMock, ConceptGetter: concordanceApiMock}
	indexer.handleMessage(consumer.Message{Body: input})

	serviceMock.AssertExpectations(t)
	concordanceApiMock.AssertExpectations(t)
}

func TestHandleWriteMessageUnknownType(t *testing.T) {
	assert := assert.New(t)

	inputJSON, err := ioutil.ReadFile("testdata/exampleEnrichedContentModel.json")
	assert.NoError(err, "Unexpected error")
	input := strings.Replace(string(inputJSON), `"Article"`, `"Content"`, 1)

	serviceMock := &esServiceMock{}

	indexer := Indexer{esService: serviceMock}
	indexer.handleMessage(consumer.Message{Body: input})

	serviceMock.AssertNotCalled(t, "WriteData", mock.Anything, "aae9611e-f66c-4fe4-a6c6-2e2bdea69060", mock.Anything)
	serviceMock.AssertNotCalled(t, "DeleteData", mock.Anything, "aae9611e-f66c-4fe4-a6c6-2e2bdea69060")
	serviceMock.AssertExpectations(t)
}

func TestHandleWriteMessageNoUUIDForMetadataPublish(t *testing.T) {
	assert := assert.New(t)

	hook := logTest.NewTestHook("content-rw-elasticsearch")

	inputJSON, err := ioutil.ReadFile("testdata/testInput4.json")
	assert.NoError(err, "Unexpected error")

	serviceMock := &esServiceMock{}

	indexer := Indexer{esService: serviceMock}
	indexer.handleMessage(consumer.Message{Body: string(inputJSON), Headers: map[string]string{originHeader: methodeOrigin}})

	serviceMock.AssertNotCalled(t, "WriteData", mock.Anything, "b17756fe-0f62-4cf1-9deb-ca7a2ff80172", mock.Anything)
	serviceMock.AssertNotCalled(t, "DeleteData", mock.Anything, "b17756fe-0f62-4cf1-9deb-ca7a2ff80172")
	serviceMock.AssertExpectations(t)

	require.NotNil(t, hook.LastEntry())
	assert.Equal("info", hook.LastEntry().Level.String(), "Wrong log")
}

func TestHandleWriteMessageNoType(t *testing.T) {
	assert := assert.New(t)

	hook := logTest.NewTestHook("content-rw-elasticsearch")

	inputJSON, err := ioutil.ReadFile("testdata/exampleEnrichedContentModel.json")
	assert.NoError(err, "Unexpected error")
	input := strings.Replace(string(inputJSON), "FTCOM-METHODE", "invalid", 1)

	serviceMock := &esServiceMock{}

	indexer := Indexer{esService: serviceMock}
	indexer.handleMessage(consumer.Message{Body: input})

	serviceMock.AssertNotCalled(t, "WriteData", mock.Anything, mock.Anything, mock.Anything)
	serviceMock.AssertNotCalled(t, "DeleteData", mock.Anything, mock.Anything)
	require.NotNil(t, hook.LastEntry())
	assert.Equal("error", hook.LastEntry().Level.String(), "Wrong log")
}

func TestHandleWriteMessageError(t *testing.T) {
	assert := assert.New(t)

	hook := logTest.NewTestHook("content-rw-elasticsearch")

	inputJSON, err := ioutil.ReadFile("testdata/exampleEnrichedContentModel.json")
	assert.NoError(err, "Unexpected error")

	serviceMock := &esServiceMock{}
	serviceMock.On("WriteData", "FTCom", "aae9611e-f66c-4fe4-a6c6-2e2bdea69060", mock.Anything).Return(&elastic.IndexResult{}, elastic.ErrTimeout)
	concordanceApiMock := new(concordanceApiMock)
	concordanceApiMock.On("GetConcepts", mock.AnythingOfType("string"), mock.AnythingOfType("[]string")).Return(map[string]*ConceptModel{}, nil)

	indexer := Indexer{esService: serviceMock, ConceptGetter: concordanceApiMock}
	indexer.handleMessage(consumer.Message{Body: string(inputJSON)})

	serviceMock.AssertExpectations(t)
	require.NotNil(t, hook.LastEntry())
	assert.Equal("error", hook.LastEntry().Level.String(), "Wrong log")

	concordanceApiMock.AssertExpectations(t)
}

func TestHandleDeleteMessage(t *testing.T) {
	assert := assert.New(t)

	inputJSON, err := ioutil.ReadFile("testdata/exampleEnrichedContentModel.json")
	assert.NoError(err, "Unexpected error")
	input := strings.Replace(string(inputJSON), `"markedDeleted": "false"`, `"markedDeleted": "true"`, 1)

	serviceMock := &esServiceMock{}
	serviceMock.On("DeleteData", "FTCom", "aae9611e-f66c-4fe4-a6c6-2e2bdea69060").Return(&elastic.DeleteResult{}, nil)

	indexer := Indexer{esService: serviceMock}
	indexer.handleMessage(consumer.Message{Body: input})

	serviceMock.AssertExpectations(t)
}

func TestHandleDeleteMessageError(t *testing.T) {
	assert := assert.New(t)

	hook := logTest.NewTestHook("content-rw-elasticsearch")

	inputJSON, err := ioutil.ReadFile("testdata/exampleEnrichedContentModel.json")
	assert.NoError(err, "Unexpected error")
	input := strings.Replace(string(inputJSON), `"markedDeleted": "false"`, `"markedDeleted": "true"`, 1)

	serviceMock := &esServiceMock{}

	serviceMock.On("DeleteData", "FTCom", "aae9611e-f66c-4fe4-a6c6-2e2bdea69060").Return(&elastic.DeleteResult{}, elastic.ErrTimeout)

	indexer := Indexer{esService: serviceMock}
	indexer.handleMessage(consumer.Message{Body: input})

	serviceMock.AssertExpectations(t)
	require.NotNil(t, hook.LastEntry())
	assert.Equal("error", hook.LastEntry().Level.String(), "Wrong log")
}

func TestHandleMessageJsonError(t *testing.T) {
	assert := assert.New(t)

	hook := logTest.NewTestHook("content-rw-elasticsearch")

	serviceMock := &esServiceMock{}

	indexer := &Indexer{esService: serviceMock}
	indexer.handleMessage(consumer.Message{Body: "malformed json"})

	require.NotNil(t, hook.LastEntry())
	assert.Equal("error", hook.LastEntry().Level.String(), "Wrong log")
	serviceMock.AssertNotCalled(t, "WriteData", mock.Anything, mock.Anything, mock.Anything)
	serviceMock.AssertNotCalled(t, "DeleteData", mock.Anything, mock.Anything)
}

func TestHandleSyntheticMessage(t *testing.T) {
	assert := assert.New(t)

	hook := logTest.NewTestHook("content-rw-elasticsearch")

	serviceMock := &esServiceMock{}
	indexer := &Indexer{esService: serviceMock}
	indexer.handleMessage(consumer.Message{Headers: map[string]string{"X-Request-Id": "SYNTHETIC-REQ-MON_WuLjbRpCgh"}})

	require.NotNil(t, hook.LastEntry())
	assert.Equal("info", hook.LastEntry().Level.String(), "Wrong log")
	serviceMock.AssertNotCalled(t, "WriteData", mock.Anything, mock.Anything, mock.Anything)
	serviceMock.AssertNotCalled(t, "DeleteData", mock.Anything, mock.Anything)
}
