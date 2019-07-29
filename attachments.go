package helpscout

import "strconv"

type reqUploadAttachment struct {
	Name     string `json:"fileName"`
	MimeType string `json:"mimeType"`
	Data     []byte `json:"data"`
}

// UploadAttachment uploads an attachment to the given conversation > thread
func (h *HelpScout) UploadAttachment(conversationID int, threadID int, name string, mimeType string, data []byte) (resp []byte, err error) {
	_, _, resp, err = h.Exec(
		"conversations/"+strconv.Itoa(conversationID)+"/threads/"+strconv.Itoa(threadID)+"/attachments",
		reqUploadAttachment{
			Name:     name,
			MimeType: mimeType,
			Data:     data,
		},
		nil,
	)
	return
}
