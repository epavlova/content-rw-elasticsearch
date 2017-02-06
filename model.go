package main

type enrichedContentModel struct {
	Content  contentModel `json:"content"`
	Metadata annotations  `json:"metadata"`
}

type contentModel struct {
	Uuid              string            `json:"uuid"`
	Title             string            `json:"title"`
	Type              string            `json:"type"`
	AlternativeTitles alternativeTitles `json:"alternativeTitles"`

	Byline string  `json:"byline"`
	Brands []brand `json:"brands"`

	Identifiers []identifier `json:"identifiers"`

	PublishedDate     string   `json:"publishedDate"`
	FirstPublishedDate     string   `json:"firstPublishedDate"`
	Standfirst        string   `json:"standfirst"`
	Body              string   `json:"body"`
	Description       string   `json:"description"`
	MediaType         string   `json:"mediaType"`
	PixelWidth        string   `json:"pixelWidth"`
	PixelHeight       string   `json:"pixelHeight"`
	InternalBinaryUrl string   `json:"internalBinaryUrl"`
	ExternalBinaryUrl string   `json:"externalBinaryUrl"`
	Members           string   `json:"members"`
	MainImage         string   `json:"mainImage"`
	Standout          standout `json:"standout"`
	Comments          comments `json:"comments"`
	Copyright         string   `json:"copyright"`
	WebUrl            string   `json:"webUrl"`
	PublishReference  string   `json:"publishReference"`
	LastModified      string   `json:"lastModified"`
	CanBeSyndicated   string   `json:"canBeSyndicated"`
}

type alternativeTitles struct {
	PromotionalTitle string `json:"promotionalTitle"`
}

type brand struct {
	Id string `json:"id"`
}

type identifier struct {
	Authority       string `json:"authority"`
	IdentifierValue string `json:"identifierValue"`
}

type standout struct {
	EditorsChoice bool `json:"editorsChoice"`
	Exclusive     bool `json:"exclusive"`
	Scoop         bool `json:"scoop"`
}

type comments struct {
	Enabled bool `json:"enabled"`
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
}

type esContentModel struct {
	// todo fix types
	Uid                   string   `json:"uid"`
	LastMetadataPublish   string   `json:"last_metadata_publish"`
	IndexDate             string   `json:"index_date"`
	MarkDeleted           bool     `json:"mark_deleted"`
	StoryId               int32    `json:"story_id"`
	LeadHeadline          string   `json:"lead_headline"`
	Byline                string   `json:"byline"`
	Body                  string   `json:"body"`
	Url                   string   `json:"url"`
	InitialPublish        string   `json:"initial_publish"`
	LastPublish           string   `json:"last_publish"`
	ContentType           string   `json:"content_type"`
	ProviderName          string   `json:"provider_name"`
	LengthMillis          int32    `json:"length_millis"`
	ShortDescription      string   `json:"short_description"`
	ThumbnailUrl          string   `json:"thumbnail_url"`
	SectionLink           string   `json:"section_link"`
	SecondaryImageId      string   `json:"secondary_image_id"`
	ContributorRights     string   `json:"contributor_rights"`
	SourceCode            string   `json:"source_code"`
	StorymodelId          int32    `json:"storymodel_id"`
	ModelApiUrl           string   `json:"model_api_url"`
	ModelMasterSource     string   `json:"model_master_source"`
	ModelMasterId         string   `json:"model_master_id"`
	ModelExcerpt          string   `json:"model_excerpt"`
	ModelResourceUri      string   `json:"model_resource_uri"`
	CmrPrimarysection     string   `json:"cmr_primarysection"`
	CmrPrimarytheme       string   `json:"cmr_primarytheme"`
	CmrMediatype          string   `json:"cmr_mediatype"`
	CmrMetadataupdatetime string   `json:"cmr_metadataupdatetime"`
	CmrPrimarysectionId   string   `json:"cmr_primarysection_id"`
	CmrPrimarythemeId     string   `json:"cmr_primarytheme_id"`
	CmrMediatypeId        string   `json:"cmr_mediatype_id"`
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
	InternalContentType string   `json:"internalContentType"`
	Category            string   `json:"category"`
	LookupFailure       bool     `json:"lookupFailure"`
	Format              string   `json:"format"`
	CmrGenre            []string `json:"cmr_genre"`

	CmrGenreId []string `json:"cmr_genre_id"`

	Region           string   `json:"region"`
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
}

type ContentType struct {
	collection string
	format     string
	category   string
}

var m = map[string]ContentType{
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

	//todo field transform
	esModel := esContentModel{}

	esModel.ContentType = m[contentType].category
	esModel.Format = m[contentType].format

	esModel.Uid = enrichedContent.Content.Uuid
	esModel.LeadHeadline = enrichedContent.Content.Title
	esModel.Byline = enrichedContent.Content.Byline
	esModel.LastPublish = enrichedContent.Content.PublishedDate
	esModel.InitialPublish = enrichedContent.Content.FirstPublishedDate
	esModel.Body = enrichedContent.Content.Body
	//esModel.ShortDescription = enrichedContent.Content.Description       string        `json:"description"`
	//todo figure out thumbnail source
	//esModel. = enrichedContent.Content.MainImage         string        `json:"mainImage"`
	esModel.Url = "https://www.ft.com/content/" + enrichedContent.Content.Uuid

	//todo are UPP ids good enough?
	for _, annotation := range enrichedContent.Metadata {
		for _, taxonomy := range annotation.Thing.Types {
			switch taxonomy {
			case "http://www.ft.com/ontology/organisation/Organisation":
				esModel.CmrOrgnames = append(esModel.CmrOrgnames, annotation.Thing.PrefLabel)
				esModel.CmrOrgnamesIds = append(esModel.CmrOrgnamesIds, annotation.Thing.ID)
			case "http://www.ft.com/ontology/person/Person":
				esModel.CmrPeople = append(esModel.CmrPeople, annotation.Thing.PrefLabel)
				esModel.CmrPeopleIds = append(esModel.CmrPeopleIds, annotation.Thing.ID)
				if annotation.Thing.Predicate == "hasAuthor" {
					esModel.CmrAuthors = append(esModel.CmrAuthors, annotation.Thing.PrefLabel)
					esModel.CmrAuthorsIds = append(esModel.CmrAuthorsIds, annotation.Thing.ID)
				}
			case "http://www.ft.com/ontology/company/Company":
				//todo make sure we get annotations in this taxo
				esModel.CmrCompanynames = append(esModel.CmrCompanynames, annotation.Thing.PrefLabel)
				esModel.CmrCompanynamesIds = append(esModel.CmrCompanynamesIds, annotation.Thing.ID)
			case "http://www.ft.com/ontology/product/Brand":
				esModel.CmrBrands = append(esModel.CmrBrands, annotation.Thing.PrefLabel)
				esModel.CmrBrandsIds = append(esModel.CmrBrandsIds, annotation.Thing.ID)
			case "http://www.ft.com/ontology/Subject":
				esModel.CmrSubjects = append(esModel.CmrSubjects, annotation.Thing.PrefLabel)
				esModel.CmrSubjectsIds = append(esModel.CmrSubjectsIds, annotation.Thing.ID)
			case "http://www.ft.com/ontology/Section":
				esModel.CmrSections = append(esModel.CmrSections, annotation.Thing.PrefLabel)
				esModel.CmrSectionsIds = append(esModel.CmrSectionsIds, annotation.Thing.ID)
				if annotation.Thing.Predicate == "isPrimarilyClassifiedBy" {
					esModel.CmrPrimarysection = annotation.Thing.PrefLabel
					esModel.CmrPrimarysectionId = annotation.Thing.ID
				}
			case "http://www.ft.com/ontology/Topic":
				esModel.CmrTopics = append(esModel.CmrTopics, annotation.Thing.PrefLabel)
				esModel.CmrTopicsIds = append(esModel.CmrTopicsIds, annotation.Thing.ID)
				if annotation.Thing.Predicate == "about" {
					esModel.CmrPrimarytheme = annotation.Thing.PrefLabel
					esModel.CmrPrimarythemeId = annotation.Thing.ID
				}
			case "http://www.ft.com/ontology/Location":
				esModel.CmrRegions = append(esModel.CmrRegions, annotation.Thing.PrefLabel)
				esModel.CmrRegionsIds = append(esModel.CmrRegionsIds, annotation.Thing.ID)
			case "http://www.ft.com/ontology/Genre":
				esModel.CmrGenre = annotation.Thing.PrefLabel
				esModel.CmrGenreId = annotation.Thing.ID
			case "http://www.ft.com/ontology/SpecialReport":
				esModel.CmrSpecialreports = append(esModel.CmrSpecialreports, annotation.Thing.PrefLabel)
				esModel.CmrSpecialreportsIds = append(esModel.CmrSpecialreportsIds, annotation.Thing.ID)
			}
		}
	}

	return esModel
}
