package main

import (
	"encoding/base64"
	log "github.com/Sirupsen/logrus"
	"strings"
	"time"
)

const (
	primaryClassification = "isPrimarilyClassifiedBy"
	about                 = "about"
	hasAuthor             = "hasAuthor"
	apiURLPrefix          = "https://www.ft.com/content/"
	imageServiceURL       = "https://www.ft.com/__origami/service/image/v2/images/raw/http%3A%2F%2Fcom.ft.imagepublish.prod-us.s3.amazonaws.com%2F[image_uuid]?source=search&fit=scale-down&width=167"
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
)

type enrichedContentModel struct {
	UUID     string       `json:"uuid"`
	Content  contentModel `json:"content"`
	Metadata annotations  `json:"v1-metadata"`
}

type contentModel struct {
	UUID               string       `json:"uuid"`
	Title              string       `json:"title"`
	Body               string       `json:"body"`
	Identifiers        []identifier `json:"identifiers"`
	PublishedDate      string       `json:"publishedDate"`
	LastModified       string       `json:"lastModified"`
	FirstPublishedDate string       `json:"firstPublishedDate"`
	MarkedDeleted      bool         `json:"marked_deleted"`
	Byline             string       `json:"byline"`
	Standfirst         string       `json:"standfirst"`
	Description        string       `json:"description"`
	MainImage          string       `json:"mainImage"`
	PublishReference   string       `json:"publishReference"`
}

type identifier struct {
	Authority       string `json:"authority"`
	IdentifierValue string `json:"identifierValue"`
}

type annotations []annotation

//Annotation is the main struct used to create and return structures
type annotation struct {
	Thing thing `json:"thing,omitempty"`
}

//Thing represents a concept being linked to
type thing struct {
	ID        string   `json:"id,omitempty"`
	PrefLabel string   `json:"prefLabel,omitempty"`
	Types     []string `json:"types,omitempty"`
	Predicate string   `json:"predicate,omitempty"`

	//INFO from the public-annotations-api
	TmeIDs []string `json:"tmeIDs,omitempty"`
}

type esContentModel struct {
	UID                   *string  `json:"uid"`
	LastMetadataPublish   *string  `json:"last_metadata_publish"`
	IndexDate             *string  `json:"index_date"`
	MarkDeleted           bool     `json:"mark_deleted"`
	StoryID               *int32   `json:"story_id"`
	LeadHeadline          *string  `json:"lead_headline"`
	Byline                *string  `json:"byline"`
	Body                  *string  `json:"body"`
	URL                   *string  `json:"url"`
	InitialPublish        *string  `json:"initial_publish"`
	LastPublish           *string  `json:"last_publish"`
	ContentType           *string  `json:"content_type"`
	ProviderName          *string  `json:"provider_name"`
	LengthMillis          int32    `json:"length_millis"`
	ShortDescription      *string  `json:"short_description"`
	ThumbnailURL          *string  `json:"thumbnail_url"`
	SectionLink           *string  `json:"section_link"`
	SecondaryImageID      *string  `json:"secondary_image_id"`
	ContributorRights     *string  `json:"contributor_rights"`
	SourceCode            *string  `json:"source_code"`
	StorymodelID          *int32   `json:"storymodel_id"`
	ModelAPIURL           *string  `json:"model_api_url"`
	ModelMasterSource     *string  `json:"model_master_source"`
	ModelMasterID         *string  `json:"model_master_id"`
	ModelExcerpt          *string  `json:"model_excerpt"`
	ModelResourceURI      *string  `json:"model_resource_uri"`
	CmrPrimarysection     *string  `json:"cmr_primarysection"`
	CmrPrimarytheme       *string  `json:"cmr_primarytheme"`
	CmrMediatype          *string  `json:"cmr_mediatype"`
	CmrMetadataupdatetime *string  `json:"cmr_metadataupdatetime"`
	CmrPrimarysectionID   *string  `json:"cmr_primarysection_id"`
	CmrPrimarythemeID     *string  `json:"cmr_primarytheme_id"`
	CmrMediatypeID        *string  `json:"cmr_mediatype_id"`
	CmrBrands             []string `json:"cmr_brands"`
	CmrBrandsIds          []string `json:"cmr_brands_ids"`
	CmrSpecialreports     []string `json:"cmr_specialreports"`
	CmrSpecialreportsIds  []string `json:"cmr_specialreports_ids"`
	CmrSections           []string `json:"cmr_sections"`

	CmrSectionsIds []string `json:"cmr_sections_ids"`

	CmrSubjects    []string `json:"cmr_subjects"`
	CmrSubjectsIds []string `json:"cmr_subjects_ids"`
	CmrTopics      []string `json:"cmr_topics"`
	CmrTopicsIds   []string `json:"cmr_topics_ids"`
	CmrPeople      []string `json:"cmr_people"`

	CmrPeopleIds []string `json:"cmr_people_ids"`

	CmrRegions []string `json:"cmr_regions"`

	CmrRegionsIds []string `json:"cmr_regions_ids"`

	CmrIcb    []string `json:"cmr_icb"`
	CmrIcbIds []string `json:"cmr_icb_ids"`
	CmrIptc   []string `json:"cmr_iptc"`

	CmrIptcIds []string `json:"cmr_iptc_ids"`

	CmrAuthorsIds []string `json:"cmr_authors_ids"`

	CmrAuthors []string `json:"cmr_authors"`

	CmrCompanynames    []string `json:"cmr_companynames"`
	CmrCompanynamesIds []string `json:"cmr_companynames_ids"`
	CmrOrgnames        []string `json:"cmr_orgnames"`

	CmrOrgnamesIds []string `json:"cmr_orgnames_ids"`

	BestStory           bool     `json:"bestStory"`
	InternalContentType *string  `json:"internalContentType"`
	Category            *string  `json:"category"`
	LookupFailure       bool     `json:"lookupFailure"`
	Format              *string  `json:"format"`
	CmrGenres           []string `json:"cmr_genre"`

	CmrGenreIds []string `json:"cmr_genre_id"`

	Region           *string  `json:"region"`
	Topics           []string `json:"topics"`
	DisplayCodes     []string `json:"displayCodes"`
	DisplayCodeNames []string `json:"displayCodeNames"`

	NaicsNames                 []string `json:"naicsNames"`
	EditorsTags                []string `json:"editorsTags"`
	CountryCodes               []string `json:"countryCodes"`
	CountryNames               []string `json:"countryNames"`
	Subjects                   []string `json:"subjects"`
	CompanyNamesAuto           []string `json:"companyNamesAuto"`
	OrganisationNamesAuto      []string `json:"organisationNamesAuto"`
	CompanyNamesEditorial      []string `json:"companyNamesEditorial"`
	CompanyTickerCodeAuto      []string `json:"companyTickerCodeAuto"`
	CompanyTickerCodeEditorial []string `json:"companyTickerCodeEditorial"`
	ArticleTypes               []string `json:"articleTypes"`
	ArticleBrands              []string `json:"articleBrands"`
	PublishReference           string   `json:"publishReference"`
}

type ContentType struct {
	collection string
	format     string
	category   string
}

var contentTypeMap = map[string]ContentType{
	"article": {
		collection: "FTCom",
		format:     "Articles",
		category:   "article",
	},
	"blogPost": {
		collection: "FTBlogs",
		format:     "Blogs",
		category:   "blogPost",
	},
	"video": {
		collection: "FTVideos",
		format:     "Videos",
		category:   "video",
	},
}

func convertToESContentModel(enrichedContent enrichedContentModel, contentType string) esContentModel {
	esModel := esContentModel{}

	esModel.IndexDate = new(string)
	*esModel.IndexDate = time.Now().UTC().Format("2006-01-02T15:04:05.999Z")

	esModel.ContentType = new(string)
	*esModel.ContentType = contentTypeMap[contentType].category
	esModel.InternalContentType = new(string)
	*esModel.InternalContentType = contentTypeMap[contentType].category
	esModel.Category = new(string)
	*esModel.Category = contentTypeMap[contentType].category
	esModel.Format = new(string)
	*esModel.Format = contentTypeMap[contentType].format

	esModel.UID = &(enrichedContent.Content.UUID)

	esModel.LeadHeadline = new(string)
	*esModel.LeadHeadline = transformText(enrichedContent.Content.Title,
		htmlEntityTransformer,
		tagsRemover,
		outerSpaceTrimmer,
		duplicateWhiteSpaceRemover)

	esModel.Byline = new(string)
	*esModel.Byline = transformText(enrichedContent.Content.Byline,
		htmlEntityTransformer,
		tagsRemover,
		outerSpaceTrimmer,
		duplicateWhiteSpaceRemover)

	if enrichedContent.Content.PublishedDate != "" {
		esModel.LastPublish = &(enrichedContent.Content.PublishedDate)
	}
	if enrichedContent.Content.FirstPublishedDate != "" {
		esModel.InitialPublish = &(enrichedContent.Content.FirstPublishedDate)
	}
	esModel.Body = new(string)

	*esModel.Body = transformText(enrichedContent.Content.Body,
		interactiveGraphicsMarkupTagRemover,
		pullTagTransformer,
		htmlEntityTransformer,
		scriptTagRemover,
		tagsRemover,
		outerSpaceTrimmer,
		embed1Replacer,
		squaredCaptionReplacer,
		duplicateWhiteSpaceRemover)

	if enrichedContent.Content.MainImage != "" {
		esModel.ThumbnailURL = new(string)
		*esModel.ThumbnailURL = strings.Replace(imageServiceURL, imagePlaceholder, enrichedContent.Content.MainImage, -1)
	}

	esModel.URL = new(string)
	*esModel.URL = apiURLPrefix + enrichedContent.Content.UUID

	esModel.PublishReference = enrichedContent.Content.PublishReference

	for _, annotation := range enrichedContent.Metadata {
		tmeID := annotation.Thing.ID
		if len(annotation.Thing.TmeIDs) != 0 {
			tmeID = annotation.Thing.TmeIDs[0]
		} else {
			log.Warnf("Indexing content with uuid %s - TME id missing for concept with id %s, using thing id instead",
				&(enrichedContent.Content.UUID), annotation.Thing.ID)
		}
		for _, taxonomy := range annotation.Thing.Types {
			switch taxonomy {
			case "http://www.ft.com/ontology/organisation/Organisation":
				esModel.CmrOrgnames = append(esModel.CmrOrgnames, annotation.Thing.PrefLabel)
				esModel.CmrOrgnamesIds = append(esModel.CmrOrgnamesIds, getCmrID(tmeOrganisations, tmeID))
			case "http://www.ft.com/ontology/person/Person":
				esModel.CmrPeople = append(esModel.CmrPeople, annotation.Thing.PrefLabel)
				esModel.CmrPeopleIds = append(esModel.CmrPeopleIds, getCmrID(tmePeople, tmeID))
				if annotation.Thing.Predicate == hasAuthor {
					esModel.CmrAuthors = append(esModel.CmrAuthors, annotation.Thing.PrefLabel)
					esModel.CmrAuthorsIds = append(esModel.CmrAuthorsIds, getCmrID(tmeAuthors, tmeID))
				}
			case "http://www.ft.com/ontology/company/Company":
				esModel.CmrCompanynames = append(esModel.CmrCompanynames, annotation.Thing.PrefLabel)
				esModel.CmrCompanynamesIds = append(esModel.CmrCompanynamesIds, getCmrID(tmeOrganisations, tmeID))
			case "http://www.ft.com/ontology/product/Brand":
				esModel.CmrBrands = append(esModel.CmrBrands, annotation.Thing.PrefLabel)
				esModel.CmrBrandsIds = append(esModel.CmrBrandsIds, getCmrID(tmeBrands, tmeID))
			case "http://www.ft.com/ontology/Subject":
				esModel.CmrSubjects = append(esModel.CmrSubjects, annotation.Thing.PrefLabel)
				esModel.CmrSubjectsIds = append(esModel.CmrSubjectsIds, getCmrID(tmeSubjects, tmeID))
			case "http://www.ft.com/ontology/Section":
				esModel.CmrSections = append(esModel.CmrSections, annotation.Thing.PrefLabel)
				esModel.CmrSectionsIds = append(esModel.CmrSectionsIds, getCmrID(tmeSections, tmeID))
				if annotation.Thing.Predicate == primaryClassification {
					esModel.CmrPrimarysection = &(annotation.Thing.PrefLabel)
					esModel.CmrPrimarysectionID = new(string)
					*esModel.CmrPrimarysectionID = getCmrID(tmeSections, tmeID)
				}
			case "http://www.ft.com/ontology/Topic":
				esModel.CmrTopics = append(esModel.CmrTopics, annotation.Thing.PrefLabel)
				esModel.CmrTopicsIds = append(esModel.CmrTopicsIds, getCmrID(tmeTopics, tmeID))
				if annotation.Thing.Predicate == about {
					esModel.CmrPrimarytheme = &(annotation.Thing.PrefLabel)
					esModel.CmrPrimarythemeID = new(string)
					*esModel.CmrPrimarythemeID = getCmrID(tmeTopics, tmeID)
				}
			case "http://www.ft.com/ontology/Location":
				esModel.CmrRegions = append(esModel.CmrRegions, annotation.Thing.PrefLabel)
				esModel.CmrRegionsIds = append(esModel.CmrRegionsIds, getCmrID(tmeRegions, tmeID))
			case "http://www.ft.com/ontology/Genre":
				esModel.CmrGenres = append(esModel.CmrGenres, annotation.Thing.PrefLabel)
				esModel.CmrGenreIds = append(esModel.CmrGenreIds, getCmrID(tmeGenres, tmeID))
			case "http://www.ft.com/ontology/SpecialReport":
				esModel.CmrSpecialreports = append(esModel.CmrSpecialreports, annotation.Thing.PrefLabel)
				esModel.CmrSpecialreportsIds = append(esModel.CmrSpecialreportsIds, getCmrID(tmeSpecialReports, tmeID))
			}
		}
	}
	return esModel
}
func getCmrID(taxonomy string, tmeID string) string {
	return base64.StdEncoding.EncodeToString([]byte(tmeID)) + "-" + base64.StdEncoding.EncodeToString([]byte(taxonomy))
}
