package main

import (
	"net/http"
	"io/ioutil"
	"fmt"
	"encoding/json"
)

const (
	thingURIPrefix         = "http://api.ft.com/things/"
	concordancesEndpoint   = "/concordances"
	concordancesQueryParam = "conceptId"
	tmeAuthority           = "http://api.ft.com/system/FT-TME"
	uppAuthority           = "http://api.ft.com/system/UPP"
)

type concept struct {
	ID     string       `json:"id"`
	APIURL string       `json:"apiUrl,omitempty"`
}

type identifier struct {
	IdentifierValue string `json:"identifierValue"`
	Authority       string `json:"authority"`
}

type concordance struct {
	Concept    concept    `json:"concept"`
	Identifier identifier `json:"identifier"`
}

type concordancesResponse struct {
	Concordances []concordance    `json:"concordances"`
}

type ConceptModel struct {
	TmeIDs []string
}

type ConceptGetter interface {
	GetConcepts(tid string, ids []string) (map[string]*ConceptModel, error)
}

type ConcordanceApiService struct {
	ConcordanceApiBaseURL string
	Client                *http.Client
}

func NewConcordanceApiService(concordanceApiBaseURL string, c *http.Client) *ConcordanceApiService {
	return &ConcordanceApiService{ConcordanceApiBaseURL: concordanceApiBaseURL, Client: c}
}

func (c *ConcordanceApiService) GetConcepts(tid string, ids []string) (map[string]*ConceptModel, error) {
	req, err := http.NewRequest(http.MethodGet, c.ConcordanceApiBaseURL+concordancesEndpoint, nil)
	if err != nil {
		return nil, err
	}

	queryParams := req.URL.Query()
	for _, id := range ids {
		queryParams.Add(concordancesQueryParam, id)
	}
	req.URL.RawQuery = queryParams.Encode()

	req.Header.Add("User-Agent", "UPP content-rw-elasticsearch")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("X-Request-Id", tid)

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Calling Concordance API returned HTTP status %s", resp.StatusCode)
	}

	var concordancesResp concordancesResponse
	if err = json.Unmarshal(body, &concordancesResp); err != nil {
		return nil, err
	}

	return transformToConceptModel(concordancesResp), nil
}

func transformToConceptModel(concordancesResp concordancesResponse) map[string]*ConceptModel {
	conceptMap := make(map[string]*ConceptModel)
	for _, c := range concordancesResp.Concordances {
		concept, found := conceptMap[c.Concept.ID]
		if !found {
			conceptMap[c.Concept.ID] = &ConceptModel{}
			concept = conceptMap[c.Concept.ID]
		}

		if c.Identifier.Authority == tmeAuthority {
			concept.TmeIDs = append(concept.TmeIDs, c.Identifier.IdentifierValue)
		}
		if c.Identifier.Authority == uppAuthority {
			_, found := conceptMap[c.Identifier.IdentifierValue]
			if !found {
				conceptMap[thingURIPrefix+c.Identifier.IdentifierValue] = concept
			}
		}
	}
	
	return conceptMap
}
