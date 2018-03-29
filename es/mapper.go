package es

import (
	"encoding/base64"
	"strings"
	"time"

	"github.com/Financial-Times/go-logger"
	"github.com/Financial-Times/uuid-utils-go"
)

const (
	primaryClassification = "isPrimarilyClassifiedBy"
	about                 = "about"
	hasAuthor             = "hasAuthor"
	apiURLPrefix          = "https://www.ft.com/content/"
	imageServiceURL       = "https://www.ft.com/__origami/service/image/v2/images/raw/http%3A%2F%2Fprod-upp-image-read.ft.com%2F[image_uuid]?source=search&fit=scale-down&width=167"
	imagePlaceholder      = "[image_uuid]"

	tmeOrganisations  = "ON"
	tmePeople         = "PN"
	tmeAuthors        = "Authors"
	tmeBrands         = "Brands"
	tmeSubjects       = "Subjects"
	tmeSections       = "Sections"
	tmeTopics         = "Topics"
	tmeRegions        = "GL"
	tmeGenres         = "Genres"
	tmeSpecialReports = "SpecialReports"

	ArticleType = "article"
	VideoType   = "video"
	BlogType    = "blog"
)

var ContentTypeMap = map[string]contentType{
	"article": {
		Collection: "FTCom",
		Format:     "Articles",
		Category:   "article",
	},
	"blog": {
		Collection: "FTBlogs",
		Format:     "Blogs",
		Category:   "blogPost",
	},
	"video": {
		Collection: "FTVideos",
		Format:     "Videos",
		Category:   "video",
	},
}

type Mapper interface {
	MapContent(enrichedContent EnrichedContent, contentType string, tid string) IndexModel
}

type ContentMapper struct {
}

func NewContentMapper() *ContentMapper {
	return &ContentMapper{}
}

func (mapper *ContentMapper) MapContent(enrichedContent EnrichedContent, contentType string, tid string) IndexModel {
	model := IndexModel{}

	model.IndexDate = new(string)
	*model.IndexDate = time.Now().UTC().Format("2006-01-02T15:04:05.999Z")

	model.ContentType = new(string)
	*model.ContentType = contentType
	model.InternalContentType = new(string)
	*model.InternalContentType = contentType
	model.Category = new(string)
	*model.Category = ContentTypeMap[contentType].Category
	model.Format = new(string)
	*model.Format = ContentTypeMap[contentType].Format

	model.UID = &(enrichedContent.Content.UUID)

	model.LeadHeadline = new(string)
	*model.LeadHeadline = transformText(enrichedContent.Content.Title,
		htmlEntityTransformer,
		tagsRemover,
		outerSpaceTrimmer,
		duplicateWhiteSpaceRemover)

	model.Byline = new(string)
	*model.Byline = transformText(enrichedContent.Content.Byline,
		htmlEntityTransformer,
		tagsRemover,
		outerSpaceTrimmer,
		duplicateWhiteSpaceRemover)

	if enrichedContent.Content.PublishedDate != "" {
		model.LastPublish = &(enrichedContent.Content.PublishedDate)
	}
	if enrichedContent.Content.FirstPublishedDate != "" {
		model.InitialPublish = &(enrichedContent.Content.FirstPublishedDate)
	}
	model.Body = new(string)

	*model.Body = transformText(enrichedContent.Content.Body,
		interactiveGraphicsMarkupTagRemover,
		pullTagTransformer,
		htmlEntityTransformer,
		scriptTagRemover,
		tagsRemover,
		outerSpaceTrimmer,
		embed1Replacer,
		squaredCaptionReplacer,
		duplicateWhiteSpaceRemover)

	if contentType != BlogType && enrichedContent.Content.MainImage != "" {
		model.ThumbnailURL = new(string)

		var imageID *uuidutils.UUID

		//Generate the actual image UUID from the received image set UUID
		imageSetUUID, err := uuidutils.NewUUIDFromString(enrichedContent.Content.MainImage)
		if err == nil {
			imageID, err = uuidutils.NewUUIDDeriverWith(uuidutils.IMAGE_SET).From(imageSetUUID)
		}

		if err != nil {
			logger.WithError(err).Warnf("Couldn't generate image uuid for the image set with uuid %s: image field won't be populated.", enrichedContent.Content.MainImage)
		}

		*model.ThumbnailURL = strings.Replace(imageServiceURL, imagePlaceholder, imageID.String(), -1)
	}

	model.URL = new(string)
	*model.URL = apiURLPrefix + enrichedContent.Content.UUID

	model.PublishReference = tid

	primaryThemeCount := 0

	for _, annotation := range enrichedContent.Metadata {
		fallbackID := annotation.Thing.ID
		tmeIDs := []string{fallbackID}
		if len(annotation.Thing.TmeIDs) != 0 {
			tmeIDs = append(tmeIDs, annotation.Thing.TmeIDs...)
		} else {
			logger.Warnf("Indexing content with uuid %s - TME id missing for concept with id %s, using thing id instead", enrichedContent.Content.UUID, fallbackID)
		}
		for _, taxonomy := range annotation.Thing.Types {
			switch taxonomy {
			case "http://www.ft.com/ontology/organisation/Organisation":
				model.CmrOrgnames = appendIfNotExists(model.CmrOrgnames, annotation.Thing.PrefLabel)
				model.CmrOrgnamesIds = appendIfNotExists(model.CmrOrgnamesIds, getCmrID(tmeOrganisations, tmeIDs))
				if strings.HasSuffix(annotation.Thing.Predicate, about) {
					setPrimaryTheme(&model, &primaryThemeCount, annotation.Thing.PrefLabel, getCmrID(tmeOrganisations, tmeIDs))
				}
			case "http://www.ft.com/ontology/person/Person":
				cmrID := getCmrID(tmePeople, tmeIDs)
				authorCmrID := getCmrID(tmeAuthors, tmeIDs)
				// if it's only author, skip adding to people
				if cmrID != fallbackID || authorCmrID == fallbackID {
					model.CmrPeople = appendIfNotExists(model.CmrPeople, annotation.Thing.PrefLabel)
					model.CmrPeopleIds = appendIfNotExists(model.CmrPeopleIds, cmrID)
				}
				if strings.HasSuffix(annotation.Thing.Predicate, hasAuthor) {
					if authorCmrID != fallbackID {
						model.CmrAuthors = appendIfNotExists(model.CmrAuthors, annotation.Thing.PrefLabel)
						model.CmrAuthorsIds = appendIfNotExists(model.CmrAuthorsIds, authorCmrID)
					}
				}
				if strings.HasSuffix(annotation.Thing.Predicate, about) {
					setPrimaryTheme(&model, &primaryThemeCount, annotation.Thing.PrefLabel, getCmrID(tmePeople, tmeIDs))
				}
			case "http://www.ft.com/ontology/company/Company":
				model.CmrCompanynames = appendIfNotExists(model.CmrCompanynames, annotation.Thing.PrefLabel)
				model.CmrCompanynamesIds = appendIfNotExists(model.CmrCompanynamesIds, getCmrID(tmeOrganisations, tmeIDs))
			case "http://www.ft.com/ontology/product/Brand":
				model.CmrBrands = appendIfNotExists(model.CmrBrands, annotation.Thing.PrefLabel)
				model.CmrBrandsIds = appendIfNotExists(model.CmrBrandsIds, getCmrID(tmeBrands, tmeIDs))
			case "http://www.ft.com/ontology/Subject":
				model.CmrSubjects = appendIfNotExists(model.CmrSubjects, annotation.Thing.PrefLabel)
				model.CmrSubjectsIds = appendIfNotExists(model.CmrSubjectsIds, getCmrID(tmeSubjects, tmeIDs))
			case "http://www.ft.com/ontology/Section":
				model.CmrSections = appendIfNotExists(model.CmrSections, annotation.Thing.PrefLabel)
				model.CmrSectionsIds = appendIfNotExists(model.CmrSectionsIds, getCmrID(tmeSections, tmeIDs))
				if strings.HasSuffix(annotation.Thing.Predicate, primaryClassification) {
					model.CmrPrimarysection = new(string)
					*model.CmrPrimarysection = annotation.Thing.PrefLabel
					model.CmrPrimarysectionID = new(string)
					*model.CmrPrimarysectionID = getCmrID(tmeSections, tmeIDs)
				}
			case "http://www.ft.com/ontology/Topic":
				model.CmrTopics = appendIfNotExists(model.CmrTopics, annotation.Thing.PrefLabel)
				model.CmrTopicsIds = appendIfNotExists(model.CmrTopicsIds, getCmrID(tmeTopics, tmeIDs))
				if strings.HasSuffix(annotation.Thing.Predicate, about) {
					setPrimaryTheme(&model, &primaryThemeCount, annotation.Thing.PrefLabel, getCmrID(tmeTopics, tmeIDs))
				}
			case "http://www.ft.com/ontology/Location":
				model.CmrRegions = appendIfNotExists(model.CmrRegions, annotation.Thing.PrefLabel)
				model.CmrRegionsIds = appendIfNotExists(model.CmrRegionsIds, getCmrID(tmeRegions, tmeIDs))
				if strings.HasSuffix(annotation.Thing.Predicate, about) {
					setPrimaryTheme(&model, &primaryThemeCount, annotation.Thing.PrefLabel, getCmrID(tmeRegions, tmeIDs))
				}
			case "http://www.ft.com/ontology/Genre":
				model.CmrGenres = appendIfNotExists(model.CmrGenres, annotation.Thing.PrefLabel)
				model.CmrGenreIds = appendIfNotExists(model.CmrGenreIds, getCmrID(tmeGenres, tmeIDs))
			case "http://www.ft.com/ontology/SpecialReport":
				model.CmrSpecialreports = appendIfNotExists(model.CmrSpecialreports, annotation.Thing.PrefLabel)
				model.CmrSpecialreportsIds = appendIfNotExists(model.CmrSpecialreportsIds, getCmrID(tmeSpecialReports, tmeIDs))
				if strings.HasSuffix(annotation.Thing.Predicate, primaryClassification) {
					model.CmrPrimarysection = new(string)
					*model.CmrPrimarysection = annotation.Thing.PrefLabel
					model.CmrPrimarysectionID = new(string)
					*model.CmrPrimarysectionID = getCmrID(tmeSpecialReports, tmeIDs)
				}
			}
		}
	}
	return model
}
func setPrimaryTheme(model *IndexModel, pTCount *int, name string, id string) {
	if *pTCount == 0 {
		model.CmrPrimarytheme = new(string)
		*model.CmrPrimarytheme = name
		model.CmrPrimarythemeID = new(string)
		*model.CmrPrimarythemeID = id
	} else {
		model.CmrPrimarytheme = nil
		model.CmrPrimarythemeID = nil
	}
	*pTCount++
}

func getCmrID(taxonomy string, tmeIDs []string) string {
	encodedTaxonomy := base64.StdEncoding.EncodeToString([]byte(taxonomy))
	for _, tmeID := range tmeIDs {
		if strings.HasSuffix(tmeID, encodedTaxonomy) {
			return tmeID
		}
	}
	return tmeIDs[0]
}
func appendIfNotExists(s []string, e string) []string {
	for _, a := range s {
		if a == e {
			return s
		}
	}
	return append(s, e)
}
