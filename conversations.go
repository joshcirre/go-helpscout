package helpscout

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

const (
	reqPhotoUnknown       = "unknown"
	reqPhotoGravatar      = "gravatar"
	reqPhotoTwitter       = "twitter"
	reqPhotoFacebook      = "facebook"
	reqPhotoGoogleProfile = "googleprofile"
	reqPhotoGooglePlus    = "googleplus"
	reqPhotoLinkedIn      = "linkedin"
)

const (
	reqGenderMale    = "male"
	reqGenderFemale  = "female"
	reqGenderUnknown = "unknown"
)

// Time is the same as time.Time, but marshals with time.RFC3339
type Time time.Time

// MarshalJSON marshalls Time with time.RFC3339
func (t Time) MarshalJSON() ([]byte, error) {
	if y := time.Time(t).Year(); y < 0 || y >= 10000 {
		// RFC 3339 is clear that years are 4 digits exactly.
		// See golang.org/issue/4556#c15 for more discussion.
		return nil, errors.New("Time.MarshalJSON: year outside of range [0,9999]")
	}

	formatISO8601 := "2006-01-02T15:04:05Z"
	b := make([]byte, 0, len(formatISO8601)+2)
	b = append(b, '"')
	b = time.Time(t).AppendFormat(b, formatISO8601)
	b = append(b, '"')
	return b, nil
}

// Customer is a customer object, as defined by Help Scout
// The use of pointers for everything here is important
// so that we can omit some values instead of sending blank strings
type Customer struct {
	ID        int    `json:"id,omitempty"`
	Email     string `json:"email,omitempty"`
	FirstName string `json:"firstName,omitempty"`
	LastName  string `json:"lastName,omitempty"`
	PhotoURL  string `json:"photoUrl,omitempty"`
	JobTitle  string `json:"jobTitle,omitempty"`
	PhotoType string `json:"photoType,omitempty"`
	Notes     string `json:"background,omitempty"`
	Location  string `json:"location,omitempty"`
	Created   *Time  `json:"createdAt,omitempty"`
	Company   string `json:"organization,omitempty"`
	Gender    string `json:"gender,omitempty"`
	Age       string `json:"age,omitempty"`
}

type reqConversation struct {
	Subject   string      `json:"subject"`
	Customer  Customer    `json:"customer"`
	MailboxID int         `json:"mailboxId"`
	Type      string      `json:"type"`
	Status    string      `json:"status"`
	Created   *Time       `json:"createdAt"`
	Threads   []NewThread `json:"threads"`
	Imported  bool        `json:"imported"`
	Tags      []string    `json:"tags"`
	Closed    *Time       `json:"closedAt"`
	User      int         `json:"user,omitempty"`
}

// NewConversationWithMessage creates a new message thread from the
// given customer in the current mailbox
func (h *HelpScout) NewConversationWithMessage(subject string, customer Customer, created time.Time, tags []string, content string, searchForThreadID bool, closed bool, user int) (conversationID int, threadID int, resp []byte, err error) {
	return h.NewConversationWithThread("customer", subject, customer, created, tags, content, searchForThreadID, closed, user)
}

// NewConversationWithReply creates a reply thread to the given customer
func (h *HelpScout) NewConversationWithReply(subject string, customer Customer, created time.Time, tags []string, content string, searchForThreadID bool, closed bool, user int) (conversationID int, threadID int, resp []byte, err error) {
	return h.NewConversationWithThread("reply", subject, customer, created, tags, content, searchForThreadID, closed, user)
}

// NewConversationWithThread creates a conversation and a thread with the given customer information
func (h *HelpScout) NewConversationWithThread(threadType string, subject string, customer Customer, created time.Time, tags []string, content string, searchForThreadID bool, closed bool, user int) (conversationID int, threadID int, resp []byte, err error) {
	conversationID, resp, err = h.NewConversation(subject, customer, created, tags, []NewThread{{
		Type:     threadType,
		Customer: customer,
		Content:  content,
		Imported: true,
		Created:  Time(created.UTC()),
	}}, closed, user)
	if err != nil {
		return
	}

	// This can be disabled since getting the thread ID takes extra API requests,
	// unlike getting the conversation ID, which is returned with the conversation
	// creation request
	if searchForThreadID {
		threadID, err = h.GetEarliestThreadID(conversationID)
	}

	return
}

// NewConversation creates a new conversation with the given customer and returns the new Conversation ID
func (h *HelpScout) NewConversation(subject string, customer Customer, created time.Time, tags []string, threads []NewThread, closed bool, user int) (conversationID int, resp []byte, err error) {
	if len(subject) == 0 {
		return 0, nil, fmt.Errorf("subjects cannot be blank")
	}

	var status string
	if closed {
		status = "closed"
	} else {
		status = "active"
	}

	closedTime := new(Time)
	if closed {
		*closedTime = Time(created.UTC())
	} else {
		closedTime = nil
	}

	createdTime := new(Time)
	*createdTime = Time(created.UTC())

	customer.Created = createdTime
	_, header, resp, err := h.Exec("conversations", &reqConversation{
		Subject:   subject,
		Customer:  customer,
		MailboxID: h.MailboxID,
		Type:      "email",
		Status:    status,
		Created:   createdTime,
		Threads:   threads,
		Imported:  true,
		Tags:      tags,
		Closed:    closedTime,
		User:      user,
	}, nil, "")
	if err != nil {
		return
	}

	conversationID, _ = strconv.Atoi(header.Get("Resource-ID"))
	return
}

// RsListConversations is a list conversations response
type RsListConversations struct {
	Embedded struct {
		Conversations []Conversation `json:"conversations"`
	} `json:"_embedded"`
	Links struct {
		First struct {
			Href string `json:"href"`
		} `json:"first"`
		Last struct {
			Href string `json:"href"`
		} `json:"last"`
		Next struct {
			Href string `json:"href"`
		} `json:"next"`
		Page struct {
			Href string `json:"href"`
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

// Conversation is a Help Scout conversation
type Conversation struct {
	ID        int    `json:"id"`
	Number    int    `json:"number"`
	Threads   int    `json:"threads"`
	Type      string `json:"type"`
	FolderID  int    `json:"folderId"`
	Status    string `json:"status"`
	State     string `json:"state"`
	Subject   string `json:"subject"`
	Preview   string `json:"preview"`
	MailboxID int    `json:"mailboxId"`
	CreatedBy struct {
		ID       int    `json:"id"`
		Type     string `json:"type"`
		First    string `json:"first"`
		Last     string `json:"last"`
		PhotoURL string `json:"photoUrl"`
		Email    string `json:"email"`
	} `json:"createdBy,omitempty"`
	CreatedAt            time.Time `json:"createdAt"`
	ClosedBy             int       `json:"closedBy"`
	UserUpdatedAt        time.Time `json:"userUpdatedAt"`
	CustomerWaitingSince struct {
		Time     time.Time `json:"time"`
		Friendly string    `json:"friendly"`
	} `json:"customerWaitingSince"`
	Source struct {
		Type string `json:"type"`
		Via  string `json:"via"`
	} `json:"source"`
	Tags            []interface{} `json:"tags"`
	Cc              []interface{} `json:"cc"`
	Bcc             []interface{} `json:"bcc"`
	PrimaryCustomer struct {
		ID       int    `json:"id"`
		Type     string `json:"type"`
		First    string `json:"first"`
		Last     string `json:"last"`
		PhotoURL string `json:"photoUrl"`
		Email    string `json:"email"`
	} `json:"primaryCustomer"`
	CustomFields []interface{} `json:"customFields"`
	Links        struct {
		ClosedBy struct {
			Href string `json:"href"`
		} `json:"closedBy"`
		CreatedByCustomer struct {
			Href string `json:"href"`
		} `json:"createdByCustomer"`
		Mailbox struct {
			Href string `json:"href"`
		} `json:"mailbox"`
		PrimaryCustomer struct {
			Href string `json:"href"`
		} `json:"primaryCustomer"`
		Self struct {
			Href string `json:"href"`
		} `json:"self"`
		Threads struct {
			Href string `json:"href"`
		} `json:"threads"`
		Web struct {
			Href string `json:"href"`
		} `json:"web"`
	} `json:"_links,omitempty"`
	ClosedAt time.Time `json:"closedAt,omitempty"`
}

// ListConversations takes a Help Scout query and returns all
// conversations on every page for that search
// https://developer.helpscout.com/mailbox-api/endpoints/conversations/list/
func (h *HelpScout) ListConversations(query string) (conversations []Conversation, err error) {
	var rs RsListConversations
	page := 1
	if len(query) != 0 {
		query = "&query=" + url.QueryEscape(query)
	}

	for {
		_, _, _, err = h.Exec("conversations?status=all&mailbox="+strconv.Itoa(h.MailboxID)+"&page="+strconv.Itoa(page)+query, nil, &rs, "")
		if err != nil {
			return nil, err
		}
		if page == 1 {
			conversations = make([]Conversation, 0, rs.Page.TotalElements)
		}
		conversations = append(conversations, rs.Embedded.Conversations...)

		if page == rs.Page.TotalPages {
			break
		}
		page++
	}

	return
}

// ListConversationsByEmail returns all conversations for the given email
func (h *HelpScout) ListConversationsByEmail(email string) (conversations []Conversation, err error) {
	return h.ListConversations(`(email:"` + email + `")`)
}
