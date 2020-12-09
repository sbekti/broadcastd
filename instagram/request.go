package instagram

import (
	"bytes"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type reqOptions struct {
	Connection string
	Endpoint   string
	IsPost     bool
	UseV2      bool
	Query      map[string]string
}

var uploadPhotoResponse struct {
	UploadID       string      `json:"upload_id"`
	XsharingNonces interface{} `json:"xsharing_nonces"`
	Status         string      `json:"status"`
}

type HTTPGenericError struct {
	Message   string `json:"message"`
	Status    string `json:"status"`
	ErrorType string `json:"error_type"`
}

func (e HTTPGenericError) Error() string {
	return fmt.Sprintf("%s: %s", e.Status, e.Message)
}

type HTTPError4xx struct {
	ChallengeError
	Action     string `json:"action"`
	StatusCode string `json:"status_code"`
	Payload    struct {
		ClientContext string `json:"client_context"`
		Message       string `json:"message"`
	} `json:"payload"`
	Status string `json:"status"`
}

type ChallengeError struct {
	Message   string `json:"message"`
	Challenge struct {
		URL               string `json:"url"`
		APIPath           string `json:"api_path"`
		HideWebviewHeader bool   `json:"hide_webview_header"`
		Lock              bool   `json:"lock"`
		Logout            bool   `json:"logout"`
		NativeFlow        bool   `json:"native_flow"`
	} `json:"challenge"`
	Status    string `json:"status"`
	ErrorType string `json:"error_type"`
}

func (e ChallengeError) Error() string {
	return fmt.Sprintf("%s: %s (%s)", e.Status, e.Message, e.ErrorType)
}

type LoginRequiredError struct {
	Message      string `json:"message"`
	ErrorTitle   string `json:"error_title"`
	ErrorBody    string `json:"error_body"`
	LogoutReason int    `json:"logout_reason"`
	Status       string `json:"status"`
}

func (e LoginRequiredError) Error() string {
	return fmt.Sprintf("%s: %s", e.Status, e.Message)
}

func (i *Instagram) prepareData(other ...map[string]interface{}) (string, error) {
	data := map[string]interface{}{
		"_uuid":      i.uuid,
		"_csrftoken": i.token,
	}

	if i.Account != nil && i.Account.ID != 0 {
		data["_uid"] = strconv.FormatInt(i.Account.ID, 10)
	}

	for i := range other {
		for key, value := range other[i] {
			data[key] = value
		}
	}

	b, err := json.Marshal(data)
	if err == nil {
		return byteToString(b), err
	}

	return "", err
}

func (i *Instagram) sendRequest(options *reqOptions) (body []byte, err error) {
	method := "GET"
	if options.IsPost {
		method = "POST"
	}

	if options.Connection == "" {
		options.Connection = "keep-alive"
	}

	baseURL := igAPIBaseURL
	if options.UseV2 {
		baseURL = igAPIBaseURLV2
	}

	reqURL, err := url.Parse(baseURL + options.Endpoint)
	if err != nil {
		return nil, err
	}

	values := url.Values{}
	buffer := bytes.NewBuffer([]byte{})

	for k, v := range options.Query {
		values.Add(k, v)
	}

	if options.IsPost {
		buffer.WriteString(values.Encode())
	} else {
		for k, v := range reqURL.Query() {
			values.Add(k, strings.Join(v, " "))
		}

		reqURL.RawQuery = values.Encode()
	}

	var req *http.Request
	req, err = http.NewRequest(method, reqURL.String(), buffer)
	if err != nil {
		return
	}

	req.Header.Set("Connection", options.Connection)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("Accept-Language", "en-US")
	req.Header.Set("User-Agent", igUserAgent)
	req.Header.Set("X-IG-App-ID", fbAnalytics)
	req.Header.Set("X-IG-Capabilities", igCapabilities)
	req.Header.Set("X-IG-Connection-Type", igConnType)
	req.Header.Set("X-IG-Connection-Speed", fmt.Sprintf("%dkbps", getRandom(1000, 3700)))
	req.Header.Set("X-IG-Bandwidth-Speed-KBPS", "-1.000")
	req.Header.Set("X-IG-Bandwidth-TotalBytes-B", "0")
	req.Header.Set("X-IG-Bandwidth-TotalTime-MS", "0")

	log.Tracef("Request %s %s", method, reqURL)
	resp, err := i.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	cookieURL, _ := url.Parse(igAPIBaseURL)
	for _, value := range i.httpClient.Jar.Cookies(cookieURL) {
		if strings.Contains(value.Name, "csrftoken") {
			i.token = value.Value
		}
	}

	body, err = ioutil.ReadAll(resp.Body)
	log.Tracef("Response %s %s: %d: %s", method, reqURL, resp.StatusCode, string(body))
	if err == nil {
		err = checkError(resp.StatusCode, body)
	}

	return body, err
}

func checkError(code int, body []byte) (err error) {
	if code == 200 {
		return nil
	}

	switch code {
	case 400:
		httpErr := &HTTPGenericError{}
		err = json.Unmarshal(body, httpErr)
		if err != nil {
			return err
		}
		if httpErr.Message == "challenge_required" {
			httpErr := &HTTPError4xx{}
			err = json.Unmarshal(body, httpErr)
			if err != nil {
				return err
			}
			return &httpErr.ChallengeError
		}
		return httpErr
	case 403:
		httpErr := &HTTPGenericError{}
		err = json.Unmarshal(body, httpErr)
		if err != nil {
			return err
		}
		if httpErr.Message == "login_required" {
			httpErr := &LoginRequiredError{}
			err = json.Unmarshal(body, httpErr)
			if err != nil {
				return err
			}
			return httpErr
		}
		return httpErr
	default:
		httpErr := &HTTPGenericError{}
		err = json.Unmarshal(body, httpErr)
		if err != nil {
			return err
		}
		return httpErr
	}
}

func (i *Instagram) UploadPhoto(photo io.Reader) (string, error) {
	uploadID := time.Now().Unix()
	rndNumber := rand.Intn(9999999999-1000000000) + 1000000000
	name := strconv.FormatInt(uploadID, 10) + "_0_" + strconv.Itoa(rndNumber)

	inBuffer := new(bytes.Buffer)
	_, err := inBuffer.ReadFrom(photo)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", igBaseURL+igAPIUploadPhoto+name, inBuffer)
	if err != nil {
		return "", err
	}

	req.Header.Set("Connection", "close")
	req.Header.Set("Content-type", "application/octet-stream")
	req.Header.Set("Accept-Language", "en-US")
	req.Header.Set("User-Agent", igUserAgent)
	req.Header.Set("Cookie2", "$Version=1")
	req.Header.Set("Offset", "0")
	req.Header.Set("X-IG-App-ID", fbAnalytics)
	req.Header.Set("X-IG-Capabilities", igCapabilities)
	req.Header.Set("X-IG-Connection-Type", igConnType)
	req.Header.Set("X-Entity-Name", name)

	ruploadParams := map[string]string{
		"retry_context":     `{"num_step_auto_retry": 0, "num_reupload": 0, "num_step_manual_retry": 0}`,
		"media_type":        "1",
		"upload_id":         strconv.FormatInt(uploadID, 10),
		"xsharing_user_ids": "[]",
		"image_compression": `{"lib_name": "moz", "lib_version": "3.1.m", "quality": "80"}`,
	}
	params, err := json.Marshal(ruploadParams)
	if err != nil {
		return "", err
	}
	req.Header.Set("X-Instagram-Rupload-Params", string(params))
	req.Header.Set("X-Entity-Length", strconv.FormatInt(req.ContentLength, 10))

	resp, err := i.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("upload photo: failed, status code, result: %s", resp.Status)
	}

	err = json.Unmarshal(body, &uploadPhotoResponse)
	if err != nil {
		return "", err
	}

	if uploadPhotoResponse.Status != "ok" {
		return "", fmt.Errorf("upload photo: unknown error, status: %s", uploadPhotoResponse.Status)
	}

	return strconv.FormatInt(uploadID, 10), nil
}
