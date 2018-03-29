package mapper

import (
	"encoding/json"
	"io/ioutil"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConvertToESContentModel(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		inputFile  string
		outputFile string
		tid        string
	}{
		{"../testdata/exampleEnrichedContentModel.json", "../testdata/exampleElasticModel.json", "tid_1"},
		{"../testdata/testInput1.json", "../testdata/testOutput1.json", "tid_1"},
		{"../testdata/testInput2.json", "../testdata/testOutput2.json", "tid_1"},
		{"../testdata/testInput3.json", "../testdata/testOutput3.json", "tid_1"},
		{"../testdata/testInputMultipleAbouts.json", "../testdata/testOutputMultipleAbouts.json", "tid_1"},
	}
	for _, test := range tests {
		ecModel := EnrichedContent{}
		inputJSON, err := ioutil.ReadFile(test.inputFile)
		assert.NoError(err, "Unexpected error")

		err = json.Unmarshal([]byte(inputJSON), &ecModel)
		assert.NoError(err, "Unexpected error")

		startTime := time.Now().UnixNano() / 1000000
		esModel := ToIndexModel(ecModel, "article", test.tid)

		endTime := time.Now().UnixNano() / 1000000

		indexDate, err := time.Parse("2006-01-02T15:04:05.999Z", *esModel.IndexDate)
		assert.NoError(err, "Unexpected error")
		indexTime := indexDate.UnixNano() / 1000000

		assert.True(indexTime >= startTime && indexTime <= endTime, "Index date %s not correct", *esModel.IndexDate)

		esModel.IndexDate = nil

		expectedJSON, err := ioutil.ReadFile(test.outputFile)
		assert.NoError(err, "Unexpected error")

		expectedESModel := IndexModel{}
		err = json.Unmarshal([]byte(expectedJSON), &expectedESModel)
		assert.NoError(err, "Unexpected error")

		//the publishRef field is actually overwritten with the x-request-header received from the message, instead of the one read from doc-store
		expectedESModel.PublishReference = test.tid
		assert.Equal(expectedESModel, esModel)
	}
}

func TestCmrID(t *testing.T) {
	assert := assert.New(t)
	assert.Equal("NzE0ZThkZGItNDAyMC00MDRjLTlkNzMtY2I5MzRmZDVhOWM2-T04=",
		getCmrID("ON", []string{"YzcxMTcyNGYtMzQyZC00ZmU2LTk0ZGYtYWI2Y2YxMDMwMTQy-QXV0aG9ycw==", "NzE0ZThkZGItNDAyMC00MDRjLTlkNzMtY2I5MzRmZDVhOWM2-T04="}), "Wrong CMR ID")
}
