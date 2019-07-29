package helpscout

import (
	"fmt"
	"strconv"
	"time"
)

// NewThread can be though of as a message;
// Conversations are named literally, and conversations contain threads
type NewThread struct {
	Type     string   `json:"type"`
	Customer Customer `json:"customer"`
	Content  string   `json:"text"`
	Imported bool     `json:"imported"`
	Created  Time     `json:"createdAt"`
}

// Thread is an already existing thread
type Thread struct {
	ID     int    `json:"id"`
	Type   string `json:"type"`
	Status string `json:"status"`
	State  string `json:"state"`
	Action struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"action"`
	Body   string `json:"body"`
	Source struct {
		Type string `json:"type"`
		Via  string `json:"via"`
	} `json:"source"`
	Customer struct {
		ID       int    `json:"id"`
		First    string `json:"first"`
		Last     string `json:"last"`
		PhotoURL string `json:"photoUrl"`
		Email    string `json:"email"`
	} `json:"customer"`
	CreatedBy struct {
		ID       int    `json:"id"`
		Type     string `json:"type"`
		First    string `json:"first"`
		Last     string `json:"last"`
		PhotoURL string `json:"photoUrl"`
		Email    string `json:"email"`
	} `json:"createdBy"`
	AssignedTo struct {
		ID    int    `json:"id"`
		Type  string `json:"type"`
		First string `json:"first"`
		Last  string `json:"last"`
		Email string `json:"email"`
	} `json:"assignedTo"`
	SavedReplyID int       `json:"savedReplyId"`
	To           []string  `json:"to"`
	Cc           []string  `json:"cc"`
	Bcc          []string  `json:"bcc"`
	CreatedAt    time.Time `json:"createdAt"`
	OpenedAt     time.Time `json:"openedAt"`
	Embedded     struct {
		Attachments []struct {
			ID       int    `json:"id"`
			Filename string `json:"filename"`
			MimeType string `json:"mimeType"`
			Width    int    `json:"width"`
			Height   int    `json:"height"`
			Size     int    `json:"size"`
			Links    struct {
				Data struct {
					Href string `json:"href"`
				} `json:"data"`
				Self struct {
					Href string `json:"href"`
				} `json:"self"`
			} `json:"_links"`
		} `json:"attachments"`
	} `json:"_embedded"`
	Links struct {
		AssignedTo struct {
			Href string `json:"href"`
		} `json:"assignedTo"`
		CreatedByCustomer struct {
			Href string `json:"href"`
		} `json:"createdByCustomer"`
		Customer struct {
			Href string `json:"href"`
		} `json:"customer"`
	} `json:"_links"`
}

type respThreads struct {
	Embedded struct {
		Threads []Thread `json:"threads"`
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

// GetThreads returns a slice of Threads
func (h *HelpScout) GetThreads(conversationID int) (threads []Thread, err error) {
	r, _, _, err := h.Exec(
		"conversations/"+strconv.Itoa(conversationID)+"/threads",
		nil,
		&respThreads{},
	)
	if err != nil {
		return
	}
	resp := r.(*respThreads)
	return resp.Embedded.Threads, nil
}

// GetLatestThreadIDFromThreads takes a Thread slice and returns the ID from the latest one
func (h *HelpScout) GetLatestThreadIDFromThreads(threads []Thread) (threadID int, err error) {
	if len(threads) == 0 {
		return 0, fmt.Errorf("no threads were given")
	}

	for _, t := range threads {
		if t.ID > threadID {
			threadID = t.ID
		}
	}
	return
}

// GetLatestThreadID takes a Conversation ID and returns the ID from the latest thread
func (h *HelpScout) GetLatestThreadID(conversationID int) (threadID int, err error) {
	threads, err := h.GetThreads(conversationID)
	if err != nil {
		return
	}

	return h.GetLatestThreadIDFromThreads(threads)
}

// GetEarliestThreadIDFromThreads takes a Thread slice and returns the ID from the earliest one
func (h *HelpScout) GetEarliestThreadIDFromThreads(threads []Thread) (threadID int, err error) {
	if len(threads) == 0 {
		return 0, fmt.Errorf("no threads were given")
	}

	for _, t := range threads {
		if t.ID < threadID || threadID == 0 {
			threadID = t.ID
		}
	}
	return
}

// GetEarliestThreadID takes a Conversation ID and returns the ID from the earliest thread
func (h *HelpScout) GetEarliestThreadID(conversationID int) (threadID int, err error) {
	threads, err := h.GetThreads(conversationID)
	if err != nil {
		return
	}

	return h.GetEarliestThreadIDFromThreads(threads)
}
