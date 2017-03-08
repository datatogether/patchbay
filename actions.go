package main

import (
	"encoding/json"
	"fmt"
)

// Action is a collection of typed events for exchange between client & server
type Action interface {
	Type() string
}

type ClientAction interface {
	Action
	Parse(json.RawMessage) ClientRequestAction
}

type ClientRequestAction interface {
	Action
	SuccessType() string
	FailureType() string
	Exec() *ClientResponse
}

// ServerRequestAction is an action from the server to send to the client
type ServerRequestAction interface {
	Action
	Send()
}

// type MessageAction struct {
// 	actionType `json:"type"`
// 	Message    string `json:"message"`
// }

// func (m MessageAction) Type() string {
// }

type ClientResponse struct {
	Type    string      `json:"type"`
	Error   error       `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// ClientReqActions is a list of all actions a client may request
var ClientReqActions = []ClientAction{
	MsgReqAct{},
}

type MsgReqAct struct {
	err     error
	Message string
}

func (m MsgReqAct) Type() string        { return "MESSAGE_REQUEST" }
func (m MsgReqAct) SuccessType() string { return "MESSAGE_SUCCESS" }
func (m MsgReqAct) FailureType() string { return "MESSAGE_FAILURE" }

func (MsgReqAct) Parse(data json.RawMessage) ClientRequestAction {
	m := &MsgReqAct{}
	m.err = json.Unmarshal(data, m)
	return m
}
func (m *MsgReqAct) Exec() (res *ClientResponse) {
	return &ClientResponse{
		Type:    m.SuccessType(),
		Message: fmt.Sprintf("oh really? %s", m.Message),
		Data: map[string]string{
			"message": fmt.Sprintf("oh really? %s", m.Message),
		},
	}
}

// func (m *MessageAction) Bytes() ([]byte, error) {
// 	return json.Marshal(m)
// }

// func NewMessageAction(message string) Action {
// 	return *MessageAction{
// 		actionType: actionType("message"),
// 		Message:    message,
// 	}
// }
