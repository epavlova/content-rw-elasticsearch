package main

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

func TestConvertToESContentModel(t *testing.T) {
	assert := assert.New(t)
	enrichedContentModel := enrichedContentModel{}
	inputJson, err := ioutil.ReadFile("exampleEnrichedContentModel.json")
	assert.NoError(err, "Unexpected error")

	err = json.Unmarshal([]byte(inputJson), &enrichedContentModel)
	assert.NoError(err, "Unexpected error")

	esModel := convertToESContentModel(enrichedContentModel, "article")

	actualJson, err := json.Marshal(esModel)
	assert.NoError(err, "Unexpected error")

	expectedJson, err := ioutil.ReadFile("exampleElasticModel.json")
	assert.NoError(err, "Unexpected error")

	assert.Equal(string(expectedJson[:]), string(actualJson[:]), "Expected JSON differs from actual JSON ")
}

func TestCmrID(t *testing.T) {
	assert := assert.New(t)
	assert.Equal("NzE0ZThkZGItNDAyMC00MDRjLTlkNzMtY2I5MzRmZDVhOWM2-T04=", getCmrID("ON", "714e8ddb-4020-404c-9d73-cb934fd5a9c6"), "Wrong CMR ID encoding")
}
