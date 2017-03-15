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
	Parse(string, json.RawMessage) ClientRequestAction
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
	Type      string      `json:"type"`
	RequestId string      `json:"requestId"`
	Error     string      `json:"error,omitempty"`
	Message   string      `json:"message,omitempty"`
	Schema    string      `json:"schema,omitempty"`
	Data      interface{} `json:"data,omitempty"`
}

// ClientReqActions is a list of all actions a client may request
var ClientReqActions = []ClientAction{
	MsgReqAct{},
	SearchReqAct{},
	ArchiveUrlAct{},
	FetchUrlAct{},
	FetchInboundLinksAct{},
	FetchOutboundLinksAct{},
	FetchContentUrlsAction{},
	FetchContentConsensusAction{},
	FetchMetadataAction{},
	SaveMetadataAction{},
	FetchPrimersAction{},
	FetchPrimerAction{},
}

type ReqAction struct {
	err       error
	RequestId string `json:"requestId"`
}

type MsgReqAct struct {
	ReqAction
	Message string
}

func (m MsgReqAct) Type() string        { return "MESSAGE_REQUEST" }
func (m MsgReqAct) SuccessType() string { return "MESSAGE_SUCCESS" }
func (m MsgReqAct) FailureType() string { return "MESSAGE_FAILURE" }

func (MsgReqAct) Parse(reqId string, data json.RawMessage) ClientRequestAction {
	a := &MsgReqAct{}
	a.RequestId = reqId
	a.err = json.Unmarshal(data, a)
	return a
}
func (a *MsgReqAct) Exec() (res *ClientResponse) {
	return &ClientResponse{
		Type:    a.SuccessType(),
		Message: fmt.Sprintf("oh really? %s", a.Message),
		Data: map[string]string{
			"message": fmt.Sprintf("oh really? %s", a.Message),
		},
	}
}

type SearchReqAct struct {
	ReqAction
	Query    string
	Page     int
	PageSize int
}

func (SearchReqAct) Type() string        { return "SEARCH_REQUEST" }
func (SearchReqAct) SuccessType() string { return "SEARCH_SUCCESS" }
func (SearchReqAct) FailureType() string { return "SEARCH_FAILURE" }

func (SearchReqAct) Parse(reqId string, data json.RawMessage) ClientRequestAction {
	a := &SearchReqAct{}
	a.RequestId = reqId
	a.err = json.Unmarshal(data, a)
	return a
}
func (s *SearchReqAct) Exec() (res *ClientResponse) {
	if s.Page > 0 {
		s.Page = s.Page - 1
	}
	results, err := Search(appDB, s.Query, s.PageSize, s.Page*s.PageSize)
	if err != nil {
		return &ClientResponse{
			Type:      s.FailureType(),
			Error:     err.Error(),
			RequestId: s.RequestId,
		}
	}
	return &ClientResponse{
		Type:      s.SuccessType(),
		RequestId: s.RequestId,
		Schema:    "SEARCH_RESULT_ARRAY",
		Data:      results,
	}
}

// FetchUrlAct fetches a url from the DB
type FetchUrlAct struct {
	ReqAction
	Url string
}

func (FetchUrlAct) Type() string        { return "URL_FETCH_REQUEST" }
func (FetchUrlAct) SuccessType() string { return "URL_FETCH_SUCCESS" }
func (FetchUrlAct) FailureType() string { return "URL_FETCH_FAILURE" }

func (FetchUrlAct) Parse(reqId string, data json.RawMessage) ClientRequestAction {
	a := &FetchUrlAct{}
	a.RequestId = reqId
	a.err = json.Unmarshal(data, a)
	return a
}

func (a *FetchUrlAct) Exec() (res *ClientResponse) {
	u := &Url{Url: a.Url}
	if err := u.Read(appDB); err != nil {
		return &ClientResponse{
			Type:      a.FailureType(),
			RequestId: a.RequestId,
			Error:     err.Error(),
		}
	}

	return &ClientResponse{
		Type:   a.SuccessType(),
		Schema: "URL",
		Data:   u,
	}
}

// FetchInboundLinksAct fetches a url's outbound links
type FetchInboundLinksAct struct {
	ReqAction
	Url string
}

func (FetchInboundLinksAct) Type() string        { return "URL_FETCH_INBOUND_LINKS_REQUEST" }
func (FetchInboundLinksAct) SuccessType() string { return "URL_FETCH_INBOUND_LINKS_SUCCESS" }
func (FetchInboundLinksAct) FailureType() string { return "URL_FETCH_INBOUND_LINKS_FAILURE" }

func (FetchInboundLinksAct) Parse(reqId string, data json.RawMessage) ClientRequestAction {
	a := &FetchInboundLinksAct{}
	a.err = json.Unmarshal(data, a)
	return a
}

func (a *FetchInboundLinksAct) Exec() (res *ClientResponse) {
	links, err := ReadSrcLinks(appDB, &Url{Url: a.Url})
	if err != nil {
		return &ClientResponse{
			Type:      a.FailureType(),
			RequestId: a.RequestId,
			Error:     err.Error(),
		}
	}

	return &ClientResponse{
		Type:      a.SuccessType(),
		RequestId: a.RequestId,
		Schema:    "LINK_ARRAY",
		Data:      links,
	}
}

// FetchOutboundLinksAct fetches a url's outbound links
type FetchOutboundLinksAct struct {
	ReqAction
	Url string
}

func (FetchOutboundLinksAct) Type() string        { return "URL_FETCH_OUTBOUND_LINKS_REQUEST" }
func (FetchOutboundLinksAct) SuccessType() string { return "URL_FETCH_OUTBOUND_LINKS_SUCCESS" }
func (FetchOutboundLinksAct) FailureType() string { return "URL_FETCH_OUTBOUND_LINKS_FAILURE" }

func (FetchOutboundLinksAct) Parse(reqId string, data json.RawMessage) ClientRequestAction {
	a := &FetchOutboundLinksAct{}
	a.err = json.Unmarshal(data, a)
	return a
}

func (a *FetchOutboundLinksAct) Exec() (res *ClientResponse) {
	links, err := ReadDstLinks(appDB, &Url{Url: a.Url})
	if err != nil {
		logger.Println(err.Error())
		return &ClientResponse{
			Type:      a.FailureType(),
			RequestId: a.RequestId,
			Error:     err.Error(),
		}
	}

	return &ClientResponse{
		Type:      a.SuccessType(),
		RequestId: a.RequestId,
		Schema:    "LINK_ARRAY",
		Data:      links,
	}
}

// ArchiveUrlAct triggers archiving a url
type ArchiveUrlAct struct {
	ReqAction
	Url string
}

func (ArchiveUrlAct) Type() string        { return "URL_ARCHIVE_REQUEST" }
func (ArchiveUrlAct) SuccessType() string { return "URL_ARCHIVE_SUCCESS" }
func (ArchiveUrlAct) FailureType() string { return "URL_ARCHIVE_FAILURE" }

func (ArchiveUrlAct) Parse(reqId string, data json.RawMessage) ClientRequestAction {
	a := &ArchiveUrlAct{}
	a.RequestId = reqId
	a.err = json.Unmarshal(data, a)
	return a
}

func (a *ArchiveUrlAct) Exec() (res *ClientResponse) {
	url, err := ArchiveUrlSync(appDB, a.Url)
	if err != nil {
		return &ClientResponse{
			Type:      a.FailureType(),
			RequestId: a.RequestId,
			Error:     err.Error(),
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
	ReqAction
	Hash string
}

func (FetchContentUrlsAction) Type() string        { return "CONTENT_URLS_REQUEST" }
func (FetchContentUrlsAction) SuccessType() string { return "CONTENT_URLS_SUCCESS" }
func (FetchContentUrlsAction) FailureType() string { return "CONTENT_URLS_FAILURE" }

func (FetchContentUrlsAction) Parse(reqId string, data json.RawMessage) ClientRequestAction {
	a := &FetchContentUrlsAction{}
	a.RequestId = reqId
	a.err = json.Unmarshal(data, a)
	return a
}

func (a *FetchContentUrlsAction) Exec() (res *ClientResponse) {
	urls, err := UrlsForHash(appDB, a.Hash)
	if err != nil {
		return &ClientResponse{
			Type:      a.FailureType(),
			RequestId: a.RequestId,
			Error:     err.Error(),
		}
	}
	return &ClientResponse{
		Type:      a.SuccessType(),
		RequestId: a.RequestId,
		Schema:    "URL_ARRAY",
		Data:      urls,
	}
}

// FetchContentConsensusAction triggers archiving a url
type FetchContentConsensusAction struct {
	ReqAction
	Hash string
}

func (FetchContentConsensusAction) Type() string        { return "CONTENT_CONSENSUS_REQUEST" }
func (FetchContentConsensusAction) SuccessType() string { return "CONTENT_CONSENSUS_SUCCESS" }
func (FetchContentConsensusAction) FailureType() string { return "CONTENT_CONSENSUS_FAILURE" }

func (FetchContentConsensusAction) Parse(reqId string, data json.RawMessage) ClientRequestAction {
	a := &FetchContentConsensusAction{}
	a.RequestId = reqId
	a.err = json.Unmarshal(data, a)
	return a
}

func (a *FetchContentConsensusAction) Exec() (res *ClientResponse) {
	return &ClientResponse{
		Type:      a.SuccessType(),
		RequestId: a.RequestId,
		Schema:    "CONSENSUS",
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

// FetchMetadataAction triggers archiving a url
type FetchMetadataAction struct {
	ReqAction
	KeyId   string `json:"keyId"`
	Subject string `json:"subject"`
}

func (FetchMetadataAction) Type() string        { return "METADATA_REQUEST" }
func (FetchMetadataAction) SuccessType() string { return "METADATA_SUCCESS" }
func (FetchMetadataAction) FailureType() string { return "METADATA_FAILURE" }

func (FetchMetadataAction) Parse(reqId string, data json.RawMessage) ClientRequestAction {
	a := &FetchMetadataAction{}
	a.RequestId = reqId
	a.err = json.Unmarshal(data, a)
	return a
}

func (a *FetchMetadataAction) Exec() (res *ClientResponse) {
	m, err := LatestMetadata(appDB, a.KeyId, a.Subject)
	if err != nil {
		if err == ErrNotFound {
			return &ClientResponse{
				Type:      a.SuccessType(),
				RequestId: a.RequestId,
			}
		}

		logger.Println(err.Error())
		return &ClientResponse{
			Type:      a.FailureType(),
			RequestId: a.RequestId,
			Error:     err.Error(),
		}
	}

	return &ClientResponse{
		Type:      a.SuccessType(),
		RequestId: a.RequestId,
		Schema:    "METADATA",
		Data:      m,
	}
}

// SaveMetadataAction triggers archiving a url
type SaveMetadataAction struct {
	ReqAction
	KeyId   string                 `json:"keyId"`
	Subject string                 `json:"subject"`
	Meta    map[string]interface{} `json:"meta"`
}

func (SaveMetadataAction) Type() string        { return "METADATA_SAVE_REQUEST" }
func (SaveMetadataAction) SuccessType() string { return "METADATA_SAVE_SUCCESS" }
func (SaveMetadataAction) FailureType() string { return "METADATA_SAVE_FAILURE" }

func (SaveMetadataAction) Parse(reqId string, data json.RawMessage) ClientRequestAction {
	a := &SaveMetadataAction{}
	a.RequestId = reqId
	a.err = json.Unmarshal(data, a)
	return a
}

func (a *SaveMetadataAction) Exec() (res *ClientResponse) {

	m, err := NextMetadata(appDB, a.KeyId, a.Subject)
	if err != nil {
		logger.Println(err.Error())
		return &ClientResponse{
			Type:      a.FailureType(),
			RequestId: a.RequestId,
			Error:     err.Error(),
		}
	}

	m.Meta = a.Meta
	if err := m.Write(appDB); err != nil {
		logger.Println(err.Error())
		return &ClientResponse{
			Type:      a.FailureType(),
			RequestId: a.RequestId,
			Error:     err.Error(),
		}
	}

	return &ClientResponse{
		Type:      a.SuccessType(),
		RequestId: a.RequestId,
		Schema:    "METADATA",
		Data:      a,
	}
}

// FetchPrimersAction grabs a page of primers
type FetchPrimersAction struct {
	ReqAction
	Page     int
	PageSize int
}

func (FetchPrimersAction) Type() string        { return "PRIMERS_FETCH_REQUEST" }
func (FetchPrimersAction) SuccessType() string { return "PRIMERS_FETCH_SUCCESS" }
func (FetchPrimersAction) FailureType() string { return "PRIMERS_FETCH_FAILURE" }

func (FetchPrimersAction) Parse(reqId string, data json.RawMessage) ClientRequestAction {
	a := &FetchPrimersAction{}
	a.RequestId = reqId
	a.err = json.Unmarshal(data, a)
	return a
}

func (a *FetchPrimersAction) Exec() (res *ClientResponse) {
	// for now we're proxying domains as "primers", more bridging work
	// is needed
	domains, err := ListDomains(appDB, 50, 0)
	if err != nil {
		logger.Println(err.Error())
		return &ClientResponse{
			Type:      a.FailureType(),
			RequestId: a.RequestId,
			Error:     err.Error(),
		}
	}

	return &ClientResponse{
		Type:      a.SuccessType(),
		RequestId: a.RequestId,
		Schema:    "PRIMER_ARRAY",
		Data:      domains,
	}
}

// FetchPrimerAction grabs a page of primers
type FetchPrimerAction struct {
	ReqAction
	Host string
}

func (FetchPrimerAction) Type() string        { return "PRIMER_FETCH_REQUEST" }
func (FetchPrimerAction) SuccessType() string { return "PRIMER_FETCH_SUCCESS" }
func (FetchPrimerAction) FailureType() string { return "PRIMER_FETCH_FAILURE" }

func (FetchPrimerAction) Parse(reqId string, data json.RawMessage) ClientRequestAction {
	a := &FetchPrimerAction{}
	a.RequestId = reqId
	a.err = json.Unmarshal(data, a)
	return a
}

func (a *FetchPrimerAction) Exec() (res *ClientResponse) {
	// for now we're proxying domains as "primers", more bridging work
	// is needed
	d := &Domain{Host: a.Host}
	if err := d.Read(appDB); err != nil {
		logger.Println(err.Error())
		return &ClientResponse{
			Type:      a.FailureType(),
			RequestId: a.RequestId,
			Error:     err.Error(),
		}
	}

	return &ClientResponse{
		Type:      a.SuccessType(),
		RequestId: a.RequestId,
		Schema:    "PRIMER",
		Data:      d,
	}
}
