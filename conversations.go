package helpscout

import (
	"errors"
	"fmt"
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
	}, nil)
	if err != nil {
		return
	}

	conversationID, _ = strconv.Atoi(header.Get("Resource-ID"))
	return
}
