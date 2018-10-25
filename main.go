package helpscout

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type HelpScout struct {
	AppID           string
	AppSecret       string
	AccessToken     string
	MailboxID       int
	MailboxSelected bool
}

// New returns a new Help Scout instance
func New(appID string, appSecret string) (h *HelpScout, err error) {
	h = &HelpScout{
		AppID:       appID,
		AppSecret:   appSecret,
		AccessToken: "",
	}

	err = h.GetAccessToken()
	if err != nil {
		return nil, err
	}

	return
}

type RespToken struct {
	TokenType   string `json:"token_type"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

func (h *HelpScout) GetAccessToken() (err error) {
	r, err, _ := h.RawExec("oauth2/token", url.Values{
		"client_id":     {h.AppID},
		"client_secret": {h.AppSecret},
		"grant_type":    {"client_credentials"},
	}, &RespToken{})
	if err != nil {
		return err
	}

	resp := r.(*RespToken)
	h.AccessToken = resp.AccessToken

	return
}

func (h *HelpScout) RawExec(u string, v interface{}, dest interface{}) (r interface{}, err error, statusCode int) {
	u = "https://api.helpscout.net/v2/" + u
	log.Println("->", u, "?", v)

	client := &http.Client{}
	var req *http.Request
	if v == nil {
		req, err = http.NewRequest("GET", u, nil)
	} else {
		switch v.(type) {
		case url.Values:
			req, err = http.NewRequest("POST", u, strings.NewReader(v.(url.Values).Encode()))
			req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		default:
			var j []byte
			j, err = json.Marshal(v)
			if err != nil {
				return nil, err, 0
			}
			req, err = http.NewRequest("POST", u, bytes.NewBuffer(j))
			req.Header.Add("Content-Type", "application/json")
		}
	}
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

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
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

func (h *HelpScout) Exec(u string, v interface{}, dest interface{}) (r interface{}, err error) {
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

type EmailAddresses map[string]string

const (
	ReqPhotoUnknown       = "unknown"
	ReqPhotoGravatar      = "gravatar"
	ReqPhotoTwitter       = "twitter"
	ReqPhotoFacebook      = "facebook"
	ReqPhotoGoogleProfile = "googleprofile"
	ReqPhotoGooglePlus    = "googleplus"
	ReqPhotoLinkedIn      = "linkedin"
)

const (
	ReqGenderMale    = "male"
	ReqGenderFemale  = "female"
	ReqGenderUnknown = "unknown"
)

type ReqConversationCustomer struct {
	ID        *int       `json:"id"`
	Email     *string    `json:"email"`
	FirstName *string    `json:"firstName"`
	LastName  *string    `json:"lastName"`
	PhotoURL  *string    `json:"photoUrl"`
	JobTitle  *string    `json:"jobTitle"`
	PhotoType *string    `json:"photoType"`
	Notes     *string    `json:"background"`
	Location  *string    `json:"location"`
	Created   *time.Time `json:"createdAt"`
	Company   *string    `json:"organization"`
	Gender    *string    `json:"gender"`
	Age       *string    `json:"age"`
}

type ReqConversationThread struct {
	Type     string                  `json:"type"`
	Customer ReqConversationCustomer `json:"customer"`
	Content  string                  `json:"text"`
	Imported bool                    `json:"imported"`
}

type ReqConversation struct {
	Subject   string                  `json:"subject"`
	Customer  ReqConversationCustomer `json:"customer"`
	MailboxID int                     `json:"mailboxId"`
	Type      string                  `json:"type"`
	Status    string                  `json:"status"`
	Created   time.Time               `json:"createdAt"`
	Threads   []ReqConversationThread `json:"threads"`
	Imported  bool                    `json:"imported"`
	Tags      []string                `json:"tags"`
}

// CreateMessage creates a message (conversation?) in the currently selected mailbox
func (h *HelpScout) CreateMessage(subject string, customer ReqConversationCustomer, content string, created time.Time, tags []string) error {
	return h.createMessage("customer", subject, customer, content, created, tags)
}

// CreateReply creates a reply to the given customer
func (h *HelpScout) CreateReply(subject string, customer ReqConversationCustomer, content string, created time.Time, tags []string) error {
	return h.createMessage("reply", subject, customer, content, created, tags)
}

func (h *HelpScout) createMessage(threadType string, subject string, customer ReqConversationCustomer, content string, created time.Time, tags []string) error {
	_, err := h.Exec("conversations", &ReqConversation{
		Subject:   subject,
		Customer:  customer,
		MailboxID: h.MailboxID,
		Type:      "email",
		Status:    "closed",
		Created:   created,
		Threads: []ReqConversationThread{{
			Type:     threadType,
			Customer: customer,
			Content:  content,
			Imported: true,
		}},
		Imported: true,
		Tags:     tags,
	}, nil)

	return err
}

type RespMailboxes struct {
	Embedded struct {
		Mailboxes []struct {
			ID        int       `json:"id"`
			Name      string    `json:"name"`
			Slug      string    `json:"slug"`
			Email     string    `json:"email"`
			CreatedAt time.Time `json:"createdAt"`
			UpdatedAt time.Time `json:"updatedAt"`
			Links     struct {
				Fields struct {
					Href string `json:"href"`
				} `json:"fields"`
				Folders struct {
					Href string `json:"href"`
				} `json:"folders"`
				Self struct {
					Href string `json:"href"`
				} `json:"self"`
			} `json:"_links"`
		} `json:"mailboxes"`
	} `json:"_embedded"`
	Links struct {
		First struct {
			Href string `json:"href"`
		} `json:"first"`
		Last struct {
			Href string `json:"href"`
		} `json:"last"`
		Page struct {
			Href      string `json:"href"`
			Templated bool   `json:"templated"`
		} `json:"page"`
		Self struct {
			Href string `json:"href"`
		} `json:"self"`
	} `json:"_links"`
	Page struct {
		Size          int `json:"size"`
		TotalElements int `json:"totalElements"`
		TotalPages    int `json:"totalPages"`
		Number        int `json:"number"`
	} `json:"page"`
}

func (h *HelpScout) SetMailboxID(id int) {
	h.MailboxID = id
	h.MailboxSelected = true
}

func (h *HelpScout) DeselectMailbox() {
	h.MailboxID = 0
	h.MailboxSelected = false
}

func (h *HelpScout) SelectMailbox(mailbox interface{}) error {
	r, err := h.Exec("mailboxes", nil, &RespMailboxes{})
	if err != nil {
		return err
	}
	resp := r.(*RespMailboxes)

	h.DeselectMailbox()
L:
	for _, m := range resp.Embedded.Mailboxes {
		switch mailbox.(type) {
		case string:
			if m.Email == mailbox || m.Name == mailbox {
				h.SetMailboxID(m.ID)
				break L
			}
		case int:
			if m.ID == mailbox {
				h.SetMailboxID(m.ID)
				break L
			}
		default:
			return fmt.Errorf("%Ts aren't supported for selecting a mailbox", mailbox)
		}
	}

	if !h.MailboxSelected {
		return fmt.Errorf("Couldn't find mailbox named/with id '%v'", mailbox)
	}

	return nil
}
