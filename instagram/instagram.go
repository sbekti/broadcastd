package instagram

import (
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"strconv"
)

type Instagram struct {
	username     string
	password     string
	deviceID     string
	uuid         string
	phoneID      string
	adID         string
	rankToken    string
	token        string
	challengeURL string
	httpClient   *http.Client

	Account   *Account
	Live      *Live
	Challenge *Challenge
}

type LogoutResponse struct {
	Status string `json:"status"`
}

func New(username, password string) *Instagram {
	jar, _ := cookiejar.New(nil)
	client := &Instagram{
		username: username,
		password: password,
		deviceID: generateDeviceID(
			generateMD5Hash(username + password),
		),
		uuid:    generateUUID(),
		phoneID: generateUUID(),
		httpClient: &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
			},
			Jar: jar,
		},
	}

	client.init()

	return client
}

func (i *Instagram) init() {
	i.Live = newLive(i)
	i.Challenge = newChallenge(i)
}

func (i *Instagram) readMSISDNHeader() error {
	data, err := json.Marshal(
		map[string]string{
			"device_id": i.uuid,
		},
	)
	if err != nil {
		return err
	}

	_, err = i.sendRequest(
		&reqOptions{
			Endpoint:   igAPIURLReadMSISDNHeader,
			IsPost:     true,
			Connection: "keep-alive",
			Query:      generateSignature(byteToString(data)),
		},
	)
	return err
}

func (i *Instagram) syncFeatures() error {
	data, err := i.prepareData(
		map[string]interface{}{
			"id":          i.uuid,
			"experiments": igExperiments,
		},
	)
	if err != nil {
		return err
	}

	_, err = i.sendRequest(
		&reqOptions{
			Endpoint: igAPIURLQeSync,
			Query:    generateSignature(data),
			IsPost:   true,
		},
	)
	return err
}

func (i *Instagram) zrToken() error {
	_, err := i.sendRequest(
		&reqOptions{
			Endpoint:   igAPIURLZrToken,
			IsPost:     false,
			Connection: "keep-alive",
			Query: map[string]string{
				"device_id":        i.deviceID,
				"token_hash":       "",
				"custom_device_id": i.uuid,
				"fetch_reason":     "token_expired",
			},
		},
	)
	return err
}

func (i *Instagram) sendAdID() error {
	data, err := i.prepareData(
		map[string]interface{}{
			"adid": i.adID,
		},
	)
	if err != nil {
		return err
	}

	_, err = i.sendRequest(
		&reqOptions{
			Endpoint:   igAPIURLLogAttribution,
			IsPost:     true,
			Connection: "keep-alive",
			Query:      generateSignature(data),
		},
	)
	return err
}

func (i *Instagram) contactPrefill() error {
	data, err := json.Marshal(
		map[string]string{
			"phone_id":   i.phoneID,
			"_csrftoken": i.token,
			"usage":      "prefill",
		},
	)
	if err != nil {
		return err
	}
	_, err = i.sendRequest(
		&reqOptions{
			Endpoint:   igAPIURLContactPrefill,
			IsPost:     true,
			Connection: "keep-alive",
			Query:      generateSignature(byteToString(data)),
		},
	)
	return err
}

func (i *Instagram) Login() error {
	err := i.readMSISDNHeader()
	if err != nil {
		return err
	}

	err = i.syncFeatures()
	if err != nil {
		return err
	}

	err = i.zrToken()
	if err != nil {
		return err
	}

	err = i.sendAdID()
	if err != nil {
		return err
	}

	err = i.contactPrefill()
	if err != nil {
		return err
	}

	result, err := json.Marshal(
		map[string]interface{}{
			"guid":                i.uuid,
			"login_attempt_count": 0,
			"_csrftoken":          i.token,
			"device_id":           i.deviceID,
			"adid":                i.adID,
			"phone_id":            i.phoneID,
			"username":            i.username,
			"password":            i.password,
			"google_tokens":       "[]",
		},
	)
	if err != nil {
		return err
	}

	body, err := i.sendRequest(
		&reqOptions{
			Endpoint: igAPIURLLogin,
			Query:    generateSignature(byteToString(result)),
			IsPost:   true,
		},
	)
	if err != nil {
		return err
	}
	i.password = ""

	res := accountResp{}
	err = json.Unmarshal(body, &res)
	if err != nil {
		return err
	}

	i.Account = &res.Account
	i.Account.client = i
	i.rankToken = strconv.FormatInt(i.Account.ID, 10) + "_" + i.uuid
	err = i.zrToken()

	return err
}

func (i *Instagram) Logout() (*LogoutResponse, error) {
	result, err := json.Marshal(
		map[string]interface{}{
			"guid":       i.uuid,
			"_csrftoken": i.token,
			"device_id":  i.deviceID,
			"phone_id":   i.phoneID,
		},
	)
	if err != nil {
		return nil, err
	}

	body, err := i.sendRequest(
		&reqOptions{
			Endpoint: igAPIURLLogout,
			Query:    generateSignature(byteToString(result)),
			IsPost:   true,
		},
	)
	if err != nil {
		return nil, err
	}

	i.httpClient.Jar = nil
	i.httpClient = nil

	res := &LogoutResponse{}
	err = json.Unmarshal(body, res)
	if err != nil {
		return nil, err
	}

	return res, nil
}
