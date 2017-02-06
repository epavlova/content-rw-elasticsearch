package main

import (
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"net/http"
)

type contentWriter struct {
	elasticService *esServiceI
}

func newESWriter(elasticService *esServiceI) (service *contentWriter) {
	return &contentWriter{elasticService: elasticService}
}

func (service *contentWriter) writeData(writer http.ResponseWriter, request *http.Request) {

	uuid := mux.Vars(request)["id"]
	contentType := mux.Vars(request)["content-type"]

	var content enrichedContentModel
	decoder := json.NewDecoder(request.Body)
	err := decoder.Decode(&content)
	if err != nil {
		log.Errorf(err.Error())
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer request.Body.Close()

	if content.Content.Uuid != uuid {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	payload := convertToESContentModel(content, contentType)

	_, err = (*service.elasticService).writeData(m[contentType].collection, uuid, payload)
	if err != nil {
		log.Errorf(err.Error())
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	writer.WriteHeader(http.StatusOK)
}

func (service *contentWriter) readData(writer http.ResponseWriter, request *http.Request) {

	uuid := mux.Vars(request)["id"]
	contentType := mux.Vars(request)["content-type"]

	getResult, err := (*service.elasticService).readData(m[contentType].collection, uuid)

	if err != nil {
		log.Errorf(err.Error())
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	if !getResult.Found {
		writer.WriteHeader(http.StatusNotFound)
		return
	}

	writer.Header().Add("Content-Type", "application/json")
	enc := json.NewEncoder(writer)
	enc.Encode(getResult.Source)

}

func (service *contentWriter) deleteData(writer http.ResponseWriter, request *http.Request) {

	uuid := mux.Vars(request)["id"]
	contentType := mux.Vars(request)["content-type"]

	res, err := (*service.elasticService).deleteData(m[contentType].collection, uuid)

	if err != nil {
		log.Errorf(err.Error())
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	if !res.Found {
		writer.WriteHeader(http.StatusNotFound)
		return
	}

	writer.WriteHeader(http.StatusOK)
}
