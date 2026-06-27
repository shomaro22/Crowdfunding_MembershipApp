package controller

type sendMessageRequestJSON struct {
	ProjectID  string `json:"project_id"`
	ReceiverID string `json:"receiver_id"`
	Content    string `json:"content"`
}
