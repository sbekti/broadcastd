package instagram

import (
	"encoding/json"
)

type accountResp struct {
	Status  string  `json:"status"`
	Account Account `json:"logged_in_user"`
}

type Account struct {
	client *Instagram

	ID                         int64        `json:"pk"`
	Username                   string       `json:"username"`
	FullName                   string       `json:"full_name"`
	Biography                  string       `json:"biography"`
	ProfilePicURL              string       `json:"profile_pic_url"`
	Email                      string       `json:"email"`
	PhoneNumber                string       `json:"phone_number"`
	IsBusiness                 bool         `json:"is_business"`
	Gender                     int          `json:"gender"`
	ProfilePicID               string       `json:"profile_pic_id"`
	CanSeeOrganicInsights      bool         `json:"can_see_organic_insights"`
	ShowInsightsTerms          bool         `json:"show_insights_terms"`
	Nametag                    nametag      `json:"nametag"`
	HasAnonymousProfilePicture bool         `json:"has_anonymous_profile_picture"`
	IsPrivate                  bool         `json:"is_private"`
	IsUnpublished              bool         `json:"is_unpublished"`
	AllowedCommenterType       string       `json:"allowed_commenter_type"`
	IsVerified                 bool         `json:"is_verified"`
	MediaCount                 int          `json:"media_count"`
	FollowerCount              int          `json:"follower_count"`
	FollowingCount             int          `json:"following_count"`
	GeoMediaCount              int          `json:"geo_media_count"`
	ExternalURL                string       `json:"external_url"`
	HasBiographyTranslation    bool         `json:"has_biography_translation"`
	ExternalLynxURL            string       `json:"external_lynx_url"`
	HdProfilePicURLInfo        picURLInfo   `json:"hd_profile_pic_url_info"`
	HdProfilePicVersions       []picURLInfo `json:"hd_profile_pic_versions"`
	UsertagsCount              int          `json:"usertags_count"`
	HasChaining                bool         `json:"has_chaining"`
	ReelAutoArchive            string       `json:"reel_auto_archive"`
	PublicEmail                string       `json:"public_email"`
	PublicPhoneNumber          string       `json:"public_phone_number"`
	PublicPhoneCountryCode     string       `json:"public_phone_country_code"`
	ContactPhoneNumber         string       `json:"contact_phone_number"`
	Byline                     string       `json:"byline"`
	SocialContext              string       `json:"social_context,omitempty"`
	SearchSocialContext        string       `json:"search_social_context,omitempty"`
	MutualFollowersCount       float64      `json:"mutual_followers_count"`
	LatestReelMedia            int64        `json:"latest_reel_media,omitempty"`
	CityID                     int64        `json:"city_id"`
	CityName                   string       `json:"city_name"`
	AddressStreet              string       `json:"address_street"`
	DirectMessaging            string       `json:"direct_messaging"`
	Latitude                   float64      `json:"latitude"`
	Longitude                  float64      `json:"longitude"`
	Category                   string       `json:"category"`
	BusinessContactMethod      string       `json:"business_contact_method"`
	IsCallToActionEnabled      bool         `json:"is_call_to_action_enabled"`
	FbPageCallToActionID       string       `json:"fb_page_call_to_action_id"`
	Zip                        string       `json:"zip"`
	AllowContactsSync          bool         `json:"allow_contacts_sync"`
	CanBoostPost               bool         `json:"can_boost_post"`
}

type picURLInfo struct {
	Height int    `json:"height"`
	URL    string `json:"url"`
	Width  int    `json:"width"`
}

type nametag struct {
	Mode          int64       `json:"mode"`
	Gradient      json.Number `json:"gradient,Number"`
	Emoji         string      `json:"emoji"`
	SelfieSticker json.Number `json:"selfie_sticker,Number"`
}

type profResp struct {
	Status  string  `json:"status"`
	Account Account `json:"user"`
}

func (account *Account) Sync() error {
	client := account.client
	data, err := client.prepareData()
	if err != nil {
		return err
	}
	body, err := client.sendRequest(&reqOptions{
		Endpoint: igAPIURLCurrentUser,
		Query:    generateSignature(data),
	})
	if err == nil {
		resp := profResp{}
		err = json.Unmarshal(body, &resp)
		if err == nil {
			*account = resp.Account
			account.client = client
		}
	}
	return err
}
