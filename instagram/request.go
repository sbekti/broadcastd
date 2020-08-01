package instagram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type reqOptions struct {
	Connection string
	Endpoint   string
	IsPost     bool
	UseV2      bool
	Query      map[string]string
}

type HTTPErrorGeneric struct {
	Message   string `json:"message"`
	Status    string `json:"status"`
	ErrorType string `json:"error_type"`
}

func (e HTTPErrorGeneric) Error() string {
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

	resp, err := i.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	reqURL, _ = url.Parse(igAPIBaseURL)
	for _, value := range i.httpClient.Jar.Cookies(reqURL) {
		if strings.Contains(value.Name, "csrftoken") {
			i.token = value.Value
		}
	}

	body, err = ioutil.ReadAll(resp.Body)
	if err == nil {
		err = checkError(resp.StatusCode, body)
	}

	return body, err
}

func checkError(code int, body []byte) (err error) {
	switch code {
	case 200:
	case 400:
		httpErr := HTTPError4xx{}
		err = json.Unmarshal(body, &httpErr)
		if err != nil {
			return err
		}

		if httpErr.Message == "challenge_required" {
			return httpErr.ChallengeError
		}

		return httpErr
	default:
		httpErr := HTTPErrorGeneric{}
		err = json.Unmarshal(body, &httpErr)
		if err != nil {
			return err
		}

		return httpErr
	}

	return nil
}
