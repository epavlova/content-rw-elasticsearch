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
	enrichedContentModel := enrichedContentModel{}
	inputJson, err := ioutil.ReadFile("exampleEnrichedContentModel.json")
	assert.NoError(err, "Unexpected error")

	err = json.Unmarshal([]byte(inputJson), &enrichedContentModel)
	assert.NoError(err, "Unexpected error")

	startTime := time.Now().UTC().Format("2006-01-02T15:04:05.999Z")

	esModel := convertToESContentModel(enrichedContentModel, "article")

	endTime := time.Now().UTC().Format("2006-01-02T15:04:05.999Z")

	assert.True(*esModel.IndexDate >= startTime && *esModel.IndexDate <= endTime, "Index date %s not correct", *esModel.IndexDate)

	esModel.IndexDate = nil

	expectedJson, err := ioutil.ReadFile("exampleElasticModel.json")
	assert.NoError(err, "Unexpected error")

	expectedESModel := esContentModel{}
	err = json.Unmarshal([]byte(expectedJson), &expectedESModel)
	assert.NoError(err, "Unexpected error")

	assert.Equal(expectedESModel, esModel)
}

func TestCmrID(t *testing.T) {
	assert := assert.New(t)
	assert.Equal("NzE0ZThkZGItNDAyMC00MDRjLTlkNzMtY2I5MzRmZDVhOWM2-T04=", getCmrID("ON", "714e8ddb-4020-404c-9d73-cb934fd5a9c6"), "Wrong CMR ID encoding")
}
