package hs

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
)

type HelpScout struct {
	AppID       string
	AppSecret   string
	AccessToken string
}

// NewHelpScout returns a new Help Scout instance
func NewHelpScout(appID string, appSecret string) (h *HelpScout, err error) {
	h = &HelpScout{
		AppID:       appID,
		AppSecret:   appSecret,
		AccessToken: "",
	}

	_, err = h.Exec("tags", nil, nil)
	if err != nil {
		return nil, err
	}

	return
}

type RToken struct {
	TokenType   string `json:"token_type"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

func (h *HelpScout) GetAccessToken() (err error) {
	r, err, _ := h.RawExec("oauth2/token", url.Values{
		"client_id":     {h.AppID},
		"client_secret": {h.AppSecret},
		"grant_type":    {"client_credentials"},
	}, &RToken{})
	if err != nil {
		return err
	}

	resp := r.(*RToken)
	h.AccessToken = resp.AccessToken

	return
}

func (h *HelpScout) RawExec(u string, v url.Values, dest interface{}) (r interface{}, err error, statusCode int) {
	u = "https://api.helpscout.net/v2/" + u
	log.Println("->", u, "?", v)

	client := &http.Client{}
	var m string
	if len(v) != 0 {
		m = "POST"
	} else {
		m = "GET"
	}
	req, err := http.NewRequest(m, u, strings.NewReader(v.Encode()))
	if err != nil {
		return nil, err, 0
	}
	if len(h.AccessToken) != 0 {
		req.Header.Add("Authorization", "Bearer "+h.AccessToken)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err, 0
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err, 0
	}

	log.Println("<-", string(body))

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("received status code %d", resp.StatusCode), 0
	}

	if dest != nil {
		err = json.Unmarshal(body, dest)
		if err != nil {
			return nil, err, 0
		}
	}

	return dest, nil, resp.StatusCode
}

func (h *HelpScout) Exec(u string, v url.Values, dest interface{}) (r interface{}, err error) {
	if len(h.AccessToken) == 0 {
		err = h.GetAccessToken()
		if err != nil {
			return nil, err
		}
	}

	r, err, statusCode := h.RawExec(u, v, dest)
	if err != nil {
		if statusCode == 401 {
			err = h.GetAccessToken()
			if err != nil {
				return nil, err
			}
			r, err, _ = h.RawExec(u, v, dest)
		}
	}

	return
}
