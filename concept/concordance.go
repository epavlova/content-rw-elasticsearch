package concept

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

const (
	ThingURIPrefix         = "http://api.ft.com/things/"
	concordancesEndpoint   = "/concordances"
	concordancesQueryParam = "conceptId"
	tmeAuthority           = "http://api.ft.com/system/FT-TME"
	uppAuthority           = "http://api.ft.com/system/UPP"
)

type concept struct {
	ID     string `json:"id"`
	APIURL string `json:"apiUrl,omitempty"`
}

type identifier struct {
	IdentifierValue string `json:"identifierValue"`
	Authority       string `json:"authority"`
}

type concordance struct {
	Concept    concept    `json:"concept"`
	Identifier identifier `json:"identifier"`
}

type ConcordancesResponse struct {
	Concordances []concordance `json:"concordances"`
}

type Model struct {
	TmeIDs []string
}

type ConceptGetter interface {
	GetConcepts(tid string, ids []string) (map[string]*Model, error)
}

type Client interface {
	Do(req *http.Request) (resp *http.Response, err error)
}

type ConcordanceApiService struct {
	ConcordanceApiBaseURL string
	Client                Client
}

func NewConcordanceApiService(concordanceApiBaseURL string, c Client) *ConcordanceApiService {
	return &ConcordanceApiService{ConcordanceApiBaseURL: concordanceApiBaseURL, Client: c}
}

func (c *ConcordanceApiService) GetConcepts(tid string, ids []string) (map[string]*Model, error) {
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

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("calling Concordance API returned HTTP status %v", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var concordancesResp ConcordancesResponse
	if err = json.Unmarshal(body, &concordancesResp); err != nil {
		return nil, err
	}

	return TransformToConceptModel(concordancesResp), nil
}

func TransformToConceptModel(concordancesResp ConcordancesResponse) map[string]*Model {
	conceptMap := make(map[string]*Model)
	for _, c := range concordancesResp.Concordances {
		concept, found := conceptMap[c.Concept.ID]
		if !found {
			conceptMap[c.Concept.ID] = &Model{}
			concept = conceptMap[c.Concept.ID]
		}

		if c.Identifier.Authority == tmeAuthority {
			concept.TmeIDs = append(concept.TmeIDs, c.Identifier.IdentifierValue)
		}
		if c.Identifier.Authority == uppAuthority {
			_, found := conceptMap[ThingURIPrefix+c.Identifier.IdentifierValue]
			if !found {
				conceptMap[ThingURIPrefix+c.Identifier.IdentifierValue] = concept
			}
		}
	}

	return conceptMap
}

func (c *ConcordanceApiService) HealthCheck() (string, error) {
	req, err := http.NewRequest("GET", c.ConcordanceApiBaseURL+"/__gtg", nil)
	if err != nil {
		return "", err
	}

	req.Header.Add("User-Agent", "UPP content-rw-elasticsearch")

	resp, err := c.Client.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Health check returned a non-200 HTTP status: %v", resp.StatusCode)
	}
	return "Concordance API is healthy", nil
}
