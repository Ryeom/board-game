package room

import "encoding/json"

type WSResponse struct {
	Status  string      `json:"status"`
	Data    any         `json:"data"`
	Message interface{} `json:"message"`
}

func SendWSJSON(att *Attender, response WSResponse) {
	if att.Connection != nil {
		if data, err := json.Marshal(response); err == nil {
			att.Connection.WriteMessage(1, data)
		}
	}
}

func SendWSError(att *Attender, message string) {
	SendWSJSON(att, WSResponse{
		Status:  "error",
		Data:    nil,
		Message: message,
	})
}
