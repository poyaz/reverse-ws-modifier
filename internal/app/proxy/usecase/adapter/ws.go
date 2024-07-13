package adapter

import (
	"github.com/poyaz/reverse-ws-modifier/internal/domain"
	"net/http"
)

type WsAdapter interface {
	New(addr string, rewriteHost string, beforeCallback func(r *http.Request) error, events ...domain.ModifierEvent) (domain.WsProxyUsecase, error)
}
