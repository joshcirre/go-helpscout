package helpscout

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	retry "github.com/StirlingMarketingGroup/go-retry"

	. "github.com/logrusorgru/aurora"
)

// RetryCount is the number of times retired functions get retried
var RetryCount = 10

func log(connNum int, msg interface{}) {
	fmt.Println(Sprintf("%s %s: %s", time.Now().Format("2006-01-02 15:04:05.000000"), Colorize(fmt.Sprintf("HelpScout%d", connNum), MagentaFg|BoldFm), msg))
}

// Verbose outputs every command and its response with the Help Scout API
var Verbose = false

// ShowPostData being set to false will hide the query in requests in verbose mode
var ShowPostData = true

// ShowResponse being set to false will hide any Help Scout responses in verbose mode
var ShowResponse = true

// RateLimitPercent is the percent (as a decimal) of how much of the available rate limit to use. E.g., rate limit is 400/minute; if .75 is given, then 300/minute will be this instance's effective rate limit
var RateLimitPercent float64 = 1

// CurrentRateMinute is the current count of API requests in the last minute
// var currentRateMinute = 0
var currentRateMinuteCh chan struct{} // = make(chan struct{}, RateLimitMinute)

// var currentRateMinuteMtx = sync.RWMutex{}

// HelpScout is a Help Scout connection instance
type HelpScout struct {
	AppID           string
	AppSecret       string
	AccessToken     string
	MailboxID       int
	MailboxSelected bool
	ConnNum         int
	accessTokenMtx  sync.RWMutex
	reqMtx          sync.Mutex
}

// ReadAccessToken safely returns the access token in a async-safe way
func (h *HelpScout) ReadAccessToken() string {
	h.accessTokenMtx.RLock()
	accessToken := h.AccessToken
	h.accessTokenMtx.RUnlock()
	return accessToken
}

var nextConnNum = 0
var nextConnNumMutex = sync.RWMutex{}

// New returns a new Help Scout instance
func New(appID string, appSecret string) (h *HelpScout, err error) {
	nextConnNumMutex.RLock()
	connNum := nextConnNum
	nextConnNumMutex.RUnlock()

	nextConnNumMutex.Lock()
	nextConnNum++
	nextConnNumMutex.Unlock()

	h = &HelpScout{
		AppID:       appID,
		AppSecret:   appSecret,
		AccessToken: "",
		ConnNum:     connNum,
	}

	err = h.GetNewAccessToken()
	if err != nil {
		return nil, err
	}

	return
}

type respToken struct {
	TokenType   string `json:"token_type"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

// GetNewAccessToken gets an access token for the Help Scout API
func (h *HelpScout) GetNewAccessToken() (err error) {
	accessToken := h.ReadAccessToken()

	h.accessTokenMtx.Lock()
	if accessToken == h.AccessToken {
		r, _, _, _, err := h.RawExec("oauth2/token", url.Values{
			"client_id":     {h.AppID},
			"client_secret": {h.AppSecret},
			"grant_type":    {"client_credentials"},
		}, &respToken{}, "POST", false, true)
		if err != nil {
			return err
		}

		resp := r.(*respToken)
		h.AccessToken = resp.AccessToken
	}
	h.accessTokenMtx.Unlock()

	return
}

// func getCurrentRateMinute() int {
// 	currentRateMinuteMtx.RLock()
// 	i := currentRateMinute
// 	currentRateMinuteMtx.RUnlock()
// 	return i
// }

// RawExec sends a request to the given URL with the given params to the
// Help Scout API and returns its response
func (h *HelpScout) RawExec(u string, v interface{}, dest interface{}, method string, rateLimited bool, mutexLocked bool) (r interface{}, statusCode int, header http.Header, resp []byte, err error) {
	u = "https://api.helpscout.net/v2/" + u
	client := &http.Client{
		Timeout: time.Minute,
	}

	var body []byte
	var _req *http.Request
	var _params string
	var _resp *http.Response
	err = retry.Retry(func() (err error) {
		var req *http.Request
		_req = req
		var params string
		if v == nil {
			if len(method) == 0 {
				method = "GET"
			}
			req, err = http.NewRequest(method, u, nil)
		} else {
			if len(method) == 0 {
				method = "POST"
			}
			switch v.(type) {
			case url.Values:
				if Verbose {
					params = v.(url.Values).Encode()
				}
				req, err = http.NewRequest(method, u, strings.NewReader(v.(url.Values).Encode()))
				req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
			default:
				var j []byte
				j, err = json.Marshal(v)
				if err != nil {
					return
				}
				if Verbose {
					params = string(j)
				}
				req, err = http.NewRequest(method, u, bytes.NewBuffer(j))
				req.Header.Add("Content-Type", "application/json")
			}
		}
		if err != nil {
			return
		}

		var accessToken string
		if mutexLocked {
			accessToken = h.AccessToken
		} else {
			accessToken = h.ReadAccessToken()
		}
		if len(accessToken) != 0 {
			req.Header.Add("Authorization", "Bearer "+accessToken)
		}

		if Verbose {
			var q string
			_params = params
			if ShowPostData {
				q = params
			}
			log(h.ConnNum, strings.Replace(fmt.Sprintf("%s %03d/%03d %s %s %s", Bold("->"), len(currentRateMinuteCh), cap(currentRateMinuteCh), req.Method, u, q), fmt.Sprintf(`"%s"`, h.AppSecret), `"****"`, -1))
		}

		if rateLimited && currentRateMinuteCh != nil {
			payloadRequests := 1
			switch strings.ToLower(req.Method) {
			case "post", "put", "delete", "patch":
				payloadRequests = 2
			}

			for i := 0; i < payloadRequests; i++ {
				currentRateMinuteCh <- struct{}{}
				go func() {
					time.Sleep(time.Minute)
					<-currentRateMinuteCh
				}()
			}
		}

		resp, err := client.Do(req)
		_resp = resp
		if err != nil {
			return fmt.Errorf("helpscout rawexec: %s", err)
		}
		// defer resp.Body.Close()
		if currentRateMinuteCh == nil {
			if rate, ok := resp.Header["X-Ratelimit-Limit-Minute"]; ok {
				n, _ := strconv.Atoi(rate[0])
				currentRateMinuteCh = make(chan struct{}, int(float64(n)*RateLimitPercent))

				if Verbose {
					log(h.ConnNum, fmt.Sprintf("Current rate limit is %d", n))
				}

				if cur, ok := resp.Header["X-Ratelimit-Remaining-Minute"]; ok {
					r, _ := strconv.Atoi(cur[0])
					used := n - r
					for i := 0; i < used; i++ {
						currentRateMinuteCh <- struct{}{}
						go func() {
							time.Sleep(time.Minute)
							<-currentRateMinuteCh
						}()
					}
				}
			}
		}

		body, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return
		}

		if Verbose && ShowResponse {
			log(h.ConnNum, fmt.Sprintf("<- %s", string(body)))
		}
		statusCode = resp.StatusCode

		if statusCode == 401 {
			err = h.GetNewAccessToken()
			if err != nil {
				return
			}
			err = &retry.NoFail{Err: fmt.Errorf("received new access token")}
			return
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("received status code %d", resp.StatusCode)
		}

		header = resp.Header

		return
	}, RetryCount, func(err error) error {
		if Verbose {
			log(h.ConnNum, Red(err))
		}
		return nil
	}, nil)
	if err != nil {
		err = fmt.Errorf("helpscout exec: %s", err)
		if Verbose {
			fmt.Println("==================\nREQUEST:")
			if _req != nil {
				fmt.Println(_req.Header)
			}
			fmt.Println(_params)
			fmt.Println("\nRESPONSE:")
			if _req != nil {
				fmt.Println(_resp.Header)
			}
			fmt.Printf("%s\n", body)
			fmt.Println("==================")
		}
		return
	}

	if dest != nil {
		err = json.Unmarshal(body, dest)
		if err != nil {
			return
		}
	}

	resp, err = ioutil.ReadAll(_resp.Body)
	if err != nil {
		return
	}
	_resp.Body.Close()
	return dest, statusCode, header, resp, nil
}

// Exec wraps the RaWExec function for common requests
func (h *HelpScout) Exec(u string, v interface{}, dest interface{}, method string) (r interface{}, header http.Header, resp []byte, err error) {
	if len(h.ReadAccessToken()) == 0 {
		err = h.GetNewAccessToken()
		if err != nil {
			return
		}
	}

	r, _, header, resp, err = h.RawExec(u, v, dest, method, true, false)
	if err != nil {
		return nil, nil, resp, err
	}
	return
}

// EmailAddresses is a map of email addresses in the form of
// EmailAddress => Name
type EmailAddresses map[string]string
