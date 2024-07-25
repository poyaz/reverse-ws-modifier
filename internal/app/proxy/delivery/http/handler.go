package http

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/poyaz/reverse-ws-modifier/internal/domain"
)

type handler struct {
	wsUsecase domain.WsProxyTableUsecase
	log       logrus.FieldLogger
	server    *http.Server
	opt       Config
}

func NewHandler(wsUsecase domain.WsProxyTableUsecase, log *logrus.Logger, config ...Config) (*handler, error) {
	var opt Config
	for _, cfg := range config {
		opt = cfg
	}

	listenAddr := opt.ListenIP + ":" + strconv.Itoa(opt.ListenPort)
	server := &http.Server{
		Addr: listenAddr,
	}

	h := &handler{
		wsUsecase: wsUsecase,
		log:       log,
		server:    server,
		opt:       opt,
	}

	return h, nil
}

func (h *handler) Run() error {
	h.server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		info := domain.WsReqInfo{
			Host:   r.Host,
			Header: r.Header,
			URI:    r.URL.RequestURI(),
		}
		ws, err := h.wsUsecase.Connect(info)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			if _, err := w.Write([]byte(err.Error())); err != nil {
				h.log.Error(err)
			}
			return
		}
		h.log.WithFields(logrus.Fields{"host": r.Host, "uri": r.URL.RequestURI()}).Info("New request income")

		ws.Proxy(w, r)
	})

	h.log.Info("Start server listen on " + h.server.Addr)
	if err := h.server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func (h *handler) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := h.server.Shutdown(ctx); err != nil {
		return err
	}

	h.log.Info("Stop server listen on " + h.server.Addr)

	return nil
}
