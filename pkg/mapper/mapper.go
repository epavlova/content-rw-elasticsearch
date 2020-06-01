package mapper

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Financial-Times/go-logger/v2"
	"github.com/Financial-Times/upp-go-sdk/pkg/internalcontent"

	"github.com/Financial-Times/content-rw-elasticsearch/v2/pkg/concept"
	"github.com/Financial-Times/content-rw-elasticsearch/v2/pkg/config"
	"github.com/Financial-Times/content-rw-elasticsearch/v2/pkg/html"
	"github.com/Financial-Times/content-rw-elasticsearch/v2/pkg/schema"
)

const (
	webURLPrefix     = "https://www.ft.com/content/"
	apiURLPrefix     = "/content/"
	imageServiceURL  = "https://www.ft.com/__origami/service/image/v2/images/raw/http%3A%2F%2Fprod-upp-image-read.ft.com%2F[image_uuid]?source=search&fit=scale-down&width=167"
	imagePlaceholder = "[image_uuid]"

	tmeOrganisations = "ON"
	tmePeople        = "PN"
	tmeAuthors       = "Authors"
	tmeTopics        = "Topics"
	tmeRegions       = "GL"

	videoPrefix = "video"
)

type Handler struct {
	ConceptReader  concept.Reader
	BaseAPIURL     string
	Config         config.AppConfig
	log            *logger.UPPLogger
	internalClient *internalcontent.ContentClient
}

var errNoAnnotation = errors.New("no annotation to be processed")

func NewMapperHandler(reader concept.Reader, baseAPIURL string, appConfig config.AppConfig, logger *logger.UPPLogger, internalClient *internalcontent.ContentClient) *Handler {
	return &Handler{
		ConceptReader:  reader,
		BaseAPIURL:     baseAPIURL,
		Config:         appConfig,
		log:            logger,
		internalClient: internalClient,
	}
}

func (h *Handler) ToIndexModel(enrichedContent schema.EnrichedContent, contentType string, tid string) schema.IndexModel {
	model := schema.IndexModel{}

	if strings.HasPrefix(h.BaseAPIURL, "http://") {
		h.BaseAPIURL = strings.Replace(h.BaseAPIURL, "http", "https", 1)
	}
	h.populateContentRelatedFields(&model, enrichedContent, contentType, tid)

	annotations, concepts, err := h.prepareAnnotationsWithConcepts(&enrichedContent, tid)
	log := h.log.WithTransactionID(tid).WithUUID(enrichedContent.UUID)
	if err != nil {
		if err == errNoAnnotation {
			log.Warn(err.Error())
		} else {
			log.WithError(err).Error(err)
		}
		return model
	}

	for _, annotation := range annotations {
		canonicalID := strings.TrimPrefix(annotation.ID, concept.ThingURIPrefix)
		concepts, found := concepts[annotation.ID]
		if !found {
			log.Warnf("No concordance found for %v", canonicalID)
			continue
		}
		annIDs := []string{canonicalID}
		if concepts.TmeIDs != nil {
			annIDs = append(annIDs, concepts.TmeIDs...)
		} else {
			log.Warnf("TME id missing for concept with id %s, using only canonical id", canonicalID)
		}

		h.populateAnnotationRelatedFields(annotation, &model, annIDs, canonicalID)
	}
	return model
}

func (h *Handler) populateAnnotationRelatedFields(annotation schema.Thing, model *schema.IndexModel, annIDs []string, canonicalID string) {
	h.handleSectionMapping(annotation, model, annIDs)

	about := h.Config.Predicates.Get("about")
	hasAuthor := h.Config.Predicates.Get("hasAuthor")
	hasContributor := h.Config.Predicates.Get("hasContributor")
	for _, taxonomy := range annotation.Types {
		conceptTypes := h.Config.ConceptTypes
		switch taxonomy {
		case conceptTypes.Get("organisation"):
			model.CmrOrgnames = appendIfNotExists(model.CmrOrgnames, annotation.PrefLabel)
			model.CmrOrgnamesIds = prepareElasticField(model.CmrOrgnamesIds, annIDs)
			if annotation.Predicate == about {
				setPrimaryTheme(model, annotation.PrefLabel, getCmrIDWithFallback(tmeOrganisations, annIDs))
			}
		case conceptTypes.Get("person"):
			_, personFound := getCmrID(tmePeople, annIDs)
			authorCmrID, authorFound := getCmrID(tmeAuthors, annIDs)
			// if it's only author, skip adding to people
			if personFound || !authorFound {
				model.CmrPeople = appendIfNotExists(model.CmrPeople, annotation.PrefLabel)
				model.CmrPeopleIds = prepareElasticField(model.CmrPeopleIds, annIDs)
			}
			if annotation.Predicate == hasAuthor || annotation.Predicate == hasContributor {
				if authorFound {
					model.CmrAuthors = appendIfNotExists(model.CmrAuthors, annotation.PrefLabel)
					model.CmrAuthorsIds = appendIfNotExists(model.CmrAuthorsIds, authorCmrID)
					model.CmrAuthorsIds = appendIfNotExists(model.CmrAuthorsIds, canonicalID)
				}
			}
			if annotation.Predicate == about {
				setPrimaryTheme(model, annotation.PrefLabel, getCmrIDWithFallback(tmePeople, annIDs))
			}
		case conceptTypes.Get("company"):
			model.CmrCompanynames = appendIfNotExists(model.CmrCompanynames, annotation.PrefLabel)
			model.CmrCompanynamesIds = prepareElasticField(model.CmrCompanynamesIds, annIDs)
		case conceptTypes.Get("brand"):
			model.CmrBrands = appendIfNotExists(model.CmrBrands, annotation.PrefLabel)
			model.CmrBrandsIds = prepareElasticField(model.CmrBrandsIds, annIDs)
		case conceptTypes.Get("topic"):
			model.CmrTopics = appendIfNotExists(model.CmrTopics, annotation.PrefLabel)
			model.CmrTopicsIds = prepareElasticField(model.CmrTopicsIds, annIDs)
			if annotation.Predicate == about {
				setPrimaryTheme(model, annotation.PrefLabel, getCmrIDWithFallback(tmeTopics, annIDs))
			}
		case conceptTypes.Get("location"):
			model.CmrRegions = appendIfNotExists(model.CmrRegions, annotation.PrefLabel)
			model.CmrRegionsIds = prepareElasticField(model.CmrRegionsIds, annIDs)
			if annotation.Predicate == about {
				setPrimaryTheme(model, annotation.PrefLabel, getCmrIDWithFallback(tmeRegions, annIDs))
			}
		case conceptTypes.Get("genre"):
			model.CmrGenres = appendIfNotExists(model.CmrGenres, annotation.PrefLabel)
			model.CmrGenreIds = prepareElasticField(model.CmrGenreIds, annIDs)
		}
	}
}

func (h *Handler) prepareAnnotationsWithConcepts(enrichedContent *schema.EnrichedContent, tid string) ([]schema.Thing, map[string]concept.Model, error) {
	var ids []string
	var anns []schema.Thing
	for _, a := range enrichedContent.Metadata {
		if a.Thing.Predicate == h.Config.Predicates.Get("mentions") || a.Thing.Predicate == h.Config.Predicates.Get("hasDisplayTag") {
			// ignore these annotations
			continue
		}
		ids = append(ids, a.Thing.ID)
		anns = append(anns, a.Thing)
	}

	if len(ids) == 0 {
		return nil, nil, errNoAnnotation
	}

	concepts, err := h.ConceptReader.GetConcepts(tid, ids)
	return anns, concepts, err
}

func (h *Handler) populateContentRelatedFields(model *schema.IndexModel, enrichedContent schema.EnrichedContent, contentType string, tid string) {
	model.IndexDate = new(string)
	*model.IndexDate = time.Now().UTC().Format("2006-01-02T15:04:05.999Z")
	model.ContentType = new(string)
	*model.ContentType = contentType
	model.InternalContentType = new(string)
	*model.InternalContentType = contentType
	model.Category = new(string)
	*model.Category = h.Config.ESContentTypeMetadataMap.Get(contentType).Category
	model.Format = new(string)
	*model.Format = h.Config.ESContentTypeMetadataMap.Get(contentType).Format
	model.UID = &(enrichedContent.Content.UUID)
	model.LeadHeadline = new(string)
	*model.LeadHeadline = html.TransformText(enrichedContent.Content.Title,
		html.EntityTransformer,
		html.TagsRemover,
		html.OuterSpaceTrimmer,
		html.DuplicateWhiteSpaceRemover)
	model.Byline = new(string)
	*model.Byline = html.TransformText(enrichedContent.Content.Byline,
		html.EntityTransformer,
		html.TagsRemover,
		html.OuterSpaceTrimmer,
		html.DuplicateWhiteSpaceRemover)
	if enrichedContent.Content.PublishedDate != "" {
		model.LastPublish = &(enrichedContent.Content.PublishedDate)
	}
	if enrichedContent.Content.FirstPublishedDate != "" {
		model.InitialPublish = &(enrichedContent.Content.FirstPublishedDate)
	}
	model.Body = new(string)
	if enrichedContent.Content.Body != "" {
		*model.Body = html.TransformText(enrichedContent.Content.Body,
			html.InteractiveGraphicsMarkupTagRemover,
			html.PullTagTransformer,
			html.EntityTransformer,
			html.ScriptTagRemover,
			html.TagsRemover,
			html.OuterSpaceTrimmer,
			html.Embed1Replacer,
			html.SquaredCaptionReplacer,
			html.DuplicateWhiteSpaceRemover)
	} else {
		*model.Body = enrichedContent.Content.Description
	}

	model.Scoop = enrichedContent.Content.Scoop
	model.CanBeSyndicated = enrichedContent.Content.CanBeSyndicated
	model.CanBeDistributed = enrichedContent.Content.CanBeDistributed

	model.ShortDescription = new(string)
	*model.ShortDescription = enrichedContent.Content.Standfirst

	if contentType != config.BlogType && enrichedContent.Content.MainImage != "" {
		model.ThumbnailURL = new(string)

		log := h.log.WithTransactionID(tid).WithUUID(enrichedContent.UUID)

		ic, err := h.internalClient.GetContent(enrichedContent.UUID, true)
		if err != nil || len(ic.MainImage.Members) == 0 || ic.MainImage.Members[0].APIURL == "" {
			log.WithError(err).Warnf("Couldn't get image UUID from %s", internalcontent.URLInternalContent)
		} else {
			uris := strings.Split(ic.MainImage.Members[0].APIURL, "/")
			if len(uris) > 0 {
				imageUUID := uris[len(uris)-1]
				*model.ThumbnailURL = strings.Replace(imageServiceURL, imagePlaceholder, imageUUID, -1)
			} else {
				log.WithError(err).Warnf("Couldn't get image UUID from %s", internalcontent.URLInternalContent)
			}
		}
	}

	if contentType == config.VideoType && len(enrichedContent.Content.DataSources) > 0 {
		for _, ds := range enrichedContent.Content.DataSources {
			if strings.HasPrefix(ds.MediaType, videoPrefix) {
				model.LengthMillis = ds.Duration
				break
			}
		}
	}

	if contentType == config.AudioType && len(enrichedContent.Content.DataSources) > 0 {
		for _, ds := range enrichedContent.Content.DataSources {
			model.LengthMillis = ds.Duration
			break
		}
	}

	model.URL = new(string)
	*model.URL = webURLPrefix + enrichedContent.Content.UUID
	model.ModelAPIURL = new(string)
	*model.ModelAPIURL = fmt.Sprintf("%v%v%v", h.BaseAPIURL, apiURLPrefix, enrichedContent.Content.UUID)
	model.PublishReference = tid
}

func prepareElasticField(elasticField []string, annIDs []string) []string {
	for _, id := range annIDs {
		elasticField = appendIfNotExists(elasticField, id)
	}
	return elasticField
}

func (h *Handler) handleSectionMapping(annotation schema.Thing, model *schema.IndexModel, annIDs []string) {
	// handle sections
	predicates := h.Config.Predicates
	switch annotation.Predicate {
	case predicates.Get("about"),
		predicates.Get("majorMentions"),
		predicates.Get("implicitlyAbout"),
		predicates.Get("isClassifiedBy"),
		predicates.Get("implicitlyClassifiedBy"):
		model.CmrSections = appendIfNotExists(model.CmrSections, annotation.PrefLabel)
		model.CmrSectionsIds = prepareElasticField(model.CmrSectionsIds, annIDs)
	case predicates.Get("isPrimaryClassifiedBy"):
		model.CmrSections = appendIfNotExists(model.CmrSections, annotation.PrefLabel)
		model.CmrSectionsIds = prepareElasticField(model.CmrSectionsIds, annIDs)
		model.CmrPrimarysection = new(string)
		*model.CmrPrimarysection = annotation.PrefLabel
		model.CmrPrimarysectionID = new(string)
		*model.CmrPrimarysectionID = getCmrIDWithFallback("Sections", annIDs)
	}
}

func setPrimaryTheme(model *schema.IndexModel, name string, id string) {
	if model.CmrPrimarytheme != nil {
		return
	}
	model.CmrPrimarytheme = new(string)
	*model.CmrPrimarytheme = name
	model.CmrPrimarythemeID = new(string)
	*model.CmrPrimarythemeID = id
}

func getCmrID(taxonomy string, annotationIDs []string) (string, bool) {
	encodedTaxonomy := base64.StdEncoding.EncodeToString([]byte(taxonomy))
	for _, annID := range annotationIDs {
		if strings.HasSuffix(annID, encodedTaxonomy) {
			return annID, true
		}
	}
	return "", false
}

func getCmrIDWithFallback(taxonomy string, annotationIDs []string) string {
	encodedTaxonomy := base64.StdEncoding.EncodeToString([]byte(taxonomy))
	for _, annID := range annotationIDs {
		if strings.HasSuffix(annID, encodedTaxonomy) {
			return annID
		}
	}
	if len(annotationIDs) > 1 {
		return annotationIDs[1]
	}
	return annotationIDs[0]
}

func appendIfNotExists(s []string, e string) []string {
	for _, a := range s {
		if a == e {
			return s
		}
	}
	return append(s, e)
}
