package main

type enrichedContentModel struct {
	Content  contentModel   `json:"content"`
	Metadata metadataModel   `json:"metadata"`
}

type contentModel struct {
	Uuid              string        `json:"uuid"`
	Title             string        `json:"title"`
	Type              string        `json:"type"`
	AlternativeTitles alternativeTitles        `json:"alternativeTitles"`

	Byline            string        `json:"byline"`
	Brands            []brand        `json:"brands"`

	Identifiers       []identifier        `json:"identifiers"`

	PublishedDate     string        `json:"publishedDate"`
	Standfirst        string        `json:"standfirst"`
	Body              string        `json:"body"`
	Description       string        `json:"description"`
	MediaType         string        `json:"mediaType"`
	PixelWidth        string        `json:"pixelWidth"`
	PixelHeight       string        `json:"pixelHeight"`
	InternalBinaryUrl string        `json:"internalBinaryUrl"`
	ExternalBinaryUrl string        `json:"externalBinaryUrl"`
	Members           string        `json:"members"`
	MainImage         string        `json:"mainImage"`
	Standout          standout        `json:"standout"`
	Comments          comments        `json:"comments"`
	Copyright         string        `json:"copyright"`
	WebUrl            string        `json:"webUrl"`
	PublishReference  string        `json:"publishReference"`
	LastModified      string        `json:"lastModified"`
	CanBeSyndicated   string        `json:"canBeSyndicated"`
}

type alternativeTitles struct {
	PromotionalTitle string        `json:"promotionalTitle"`
}

type brand struct {
	Id string        `json:"id"`
}

type identifier struct {
	Authority       string        `json:"authority"`
	IdentifierValue string        `json:"identifierValue"`
}

type standout struct {
	EditorsChoice bool        `json:"editorsChoice"`
	Exclusive     bool        `json:"exclusive"`
	Scoop         bool        `json:"scoop"`
}

type comments struct {
	Enabled bool        `json:"enabled"`
}

type metadataModel struct {

}

type esContentModel struct {
	// todo fix types
	Uid                        string        `json:"uid"`
	LastMetadataPublish        string        `json:"last_metadata_publish"`
	IndexDate                  string        `json:"index_date"`
	MarkDeleted                bool        `json:"mark_deleted"`
	StoryId                    int32        `json:"story_id"`
	LeadHeadline               string        `json:"lead_headline"`
	Byline                     string        `json:"byline"`
	Body                       string        `json:"body"`
	Url                        string        `json:"url"`
	InitialPublish             string        `json:"initial_publish"`
	LastPublish                string        `json:"last_publish"`
	ContentType                string        `json:"content_type"`
	ProviderName               string        `json:"provider_name"`
	LengthMillis               int32        `json:"length_millis"`
	ShortDescription           string        `json:"short_description"`
	ThumbnailUrl               string        `json:"thumbnail_url"`
	SectionLink                string        `json:"section_link"`
	SecondaryImageId           string        `json:"secondary_image_id"`
	ContributorRights          string        `json:"contributor_rights"`
	SourceCode                 string        `json:"source_code"`
	StorymodelId               int32        `json:"storymodel_id"`
	ModelApiUrl                string        `json:"model_api_url"`
	ModelMasterSource          string        `json:"model_master_source"`
	ModelMasterId              string        `json:"model_master_id"`
	ModelExcerpt               string        `json:"model_excerpt"`
	ModelResourceUri           string        `json:"model_resource_uri"`
	CmrPrimarysection          string        `json:"cmr_primarysection"`
	CmrPrimarytheme            string        `json:"cmr_primarytheme"`
	CmrMediatype               string        `json:"cmr_mediatype"`
	CmrMetadataupdatetime      string        `json:"cmr_metadataupdatetime"`
	CmrPrimarysectionId        string        `json:"cmr_primarysection_id"`
	CmrPrimarythemeId          string        `json:"cmr_primarytheme_id"`
	CmrMediatypeId             string        `json:"cmr_mediatype_id"`
	CmrBrands                  string        `json:"cmr_brands"`
	CmrBrandsIds               string        `json:"cmr_brands_ids"`
	CmrSpecialreports          string        `json:"cmr_specialreports"`
	CmrSpecialreportsIds       string        `json:"cmr_specialreports_ids"`
	CmrSections                string        `json:"cmr_sections"`

	CmrSectionsIds             string        `json:"cmr_sections_ids"`

	CmrSubjects                string        `json:"cmr_subjects"`
	CmrSubjectsIds             string        `json:"cmr_subjects_ids"`
	CmrTopics                  string        `json:"cmr_topics"`
	CmrTopicsIds               string        `json:"cmr_topics_ids"`
	CmrPeople                  string        `json:"cmr_people"`

	CmrPeopleIds               string        `json:"cmr_people_ids"`

	CmrRegions                 string        `json:"cmr_regions"`

	CmrRegionsIds              string        `json:"cmr_regions_ids"`

	CmrIcb                     string        `json:"cmr_icb"`
	CmrIcbIds                  string        `json:"cmr_icb_ids"`
	CmrIptc                    string        `json:"cmr_iptc"`

	CmrIptcIds                 string        `json:"cmr_iptc_ids"`

	CmrAuthorsIds              string        `json:"cmr_authors_ids"`

	CmrAuthors                 string        `json:"cmr_authors"`

	CmrCompanynames            string        `json:"cmr_companynames"`
	CmrCompanynamesIds         string        `json:"cmr_companynames_ids"`
	CmrOrgnames                string        `json:"cmr_orgnames"`

	CmrOrgnamesIds             string        `json:"cmr_orgnames_ids"`

	BestStory                  bool        `json:"bestStory"`
	InternalContentType        string        `json:"internalContentType"`
	Category                   string        `json:"category"`
	LookupFailure              bool        `json:"lookupFailure"`
	Format                     string        `json:"format"`
	CmrGenre                   string        `json:"cmr_genre"`

	CmrGenreId                 string        `json:"cmr_genre_id"`

	Region                     string        `json:"region"`
	Topics                     string        `json:"topics"`
	DisplayCodes               string        `json:"displayCodes"`

	DisplayCodeNames           string        `json:"displayCodeNames"`

	NaicsNames                 string        `json:"naicsNames"`
	EditorsTags                string        `json:"editorsTags"`
	CountryCodes               string        `json:"countryCodes"`
	CountryNames               string        `json:"countryNames"`
	Subjects                   string        `json:"subjects"`
	CompanyNamesAuto           string        `json:"companyNamesAuto"`
	OrganisationNamesAuto      string        `json:"organisationNamesAuto"`
	CompanyNamesEditorial      string        `json:"companyNamesEditorial"`
	CompanyTickerCodeAuto      string        `json:"companyTickerCodeAuto"`
	CompanyTickerCodeEditorial string        `json:"companyTickerCodeEditorial"`
	ArticleTypes               string        `json:"articleTypes"`

	ArticleBrands              string        `json:"articleBrands"`
}

type ContentType struct {
	collection string
	format     string
	category   string
}

var m = map[string]ContentType{
	"article" : {
		collection:"FTCom",
		format:  "Articles",
		category   :"article",
	},
	"blogPost" : {
		collection:"FTBlogs",
		format:  "Blogs",
		category   :"blogPost",
	},
	"video" : {
		collection:"FTVideos",
		format:  "Videos",
		category   :"video",
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
	//esModel. = enrichedContent.Content.Brands            []brand        `json:"brands"`
	//esModel. = enrichedContent.Content.Identifiers       []identifier        `json:"identifiers"`
	esModel.LastPublish = enrichedContent.Content.PublishedDate
	//esModel.InitialPublish = todo
	//esModel. = enrichedContent.Content.Standfirst        string        `json:"standfirst"`
	esModel.Body = enrichedContent.Content.Body
	//esModel.ShortDescription = enrichedContent.Content.Description       string        `json:"description"`
	//esModel. = enrichedContent.Content.MediaType         string        `json:"mediaType"`
	//todo figure out thumbnail source
	//esModel. = enrichedContent.Content.MainImage         string        `json:"mainImage"`
	esModel.Url = "https://www.ft.com/content/" + enrichedContent.Content.Uuid
	//enrichedContent.Content.WebUrl            string        `json:"webUrl"`
	//esModel. = enrichedContent.Content.LastModified
	return esModel
}

