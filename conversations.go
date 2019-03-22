package helpscout

import (
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

// Customer is a customer object, as defined by Help Scout
// The use of pointers for everything here is important
// so that we can omit some values instead of sending blank strings
type Customer struct {
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

type reqConversation struct {
	Subject   string      `json:"subject"`
	Customer  Customer    `json:"customer"`
	MailboxID int         `json:"mailboxId"`
	Type      string      `json:"type"`
	Status    string      `json:"status"`
	Created   time.Time   `json:"createdAt"`
	Threads   []NewThread `json:"threads"`
	Imported  bool        `json:"imported"`
	Tags      []string    `json:"tags"`
	Closed    time.Time   `json:"closedAt"`
}

// NewConversationWithMessage creates a new message thread from the
// given customer in the current mailbox
func (h *HelpScout) NewConversationWithMessage(subject string, customer Customer, created time.Time, tags []string, content string, searchForThreadID bool) (conversationID int, threadID int, err error) {
	return h.NewConversationWithThread("customer", subject, customer, created, tags, content, searchForThreadID)
}

// NewConversationWithReply creates a reply thread to the given customer
func (h *HelpScout) NewConversationWithReply(subject string, customer Customer, created time.Time, tags []string, content string, searchForThreadID bool) (conversationID int, threadID int, err error) {
	return h.NewConversationWithThread("reply", subject, customer, created, tags, content, searchForThreadID)
}

// NewConversationWithThread creates a conversation and a thread with the given customer information
func (h *HelpScout) NewConversationWithThread(threadType string, subject string, customer Customer, created time.Time, tags []string, content string, searchForThreadID bool) (conversationID int, threadID int, err error) {
	conversationID, err = h.NewConversation(subject, customer, created, tags, []NewThread{{
		Type:     threadType,
		Customer: customer,
		Content:  content,
		Imported: true,
		Created:  created,
	}})
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
func (h *HelpScout) NewConversation(subject string, customer Customer, created time.Time, tags []string, threads []NewThread) (conversationID int, err error) {
	if len(subject) == 0 {
		return 0, fmt.Errorf("subjects cannot be blank")
	}

	_, header, err := h.Exec("conversations", &reqConversation{
		Subject:   subject,
		Customer:  customer,
		MailboxID: h.MailboxID,
		Type:      "email",
		Status:    "closed",
		Created:   created,
		Threads:   threads,
		Imported:  true,
		Tags:      tags,
		Closed:    created,
	}, nil)
	if err != nil {
		return
	}

	conversationID, _ = strconv.Atoi(header.Get("Resource-ID"))
	return
}
