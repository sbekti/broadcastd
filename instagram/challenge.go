package instagram

import (
	"encoding/json"
	"strings"
)

type ChallengeStepData struct {
	Choice           string      `json:"choice"`
	FbAccessToken    string      `json:"fb_access_token"`
	BigBlueToken     string      `json:"big_blue_token"`
	GoogleOauthToken string      `json:"google_oauth_token"`
	Email            string      `json:"email"`
	SecurityCode     string      `json:"security_code"`
	ResendDelay      interface{} `json:"resend_delay"`
	ContactPoint     string      `json:"contact_point"`
	FormType         string      `json:"form_type"`
}

type Challenge struct {
	client       *Instagram
	StepName     string            `json:"step_name"`
	StepData     ChallengeStepData `json:"step_data"`
	LoggedInUser *Account          `json:"logged_in_user,omitempty"`
	UserID       int64             `json:"user_id"`
	NonceCode    string            `json:"nonce_code"`
	Action       string            `json:"action"`
	Status       string            `json:"status"`
}

type challengeResp struct {
	*Challenge
}

type ChallengeProcessError struct {
	StepName string
}

func (e ChallengeProcessError) Error() string {
	return e.StepName
}

func newChallenge(i *Instagram) *Challenge {
	return &Challenge{
		client: i,
	}
}

func (challenge *Challenge) updateState() error {
	client := challenge.client

	data, err := client.prepareData(map[string]interface{}{
		"guid":      client.uuid,
		"device_id": client.deviceID,
	})
	if err != nil {
		return err
	}

	body, err := client.sendRequest(
		&reqOptions{
			Endpoint: challenge.client.challengeURL,
			Query:    generateSignature(data),
		},
	)
	if err == nil {
		resp := challengeResp{}
		err = json.Unmarshal(body, &resp)
		if err == nil {
			*challenge = *resp.Challenge
			challenge.client = client
		}
	}
	return err
}

func (challenge *Challenge) selectVerifyMethod(choice string, isReplay ...bool) error {
	client := challenge.client
	url := challenge.client.challengeURL

	if len(isReplay) > 0 && isReplay[0] {
		url = strings.Replace(url, "/challenge/", "/challenge/replay/", -1)
	}

	data, err := client.prepareData(map[string]interface{}{
		"choice":    choice,
		"guid":      client.uuid,
		"device_id": client.deviceID,
	})
	if err != nil {
		return err
	}

	body, err := client.sendRequest(
		&reqOptions{
			Endpoint: url,
			Query:    generateSignature(data),
			IsPost:   true,
		},
	)
	if err != nil {
		return err
	}

	resp := challengeResp{}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return err
	}

	*challenge = *resp.Challenge
	challenge.client = client

	return nil
}

func (challenge *Challenge) SendSecurityCode(code string) error {
	client := challenge.client
	url := challenge.client.challengeURL

	data, err := client.prepareData(map[string]interface{}{
		"security_code": code,
		"guid":          client.uuid,
		"device_id":     client.deviceID,
	})
	if err != nil {
		return err
	}

	body, err := client.sendRequest(
		&reqOptions{
			Endpoint: url,
			Query:    generateSignature(data),
			IsPost:   true,
		},
	)
	if err != nil {
		return err
	}

	resp := challengeResp{}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return err
	}

	*challenge = *resp.Challenge
	challenge.client = client

	return nil
}

func (challenge *Challenge) deltaLoginReview() error {
	// It was me = 0, It wasn't me = 1
	return challenge.selectVerifyMethod("0")
}

func (challenge *Challenge) Process(apiURL string) error {
	challenge.client.challengeURL = apiURL[1:]

	if err := challenge.updateState(); err != nil {
		return err
	}

	switch challenge.StepName {
	case "select_verify_method":
		return challenge.selectVerifyMethod(challenge.StepData.Choice)
	case "delta_login_review":
		return challenge.deltaLoginReview()
	}

	return ChallengeProcessError{StepName: challenge.StepName}
}
