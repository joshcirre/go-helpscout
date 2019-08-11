package helpscout

import (
	"fmt"
	"strconv"
	"time"

	cache "github.com/patrickmn/go-cache"
)

// RsListMailboxCustomFields is a mailbox custom fields response
// https://developer.helpscout.com/mailbox-api/endpoints/mailboxes/mailbox-fields/
type RsListMailboxCustomFields struct {
	Embedded struct {
		Fields []struct {
			ID       int    `json:"id"`
			Required bool   `json:"required"`
			Order    int    `json:"order"`
			Type     string `json:"type"`
			Name     string `json:"name"`
			Options  []struct {
				ID    int    `json:"id"`
				Order int    `json:"order"`
				Label string `json:"label"`
			} `json:"options"`
		} `json:"fields"`
	} `json:"_embedded"`
	Links struct {
		First struct {
			Href string `json:"href"`
		} `json:"first"`
		Last struct {
			Href string `json:"href"`
		} `json:"last"`
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

var getCustomFieldsCache = cache.New(10*time.Second, 20*time.Second)

// ListCustomFields returns all the current mailbox's custom fields
func (h *HelpScout) ListCustomFields() (fields RsListMailboxCustomFields, err error) {
	key := strconv.Itoa(h.MailboxID) + ":ListCustomFields"
	v, found := getCustomFieldsCache.Get(key)
	if found {
		return v.(RsListMailboxCustomFields), nil
	}

	_, _, _, err = h.Exec("mailboxes/"+strconv.Itoa(h.MailboxID)+"/fields", nil, &fields, "")
	getCustomFieldsCache.Set(key, fields, cache.DefaultExpiration)

	return
}

// GetCustomFieldIDByName gets a custom field ID by name in the current mailbox
func (h *HelpScout) GetCustomFieldIDByName(name string) (customerFieldID int, err error) {
	key := strconv.Itoa(h.MailboxID) + ":GetCustomFieldIDByName:" + name
	v, found := getCustomFieldsCache.Get(key)
	if found {
		return v.(int), nil
	}

	fields, err := h.ListCustomFields()
	if err != nil {
		return
	}

	for _, f := range fields.Embedded.Fields {
		if f.Name == name {
			getCustomFieldsCache.Set(key, f.ID, cache.NoExpiration)
			return f.ID, nil
		}
	}

	return 0, fmt.Errorf("couldn't find the custom field %q", name)
}

// RqUpdateCustomFields is a request for updating fields
// https://developer.helpscout.com/mailbox-api/endpoints/conversations/custom_fields/update/
type RqUpdateCustomFields struct {
	Fields []RqUpdateCustomField `json:"fields"`
}

// RqUpdateCustomField is a single field in the RqUpdateCustomFields request
type RqUpdateCustomField struct {
	ID    int         `json:"id"`
	Value interface{} `json:"value"`
}

// UpdateCustomFields updates all customer fields' values for the given conversation
func (h *HelpScout) UpdateCustomFields(conversationID int, fields map[string]interface{}) (err error) {
	f := make([]RqUpdateCustomField, len(fields))
	i := 0
	for k, v := range fields {
		customFieldID, err := h.GetCustomFieldIDByName(k)
		if err != nil {
			return err
		}

		f[i] = RqUpdateCustomField{
			ID:    customFieldID,
			Value: v,
		}

		i++
	}

	_, _, _, err = h.Exec("conversations/"+strconv.Itoa(conversationID)+"/fields", RqUpdateCustomFields{
		Fields: f,
	}, nil, "PUT")
	return
}
