package content

import (
	logTest "github.com/Financial-Times/go-logger/test"
	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gopkg.in/olivere/elastic.v2"
	"io/ioutil"
	"net/url"
	"strings"
	"testing"
	"time"
	"github.com/stretchr/testify/require"
	"github.com/Financial-Times/content-rw-elasticsearch/es"
	"net/http"
	"sync"
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
	var wg sync.WaitGroup
	indexer := NewContentIndexer(es.NewService("index"), es.NewContentMapper(), http.DefaultClient, queueConfig, &wg, NewClient)

	indexer.Start("app", "name", "index", "1984", accessConfig)
	defer indexer.Stop()

	time.Sleep(100 * time.Millisecond)

	assert.NotNil(indexer.esServiceInstance, "Elastic Service should be initialized")
	assert.Equal("index", (indexer.esServiceInstance).(*es.Service).IndexName, "Wrong index")
	(indexer.esServiceInstance).(*es.Service).Lock()
	assert.NotNil((indexer.esServiceInstance).(*es.Service).ElasticClient, "Elastic client should be initialized")
	(indexer.esServiceInstance).(*es.Service).Unlock()
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

	var wg sync.WaitGroup
	indexer := NewContentIndexer(es.NewService("index"), es.NewContentMapper(), http.DefaultClient, queueConfig, &wg, NewClient)

	indexer.Start("app", "name", "index", "1984", accessConfig)
	defer indexer.Stop()

	time.Sleep(100 * time.Millisecond)

	require.NotNil(t, hook.LastEntry())
	assert.Equal("error", hook.LastEntry().Level.String(), "Wrong log")
	assert.NotNil(indexer.esServiceInstance, "Elastic Service should be initialized")
	assert.Equal("index", (indexer.esServiceInstance).(*es.Service).IndexName, "Wrong index")
	assert.Nil((indexer.esServiceInstance).(*es.Service).ElasticClient, "Elastic client should not be initialized")
}

func TestHandleWriteMessage(t *testing.T) {
	assert := assert.New(t)

	inputJSON, err := ioutil.ReadFile("../testdata/exampleEnrichedContentModel.json")
	assert.NoError(err, "Unexpected error")

	serviceMock := &esServiceMock{}

	serviceMock.On("WriteData", "FTCom", "aae9611e-f66c-4fe4-a6c6-2e2bdea69060", mock.Anything).Return(&elastic.IndexResult{}, nil)

	indexer := Indexer{esServiceInstance: serviceMock, Mapper: es.NewContentMapper()}
	indexer.handleMessage(consumer.Message{Body: string(inputJSON)})

	serviceMock.AssertExpectations(t)
}

func TestHandleWriteMessageBlog(t *testing.T) {
	assert := assert.New(t)

	inputJSON, err := ioutil.ReadFile("../testdata/exampleEnrichedContentModel.json")
	assert.NoError(err, "Unexpected error")
	input := strings.Replace(string(inputJSON), "FTCOM-METHODE", "FT-LABS-WP1234", 1)

	serviceMock := &esServiceMock{}

	serviceMock.On("WriteData", "FTBlogs", "aae9611e-f66c-4fe4-a6c6-2e2bdea69060", mock.Anything).Return(&elastic.IndexResult{}, nil)

	indexer := Indexer{esServiceInstance: serviceMock, Mapper: es.NewContentMapper()}
	indexer.handleMessage(consumer.Message{Body: input})

	serviceMock.AssertExpectations(t)
}

func TestHandleWriteMessageBlogWithHeader(t *testing.T) {
	assert := assert.New(t)

	inputJSON, err := ioutil.ReadFile("../testdata/exampleEnrichedContentModel.json")
	assert.NoError(err, "Unexpected error")
	input := strings.Replace(string(inputJSON), "FTCOM-METHODE", "invalid", 1)

	serviceMock := &esServiceMock{}

	serviceMock.On("WriteData", "FTBlogs", "aae9611e-f66c-4fe4-a6c6-2e2bdea69060", mock.Anything).Return(&elastic.IndexResult{}, nil)

	indexer := Indexer{esServiceInstance: serviceMock, Mapper: es.NewContentMapper()}
	indexer.handleMessage(consumer.Message{Body: input, Headers: map[string]string{"Origin-System-Id": "wordpress"}})

	serviceMock.AssertExpectations(t)
}

func TestHandleWriteMessageVideo(t *testing.T) {
	assert := assert.New(t)

	inputJSON, err := ioutil.ReadFile("../testdata/exampleEnrichedContentModel.json")
	assert.NoError(err, "Unexpected error")
	input := strings.Replace(string(inputJSON), "FTCOM-METHODE", "NEXT-VIDEO-EDITOR", 1)

	serviceMock := &esServiceMock{}

	serviceMock.On("WriteData", "FTVideos", "aae9611e-f66c-4fe4-a6c6-2e2bdea69060", mock.Anything).Return(&elastic.IndexResult{}, nil)

	indexer := Indexer{esServiceInstance: serviceMock, Mapper: es.NewContentMapper()}
	indexer.handleMessage(consumer.Message{Body: input})

	serviceMock.AssertExpectations(t)
}

func TestHandleWriteMessageUnknownType(t *testing.T) {
	assert := assert.New(t)

	inputJSON, err := ioutil.ReadFile("../testdata/exampleEnrichedContentModel.json")
	assert.NoError(err, "Unexpected error")
	input := strings.Replace(string(inputJSON), `"Article"`, `"Content"`, 1)

	serviceMock := &esServiceMock{}

	indexer := Indexer{esServiceInstance: serviceMock, Mapper: es.NewContentMapper()}
	indexer.handleMessage(consumer.Message{Body: input})

	serviceMock.AssertNotCalled(t, "WriteData", mock.Anything, "aae9611e-f66c-4fe4-a6c6-2e2bdea69060", mock.Anything)
	serviceMock.AssertNotCalled(t, "DeleteData", mock.Anything, "aae9611e-f66c-4fe4-a6c6-2e2bdea69060")
	serviceMock.AssertExpectations(t)
}

func TestHandleWriteMessageNoUUIDForMetadataPublish(t *testing.T) {
	assert := assert.New(t)

	hook := logTest.NewTestHook("content-rw-elasticsearch")

	inputJSON, err := ioutil.ReadFile("../testdata/testInput4.json")
	assert.NoError(err, "Unexpected error")

	serviceMock := &esServiceMock{}

	indexer := Indexer{esServiceInstance: serviceMock, Mapper: es.NewContentMapper()}
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

	inputJSON, err := ioutil.ReadFile("../testdata/exampleEnrichedContentModel.json")
	assert.NoError(err, "Unexpected error")
	input := strings.Replace(string(inputJSON), "FTCOM-METHODE", "invalid", 1)

	serviceMock := &esServiceMock{}

	indexer := Indexer{esServiceInstance: serviceMock, Mapper: es.NewContentMapper()}
	indexer.handleMessage(consumer.Message{Body: input})

	serviceMock.AssertNotCalled(t, "WriteData", mock.Anything, mock.Anything, mock.Anything)
	serviceMock.AssertNotCalled(t, "DeleteData", mock.Anything, mock.Anything)
	require.NotNil(t, hook.LastEntry())
	assert.Equal("error", hook.LastEntry().Level.String(), "Wrong log")
}

func TestHandleWriteMessageError(t *testing.T) {
	assert := assert.New(t)

	hook := logTest.NewTestHook("content-rw-elasticsearch")

	inputJSON, err := ioutil.ReadFile("../testdata/exampleEnrichedContentModel.json")
	assert.NoError(err, "Unexpected error")

	serviceMock := &esServiceMock{}

	serviceMock.On("WriteData", "FTCom", "aae9611e-f66c-4fe4-a6c6-2e2bdea69060", mock.Anything).Return(&elastic.IndexResult{}, elastic.ErrTimeout)

	indexer := Indexer{esServiceInstance: serviceMock, Mapper: es.NewContentMapper()}
	indexer.handleMessage(consumer.Message{Body: string(inputJSON)})

	serviceMock.AssertExpectations(t)
	require.NotNil(t, hook.LastEntry())
	assert.Equal("error", hook.LastEntry().Level.String(), "Wrong log")
}

func TestHandleDeleteMessage(t *testing.T) {
	assert := assert.New(t)

	inputJSON, err := ioutil.ReadFile("../testdata/exampleEnrichedContentModel.json")
	assert.NoError(err, "Unexpected error")
	input := strings.Replace(string(inputJSON), `"marked_deleted": false`, `"marked_deleted": true`, 1)

	serviceMock := &esServiceMock{}

	serviceMock.On("DeleteData", "FTCom", "aae9611e-f66c-4fe4-a6c6-2e2bdea69060").Return(&elastic.DeleteResult{}, nil)

	indexer := Indexer{esServiceInstance: serviceMock, Mapper: es.NewContentMapper()}
	indexer.handleMessage(consumer.Message{Body: input})

	serviceMock.AssertExpectations(t)
}

func TestHandleDeleteMessageError(t *testing.T) {
	assert := assert.New(t)

	hook := logTest.NewTestHook("content-rw-elasticsearch")

	inputJSON, err := ioutil.ReadFile("../testdata/exampleEnrichedContentModel.json")
	assert.NoError(err, "Unexpected error")
	input := strings.Replace(string(inputJSON), `"marked_deleted": false`, `"marked_deleted": true`, 1)

	serviceMock := &esServiceMock{}

	serviceMock.On("DeleteData", "FTCom", "aae9611e-f66c-4fe4-a6c6-2e2bdea69060").Return(&elastic.DeleteResult{}, elastic.ErrTimeout)

	indexer := Indexer{esServiceInstance: serviceMock, Mapper: es.NewContentMapper()}
	indexer.handleMessage(consumer.Message{Body: input})

	serviceMock.AssertExpectations(t)
	require.NotNil(t, hook.LastEntry())
	assert.Equal("error", hook.LastEntry().Level.String(), "Wrong log")
}

func TestHandleMessageJsonError(t *testing.T) {
	assert := assert.New(t)

	hook := logTest.NewTestHook("content-rw-elasticsearch")

	serviceMock := &esServiceMock{}

	indexer := &Indexer{esServiceInstance: serviceMock}
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
	indexer := &Indexer{esServiceInstance: serviceMock, Mapper: es.NewContentMapper()}
	indexer.handleMessage(consumer.Message{Headers: map[string]string{"X-Request-Id": "SYNTHETIC-REQ-MON_WuLjbRpCgh"}})

	require.NotNil(t, hook.LastEntry())
	assert.Equal("info", hook.LastEntry().Level.String(), "Wrong log")
	serviceMock.AssertNotCalled(t, "WriteData", mock.Anything, mock.Anything, mock.Anything)
	serviceMock.AssertNotCalled(t, "DeleteData", mock.Anything, mock.Anything)
}
