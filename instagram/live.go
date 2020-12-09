package instagram

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
)

type Live struct {
	client *Instagram
}

func newLive(i *Instagram) *Live {
	return &Live{
		client: i,
	}
}

type LiveCreateResponse struct {
	BroadcastID int    `json:"broadcast_id"`
	UploadURL   string `json:"upload_url"`
	Status      string `json:"status"`
}

type LiveStartResponse struct {
	MediaID string `json:"media_id"`
	Status  string `json:"status"`
}

type LiveEndResponse struct {
	Status string `json:"status"`
}

type LiveAddToPostResponse struct {
	Status string `json:"status"`
}

type LiveUnmuteCommentResponse struct {
	CommentMuted int    `json:"comment_muted"`
	Status       string `json:"status"`
}

type LiveCommentResponse struct {
	Comment struct {
		PK int64 `json:"pk"`
	} `json:"comment"`
	Status string `json:"status"`
}

type LivePinCommentResponse struct {
	Status string `json:"status"`
}

type LiveDisableRequestToJoinResponse struct {
	Status string `json:"status"`
}

type LiveInfoResponse struct {
	ID                             int     `json:"id"`
	RTMPPlaybackURL                string  `json:"rtmp_playback_url"`
	DASHPlaybackURL                string  `json:"dash_playback_url"`
	DASHABRPlaybackURL             string  `json:"dash_abr_playback_url"`
	DASHPLivePredictivePlaybackURL string  `json:"dash_live_predictive_playback_url"`
	BroadcastStatus                string  `json:"broadcast_status"`
	ViewerCount                    float64 `json:"viewer_count"`
	InternalOnly                   bool    `json:"internal_only"`
	PublishedTime                  int     `json:"published_time"`
	HideFromFeedUnit               bool    `json:"hide_from_feed_unit"`
	MediaID                        string  `json:"media_id"`
	BroadcastMessage               string  `json:"broadcast_message"`
	Status                         string  `json:"status"`
}

type LiveHeartbeatAndGetViewerCountResponse struct {
	ViewerCount            float64 `json:"viewer_count"`
	BroadcastStatus        string  `json:"broadcast_status"`
	OffsetToVideoStart     int     `json:"offset_to_video_start"`
	TotalUniqueViewerCount int     `json:"total_unique_viewer_count"`
	IsTopLiveEligible      int     `json:"is_top_live_eligible"`
	Status                 string  `json:"status"`
}

type LiveGetCommentResponse struct {
	CommentLikesEnabled        bool          `json:"comment_likes_enabled"`
	Comments                   []LiveComment `json:"comments"`
	CommentCount               int           `json:"comment_count"`
	Caption                    interface{}   `json:"caption"`
	CaptionIsEdited            bool          `json:"caption_is_edited"`
	HasMoreComments            bool          `json:"has_more_comments"`
	HasMoreHeadloadComments    bool          `json:"has_more_headload_comments"`
	MediaHeaderDisplay         string        `json:"media_header_display"`
	CanViewMorePreviewComments bool          `json:"can_view_more_preview_comments"`
	LiveSecondsPerComment      int           `json:"live_seconds_per_comment"`
	IsFirstFetch               string        `json:"is_first_fetch"`
	SystemComments             interface{}   `json:"system_comments"`
	CommentMuted               int           `json:"comment_muted"`
	Status                     string        `json:"status"`
}

type LiveComment struct {
	PK              int64  `json:"pk"`
	UserID          int    `json:"user_id"`
	Text            string `json:"text"`
	Type            int    `json:"type"`
	CreatedAt       int    `json:"created_at"`
	CreatedAtUtc    int    `json:"created_at_utc"`
	ContentType     string `json:"content_type"`
	Status          string `json:"status"`
	BitFlags        int    `json:"bit_flags"`
	DidReportAsSpam bool   `json:"did_report_as_spam"`
	ShareEnabled    bool   `json:"share_enabled"`
	User            struct {
		PK                  int64  `json:"pk"`
		Username            string `json:"username"`
		FullName            string `json:"full_name"`
		IsPrivate           bool   `json:"is_private"`
		ProfilePicURL       string `json:"profile_pic_url"`
		ProfilePicID        string `json:"profile_pic_id"`
		IsVerified          bool   `json:"is_verified"`
		LiveWithEligibility string `json:"live_with_eligibility"`
	} `json:"user"`
}

type LiveGetPostLiveThumbnailsResponse struct {
	Thumbnails []string `json:"thumbnails"`
	Status     string   `json:"status"`
}

type LiveAddPostLiveToIGTVResponse struct {
	Success    bool   `json:"success"`
	IGTVPostID int64  `json:"igtv_post_id"`
	Status     string `json:"status"`
}

type LiveGetFinalViewerListResponse struct {
	Users                  []LiveViewerUser `json:"users"`
	TotalUniqueViewerCount int              `json:"total_unique_viewer_count"`
	Status                 string           `json:"status"`
}

type LiveViewerUser struct {
	PK            int64  `json:"pk"`
	Username      string `json:"username"`
	FullName      string `json:"full_name"`
	IsPrivate     bool   `json:"is_private"`
	ProfilePicURL string `json:"profile_pic_url"`
	ProfilePicID  string `json:"profile_pic_id"`
	IsVerified    bool   `json:"is_verified"`
}

func (live *Live) Create(width int, height int, message string) (*LiveCreateResponse, error) {
	client := live.client

	data, err := client.prepareData(
		map[string]interface{}{
			"preview_width":     width,
			"preview_height":    height,
			"broadcast_message": message,
			"broadcast_type":    "RTMP",
			"internal_only":     0,
		},
	)
	if err != nil {
		return nil, err
	}

	body, err := client.sendRequest(
		&reqOptions{
			Endpoint: igAPILiveCreate,
			IsPost:   true,
			Query:    generateSignature(data),
		},
	)
	if err != nil {
		return nil, err
	}

	res := &LiveCreateResponse{}
	err = json.Unmarshal(body, res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (live *Live) Start(broadcastID int, notify bool) (*LiveStartResponse, error) {
	client := live.client

	data, err := client.prepareData(
		map[string]interface{}{
			"should_send_notifications": notify,
		},
	)
	if err != nil {
		return nil, err
	}

	body, err := client.sendRequest(
		&reqOptions{
			Endpoint: fmt.Sprintf(igAPILiveStart, broadcastID),
			IsPost:   true,
			Query:    generateSignature(data),
		},
	)
	if err != nil {
		return nil, err
	}

	res := &LiveStartResponse{}
	err = json.Unmarshal(body, res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (live *Live) End(broadcastID int, endAfterCopyrightWarning bool) (*LiveEndResponse, error) {
	client := live.client

	data, err := client.prepareData(
		map[string]interface{}{
			"end_after_copyright_warning": endAfterCopyrightWarning,
		},
	)
	if err != nil {
		return nil, err
	}

	body, err := client.sendRequest(
		&reqOptions{
			Endpoint: fmt.Sprintf(igAPILiveEnd, broadcastID),
			IsPost:   true,
			Query:    generateSignature(data),
		},
	)
	if err != nil {
		return nil, err
	}

	res := &LiveEndResponse{}
	err = json.Unmarshal(body, res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (live *Live) Info(broadcastID int) (*LiveInfoResponse, error) {
	client := live.client

	data, err := client.prepareData(
		map[string]interface{}{},
	)
	if err != nil {
		return nil, err
	}

	body, err := client.sendRequest(
		&reqOptions{
			Endpoint: fmt.Sprintf(igAPILiveInfo, broadcastID),
			IsPost:   false,
			Query:    generateSignature(data),
		},
	)
	if err != nil {
		return nil, err
	}

	res := &LiveInfoResponse{}
	err = json.Unmarshal(body, res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (live *Live) UnmuteComment(broadcastID int) (*LiveUnmuteCommentResponse, error) {
	client := live.client

	data, err := client.prepareData(
		map[string]interface{}{},
	)
	if err != nil {
		return nil, err
	}

	body, err := client.sendRequest(
		&reqOptions{
			Endpoint: fmt.Sprintf(igAPILiveUnmuteComment, broadcastID),
			IsPost:   true,
			Query:    generateSignature(data),
		},
	)
	if err != nil {
		return nil, err
	}

	res := &LiveUnmuteCommentResponse{}
	err = json.Unmarshal(body, res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (live *Live) DisableRequestToJoin(broadcastID int) (*LiveDisableRequestToJoinResponse, error) {
	client := live.client

	data, err := client.prepareData(
		map[string]interface{}{},
	)
	if err != nil {
		return nil, err
	}

	body, err := client.sendRequest(
		&reqOptions{
			Endpoint: fmt.Sprintf(igAPILiveDisableRequestToJoin, broadcastID),
			IsPost:   true,
			Query:    generateSignature(data),
		},
	)
	if err != nil {
		return nil, err
	}

	res := &LiveDisableRequestToJoinResponse{}
	err = json.Unmarshal(body, res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (live *Live) GetComment(broadcastID int, numCommentsRequested int, lastCommentTS int) (*LiveGetCommentResponse, error) {
	client := live.client

	data, err := client.prepareData(
		map[string]interface{}{
			"num_comments_requested": numCommentsRequested,
			"last_comment_ts":        lastCommentTS,
		},
	)

	body, err := client.sendRequest(
		&reqOptions{
			Endpoint: fmt.Sprintf(igAPILiveGetComment, broadcastID),
			Query:    generateSignature(data),
			IsPost:   false,
		},
	)
	if err != nil {
		return nil, err
	}

	res := &LiveGetCommentResponse{}
	err = json.Unmarshal(body, res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (live *Live) Comment(broadcastID int, message string) (*LiveCommentResponse, error) {
	client := live.client

	data, err := client.prepareData(
		map[string]interface{}{
			"user_breadcrumb":       generateBreadcrumb(len(message)),
			"idempotence_token":     uuid.New(),
			"comment_text":          message,
			"live_or_vod":           1,
			"offset_to_video_start": 0,
		},
	)
	if err != nil {
		return nil, err
	}

	body, err := client.sendRequest(
		&reqOptions{
			Endpoint: fmt.Sprintf(igAPILiveComment, broadcastID),
			IsPost:   true,
			Query:    generateSignature(data),
		},
	)
	if err != nil {
		return nil, err
	}

	res := &LiveCommentResponse{}
	err = json.Unmarshal(body, res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (live *Live) PinComment(broadcastID int, commentID int64) (*LivePinCommentResponse, error) {
	client := live.client

	data, err := client.prepareData(
		map[string]interface{}{
			"offset_to_video_start": 0,
			"comment_id":            commentID,
		},
	)
	if err != nil {
		return nil, err
	}

	body, err := client.sendRequest(
		&reqOptions{
			Endpoint: fmt.Sprintf(igAPILivePinComment, broadcastID),
			IsPost:   true,
			Query:    generateSignature(data),
		},
	)
	if err != nil {
		return nil, err
	}

	res := &LivePinCommentResponse{}
	err = json.Unmarshal(body, res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (live *Live) HeartbeatAndGetViewerCount(broadcastID int) (*LiveHeartbeatAndGetViewerCountResponse, error) {
	client := live.client

	data, err := client.prepareData(
		map[string]interface{}{
			"offset_to_video_start": 0,
		},
	)
	if err != nil {
		return nil, err
	}

	body, err := client.sendRequest(
		&reqOptions{
			Endpoint: fmt.Sprintf(igAPILiveHeartbeatAndGetViewerCount, broadcastID),
			IsPost:   true,
			Query:    generateSignature(data),
		},
	)
	if err != nil {
		return nil, err
	}

	res := &LiveHeartbeatAndGetViewerCountResponse{}
	err = json.Unmarshal(body, res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (live *Live) GetPostLiveThumbnails(broadcastID int) (*LiveGetPostLiveThumbnailsResponse, error) {
	client := live.client

	data, err := client.prepareData(
		map[string]interface{}{},
	)

	body, err := client.sendRequest(
		&reqOptions{
			Endpoint: fmt.Sprintf(igAPILiveGetPostLiveThumbnails, broadcastID),
			Query:    generateSignature(data),
			IsPost:   false,
		},
	)
	if err != nil {
		return nil, err
	}

	res := &LiveGetPostLiveThumbnailsResponse{}
	err = json.Unmarshal(body, res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (live *Live) AddPostLiveToIGTV(broadcastID int, coverUploadID string, title string, description string, sharePreviewToFeed bool) (*LiveAddPostLiveToIGTVResponse, error) {
	client := live.client

	data, err := client.prepareData(
		map[string]interface{}{
			"broadcast_id":               broadcastID,
			"cover_upload_id":            coverUploadID,
			"description":                description,
			"title":                      title,
			"internal_only":              false,
			"igtv_share_preview_to_feed": sharePreviewToFeed,
		},
	)
	if err != nil {
		return nil, err
	}

	body, err := client.sendRequest(
		&reqOptions{
			Endpoint: igAPILiveAddPostLiveToIGTV,
			IsPost:   true,
			Query:    generateSignature(data),
		},
	)
	if err != nil {
		return nil, err
	}

	res := &LiveAddPostLiveToIGTVResponse{}
	err = json.Unmarshal(body, res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (live *Live) GetFinalViewerList(broadcastID int) (*LiveGetFinalViewerListResponse, error) {
	client := live.client

	data, err := client.prepareData(
		map[string]interface{}{},
	)

	body, err := client.sendRequest(
		&reqOptions{
			Endpoint: fmt.Sprintf(igAPILiveGetFinalViewerList, broadcastID),
			Query:    generateSignature(data),
			IsPost:   false,
		},
	)
	if err != nil {
		return nil, err
	}

	res := &LiveGetFinalViewerListResponse{}
	err = json.Unmarshal(body, res)
	if err != nil {
		return nil, err
	}

	return res, nil
}
