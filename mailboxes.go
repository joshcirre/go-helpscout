package helpscout

import (
	"fmt"
	"time"
)

type respMailboxes struct {
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

// SetMailboxID sets the current mailbox ID
func (h *HelpScout) SetMailboxID(id int) {
	h.MailboxID = id
	h.MailboxSelected = true
}

// DeselectMailbox set no currently selected mailbox
func (h *HelpScout) DeselectMailbox() {
	h.MailboxID = 0
	h.MailboxSelected = false
}

// SelectMailbox searches for a mailbox ID with the given ID,
// mailbox name, or email address and selects it
func (h *HelpScout) SelectMailbox(mailbox interface{}) error {
	r, _, _, err := h.Exec("mailboxes", nil, &respMailboxes{})
	if err != nil {
		return err
	}
	resp := r.(*respMailboxes)

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
