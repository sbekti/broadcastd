package instagram

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	neturl "net/url"
	"time"
)

const (
	httpTimeout = 30 * time.Second
)

type config struct {
	ID        int64          `json:"id"`
	Username  string         `json:"username"`
	DeviceID  string         `json:"device_id"`
	UUID      string         `json:"uuid"`
	RankToken string         `json:"rank_token"`
	Token     string         `json:"token"`
	PhoneID   string         `json:"phone_id"`
	Cookies   []*http.Cookie `json:"cookies"`
}

func exportConfig(client *Instagram, writer io.Writer) error {
	url, err := neturl.Parse(igAPIBaseURL)
	if err != nil {
		return err
	}

	config := config{
		ID:        client.Account.ID,
		Username:  client.username,
		DeviceID:  client.deviceID,
		UUID:      client.uuid,
		RankToken: client.rankToken,
		Token:     client.token,
		PhoneID:   client.phoneID,
		Cookies:   client.httpClient.Jar.Cookies(url),
	}

	jsonBytes, err := json.Marshal(config)
	if err != nil {
		return err
	}

	_, err = writer.Write(jsonBytes)
	return err
}

func ExportToString(client *Instagram) (string, error) {
	buffer := &bytes.Buffer{}
	err := exportConfig(client, buffer)
	if err != nil {
		return "", err
	}

	encoded := base64.StdEncoding.EncodeToString(buffer.Bytes())
	return encoded, nil
}

func importConfig(config config) (*Instagram, error) {
	url, err := neturl.Parse(igAPIBaseURL)
	if err != nil {
		return nil, err
	}

	client := &Instagram{
		username:  config.Username,
		deviceID:  config.DeviceID,
		uuid:      config.UUID,
		rankToken: config.RankToken,
		token:     config.Token,
		phoneID:   config.PhoneID,
		httpClient: &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
			},
			Timeout: httpTimeout,
		},
	}

	client.httpClient.Jar, err = cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	client.httpClient.Jar.SetCookies(url, config.Cookies)
	client.init()

	client.Account = &Account{client: client, ID: config.ID}
	err = client.Account.Sync()
	if err != nil {
		return nil, err
	}

	return client, nil
}

func ImportFromString(base64String string) (*Instagram, error) {
	decoded, err := base64.StdEncoding.DecodeString(base64String)
	if err != nil {
		return nil, err
	}

	allBytes, err := ioutil.ReadAll(bytes.NewReader(decoded))
	if err != nil {
		return nil, err
	}

	config := config{}
	err = json.Unmarshal(allBytes, &config)
	if err != nil {
		return nil, err
	}

	return importConfig(config)
}
