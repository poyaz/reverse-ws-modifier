package domain

import (
	"net/http"
)

type FindMatch int

const (
	ExactMatch FindMatch = iota + 1
	RegexMatch
	PrefixMatch
)

type WsProxyTable struct {
	Type FindMatch
	Host string
	URI  string
	Ws   WsProxyUsecase
}

type WsReqInfo struct {
	Host   string
	Header http.Header
	URI    string
}

type WsProxyTableUsecase interface {
	Connect(info WsReqInfo) (WsProxyUsecase, error)
}

type ModifierEvent struct {
	On      OpcodeType
	Handler ModifierFunc
}

type ModifierFunc func(frame Frame) (Frame, error)

type WsProxyUsecase interface {
	Proxy(writer http.ResponseWriter, request *http.Request)
}
