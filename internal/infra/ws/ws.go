package ws

import (
	"bufio"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/poyaz/reverse-ws-modifier/internal/app/proxy/usecase/adapter"
	"github.com/poyaz/reverse-ws-modifier/internal/domain"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const (
	WsScheme  = "ws"
	WssScheme = "wss"
	BufSize   = 1024 * 32
)

var ErrFormatAddr = errors.New("remote websockets addr format error")

var _ domain.WsProxyUsecase = (*WebsocketProxy)(nil)

type WebsocketProxy struct {
	scheme          string
	remoteAddr      string
	rewriteHost     string
	defaultPath     string
	tlsc            *tls.Config
	logger          *log.Logger
	beforeHandshake func(r *http.Request) error
	events          []domain.ModifierEvent
}

var _ adapter.WsAdapter = (*wsInfra)(nil)

type wsInfra struct{}

func NewWsInfra() (*wsInfra, error) {
	return &wsInfra{}, nil
}

func (w *wsInfra) New(addr string, rewriteHost string, beforeCallback func(r *http.Request) error, events ...domain.ModifierEvent) (domain.WsProxyUsecase, error) {
	u, err := url.Parse(addr)
	if err != nil {
		return nil, ErrFormatAddr
	}
	host, port, err := net.SplitHostPort(u.Host)
	if err != nil {
		return nil, ErrFormatAddr
	}
	if u.Scheme != WsScheme && u.Scheme != WssScheme {
		return nil, ErrFormatAddr
	}
	wp := &WebsocketProxy{
		scheme:          u.Scheme,
		remoteAddr:      fmt.Sprintf("%s:%s", host, port),
		rewriteHost:     rewriteHost,
		beforeHandshake: beforeCallback,
		logger:          log.New(os.Stderr, "", log.LstdFlags),
		events:          events,
	}
	if u.Scheme == WssScheme {
		wp.tlsc = &tls.Config{InsecureSkipVerify: true}
	}

	return wp, nil
}

func (wp *WebsocketProxy) Proxy(writer http.ResponseWriter, request *http.Request) {
	if strings.ToLower(request.Header.Get("Connection")) != "upgrade" ||
		strings.ToLower(request.Header.Get("Upgrade")) != "websocket" {
		_, _ = writer.Write([]byte(`Must be a websocket request`))
		return
	}
	hijacker, ok := writer.(http.Hijacker)
	if !ok {
		return
	}
	downstreamConn, bufrw, err := hijacker.Hijack()
	if err != nil {
		return
	}
	defer downstreamConn.Close()
	req := request.Clone(request.Context())
	req.Host = wp.rewriteHost
	if wp.beforeHandshake != nil {
		// Add headers, permission authentication + masquerade sources
		err = wp.beforeHandshake(req)
		if err != nil {
			_, _ = writer.Write([]byte(err.Error()))
			return
		}
	}
	var upstreamConn net.Conn
	switch wp.scheme {
	case WsScheme:
		upstreamConn, err = net.Dial("tcp", wp.remoteAddr)
	case WssScheme:
		upstreamConn, err = tls.Dial("tcp", wp.remoteAddr, wp.tlsc)
	}
	if err != nil {
		_, _ = writer.Write([]byte(err.Error()))
		return
	}
	defer upstreamConn.Close()
	err = req.Write(upstreamConn)
	if err != nil {
		_, _ = writer.Write([]byte(err.Error()))
		return
	}

	downstreamWs := wsConn{downstreamConn, bufrw, req.Header, 1000}
	upstreamWs := wsConn{conn: upstreamConn, bufrw: bufio.NewReadWriter(bufio.NewReader(upstreamConn), bufio.NewWriter(upstreamConn)), status: 1000}

	var textOpcodeEvents []domain.ModifierFunc
	for _, event := range wp.events {
		if event.On == domain.TextOpcode {
			textOpcodeEvents = append(textOpcodeEvents, event.Handler)
		}
	}

	errChan := make(chan error, 2)
	go func() {
		for {
			f, err := downstreamWs.recv()
			if err != nil {
				errChan <- err
				return
			}

			switch f.Opcode {
			case domain.CloseOpcode:
				return
			case domain.PingOpcode:
				f.Opcode = domain.PongOpcode
			case domain.ContinuationOpcode:
			case domain.TextOpcode:
				for _, textOpcodeEvent := range textOpcodeEvents {
					orFr, err := textOpcodeEvent(f)
					if err != nil {
						errChan <- err
						return
					}

					f = orFr
				}
			case domain.BinaryOpcode:
			}

			if err = upstreamWs.send(f); err != nil {
				errChan <- err
				return
			}
		}
	}()
	go func() {
		buf := make([]byte, 300)
		_, err := io.CopyBuffer(downstreamConn, upstreamConn, buf)
		errChan <- err
	}()

	select {
	case err = <-errChan:
		if err != nil {
			_, _ = writer.Write([]byte(err.Error()))
		}
	}
}
