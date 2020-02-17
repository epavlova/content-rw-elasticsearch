package test

import (
	"fmt"
	"github.com/Financial-Times/content-rw-elasticsearch/pkg/config"
	"io/ioutil"
	"log"
)

func ReadTestResource(testDataFileName string) []byte {
	filePath := config.GetResourceFilePath("test/data/" + testDataFileName)
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		err := fmt.Errorf("cannot read test resource '%s': %s", testDataFileName, err)
		log.Fatal(err)
	}
	return content
}