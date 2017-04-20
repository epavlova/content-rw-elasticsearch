package main

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
	"time"
)

func TestConvertToESContentModel(t *testing.T) {
	assert := assert.New(t)
	ecModel := enrichedContentModel{}
	inputJSON, err := ioutil.ReadFile("testdata/exampleEnrichedContentModel.json")
	assert.NoError(err, "Unexpected error")

	err = json.Unmarshal([]byte(inputJSON), &ecModel)
	assert.NoError(err, "Unexpected error")

	startTime := time.Now().UnixNano() / 1000000
	esModel := convertToESContentModel(ecModel, "article")

	endTime := time.Now().UnixNano() / 1000000

	indexDate, err := time.Parse("2006-01-02T15:04:05.999Z", *esModel.IndexDate)
	assert.NoError(err, "Unexpected error")
	indexTime := indexDate.UnixNano() / 1000000

	assert.True(indexTime >= startTime && indexTime <= endTime, "Index date %s not correct", *esModel.IndexDate)

	esModel.IndexDate = nil

	expectedJSON, err := ioutil.ReadFile("testdata/exampleElasticModel.json")
	assert.NoError(err, "Unexpected error")

	expectedESModel := esContentModel{}
	err = json.Unmarshal([]byte(expectedJSON), &expectedESModel)
	assert.NoError(err, "Unexpected error")

	assert.Equal(expectedESModel, esModel)
}

func TestCmrID(t *testing.T) {
	assert := assert.New(t)
	assert.Equal("NzE0ZThkZGItNDAyMC00MDRjLTlkNzMtY2I5MzRmZDVhOWM2-T04=", getCmrID("ON", "714e8ddb-4020-404c-9d73-cb934fd5a9c6"), "Wrong CMR ID encoding")
}
