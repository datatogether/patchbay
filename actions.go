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

type ClientResponse struct {
	Type    string      `json:"type"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
	Schema  string      `json:"schema,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// ClientReqActions is a list of all actions a client may request
var ClientReqActions = []ClientAction{
	MsgReqAct{},
	SearchReqAct{},
	ArchiveUrlAct{},
	FetchUrlAct{},
	FetchOutboundLinksAct{},
	FetchContentUrlsAction{},
	FetchContentConsensusAction{},
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

type SearchReqAct struct {
	err      error
	Query    string
	Page     int
	PageSize int
}

func (SearchReqAct) Type() string        { return "SEARCH_REQUEST" }
func (SearchReqAct) SuccessType() string { return "SEARCH_SUCCESS" }
func (SearchReqAct) FailureType() string { return "SEARCH_FAILURE" }

func (SearchReqAct) Parse(data json.RawMessage) ClientRequestAction {
	s := &SearchReqAct{}
	s.err = json.Unmarshal(data, s)
	return s
}
func (s *SearchReqAct) Exec() (res *ClientResponse) {
	if s.Page > 0 {
		s.Page = s.Page - 1
	}
	results, err := Search(appDB, s.Query, s.PageSize, s.Page*s.PageSize)
	if err != nil {
		return &ClientResponse{
			Type:  s.FailureType(),
			Error: err.Error(),
		}
	}
	return &ClientResponse{
		Type:   s.SuccessType(),
		Schema: "SEARCH_RESULT_ARRAY",
		Data:   results,
	}
}

// FetchUrlAct fetches a url from the DB
type FetchUrlAct struct {
	err error
	Url string
}

func (FetchUrlAct) Type() string        { return "URL_FETCH_REQUEST" }
func (FetchUrlAct) SuccessType() string { return "URL_FETCH_SUCCESS" }
func (FetchUrlAct) FailureType() string { return "URL_FETCH_FAILURE" }

func (FetchUrlAct) Parse(data json.RawMessage) ClientRequestAction {
	a := &FetchUrlAct{}
	a.err = json.Unmarshal(data, a)
	return a
}

func (a *FetchUrlAct) Exec() (res *ClientResponse) {
	u := &Url{Url: a.Url}
	if err := u.Read(appDB); err != nil {
		return &ClientResponse{
			Type:  a.FailureType(),
			Error: err.Error(),
		}
	}

	return &ClientResponse{
		Type:   a.SuccessType(),
		Schema: "URL",
		Data:   u,
	}
}

// FetchOutboundLinksAct fetches a url's outbound links
type FetchOutboundLinksAct struct {
	err error
	Url string
}

func (FetchOutboundLinksAct) Type() string        { return "URL_FETCH_OUTBOUND_LINKS_REQUEST" }
func (FetchOutboundLinksAct) SuccessType() string { return "URL_FETCH_OUTBOUND_LINKS_SUCCESS" }
func (FetchOutboundLinksAct) FailureType() string { return "URL_FETCH_OUTBOUND_LINKS_FAILURE" }

func (FetchOutboundLinksAct) Parse(data json.RawMessage) ClientRequestAction {
	a := &FetchOutboundLinksAct{}
	a.err = json.Unmarshal(data, a)
	return a
}

func (a *FetchOutboundLinksAct) Exec() (res *ClientResponse) {
	links, err := ReadDstLinks(appDB, &Url{Url: a.Url})
	if err != nil {
		return &ClientResponse{
			Type:  a.FailureType(),
			Error: err.Error(),
		}
	}

	return &ClientResponse{
		Type:   a.SuccessType(),
		Schema: "LINK_ARRAY",
		Data:   links,
	}
}

// ArchiveUrlAct triggers archiving a url
type ArchiveUrlAct struct {
	err error
	Url string
}

func (ArchiveUrlAct) Type() string        { return "URL_ARCHIVE_REQUEST" }
func (ArchiveUrlAct) SuccessType() string { return "URL_ARCHIVE_SUCCESS" }
func (ArchiveUrlAct) FailureType() string { return "URL_ARCHIVE_FAILURE" }

func (ArchiveUrlAct) Parse(data json.RawMessage) ClientRequestAction {
	a := &ArchiveUrlAct{}
	a.err = json.Unmarshal(data, a)
	return a
}

func (a *ArchiveUrlAct) Exec() (res *ClientResponse) {
	url, err := ArchiveUrlSync(appDB, a.Url)
	if err != nil {
		return &ClientResponse{
			Type:  a.FailureType(),
			Error: err.Error(),
		}
	}
	return &ClientResponse{
		Type:   a.SuccessType(),
		Schema: "URL",
		Data:   url,
	}
}

// FetchContentUrlsAction triggers archiving a url
type FetchContentUrlsAction struct {
	err  error
	Hash string
}

func (FetchContentUrlsAction) Type() string        { return "CONTENT_URLS_REQUEST" }
func (FetchContentUrlsAction) SuccessType() string { return "CONTENT_URLS_SUCCESS" }
func (FetchContentUrlsAction) FailureType() string { return "CONTENT_URLS_FAILURE" }

func (FetchContentUrlsAction) Parse(data json.RawMessage) ClientRequestAction {
	a := &FetchContentUrlsAction{}
	a.err = json.Unmarshal(data, a)
	return a
}

func (a *FetchContentUrlsAction) Exec() (res *ClientResponse) {
	urls, err := UrlsForHash(appDB, a.Hash)
	if err != nil {
		return &ClientResponse{
			Type:  a.FailureType(),
			Error: err.Error(),
		}
	}
	return &ClientResponse{
		Type:   a.SuccessType(),
		Schema: "URL_ARRAY",
		Data:   urls,
	}
}

// FetchContentConsensusAction triggers archiving a url
type FetchContentConsensusAction struct {
	err  error
	Hash string
}

func (FetchContentConsensusAction) Type() string        { return "CONTENT_CONSENSUS_REQUEST" }
func (FetchContentConsensusAction) SuccessType() string { return "CONTENT_CONSENSUS_SUCCESS" }
func (FetchContentConsensusAction) FailureType() string { return "CONTENT_CONSENSUS_FAILURE" }

func (FetchContentConsensusAction) Parse(data json.RawMessage) ClientRequestAction {
	a := &FetchContentConsensusAction{}
	a.err = json.Unmarshal(data, a)
	return a
}

func (a *FetchContentConsensusAction) Exec() (res *ClientResponse) {
	// urls, err := UrlsForHash(appDB, a.Hash)
	// if err != nil {
	//   return &ClientResponse{
	//     Type:  a.FailureType(),
	//     Error: err.Error(),
	//   }
	// }
	return &ClientResponse{
		Type:   a.SuccessType(),
		Schema: "CONSENSUS",
		Data: map[string]interface{}{
			"subject": a.Hash,
			"title": map[string]interface{}{
				"this is just a test": 3,
			},
			"format": map[string]interface{}{
				"html":      2,
				"text/html": 1,
			},
		},
	}
}
