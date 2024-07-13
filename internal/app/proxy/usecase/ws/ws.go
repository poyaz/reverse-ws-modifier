package ws

import (
	"errors"
	"github.com/poyaz/reverse-ws-modifier/internal/app/proxy/usecase/adapter"
	"github.com/poyaz/reverse-ws-modifier/internal/domain"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

type ws struct {
	ws  adapter.WsAdapter
	opt Config
}

var _ domain.WsProxyTableUsecase = (*ws)(nil)

func NewWs(wsInfra adapter.WsAdapter, config ...Config) (*ws, error) {
	var opt Config
	for _, cfg := range config {
		opt = cfg
	}

	return &ws{ws: wsInfra, opt: opt}, nil
}

func (w *ws) Connect(info domain.WsReqInfo) (domain.WsProxyUsecase, error) {
	var remHost string
	remOriginHed, ok := info.Header["origin"]
	if ok && len(remOriginHed) > 0 {
		remHost = remOriginHed[0]
	} else {
		remHost = info.Host
	}

	err, upstream, isFind := w.findUpstreamByPath(info.URI)
	if err != nil {
		return nil, err
	}
	if !isFind {
		return nil, errors.New("upstream not found")
	}

	upstreamAddr := "ws://" + upstream.Ip + ":" + strconv.Itoa(upstream.Port) + info.URI
	if upstream.Override.Host != "" {
		remHost = upstream.Override.Host
	}
	overridePayload, err := initOverridePayload(upstream.Override.WebsocketPayload)
	if err != nil {
		return nil, err
	}
	wsp, err := w.ws.New(
		upstreamAddr,
		remHost,
		func(r *http.Request) error {
			for _, oh := range upstream.Override.Header {
				r.Header.Set(oh.Key, oh.Value)
			}
			return nil
		},
		overridePayload...,
	)

	return wsp, err
}

func (w *ws) findUpstreamByPath(url string) (error, UpstreamConfig, bool) {
	for _, s := range w.opt.Servers {
		for _, mp := range s.MatchPath {
			if mp.Type == domain.ExactMatch && mp.Value == url {
				return nil, s.Upstream, true
			}
			if mp.Type == domain.PrefixMatch && strings.HasPrefix(url, mp.Value) {
				return nil, s.Upstream, true
			}
			if mp.Type == domain.RegexMatch {
				match, err := regexp.MatchString(mp.Value, url)
				if err != nil {
					return err, UpstreamConfig{}, false
				}

				return nil, s.Upstream, match
			}
		}
	}

	return nil, UpstreamConfig{}, false
}

func initOverridePayload(override []WebsocketPayloadOverrideConfig) ([]domain.ModifierEvent, error) {
	var overridePayload []domain.ModifierEvent

	for _, o := range override {
		handler := func(frame domain.Frame) (domain.Frame, error) {
			return frame, nil
		}
		if o.Type == domain.ExactMatch {
			handler = func(frame domain.Frame) (domain.Frame, error) {
				if o.Match != string(frame.Payload) {
					return frame, nil
				}
				frame.Payload = []byte(o.Value)
				frame.Length = uint64(len(frame.Payload))

				return frame, nil
			}
		} else if o.Type == domain.RegexMatch {
			rp, err := regexp.Compile(o.Match)
			if err != nil {
				return nil, err
			}

			handler = func(frame domain.Frame) (domain.Frame, error) {
				frame.Payload = []byte(rp.ReplaceAllString(string(frame.Payload), o.Value))
				frame.Length = uint64(len(frame.Payload))

				return frame, nil
			}
		}

		overridePayload = append(
			overridePayload,
			domain.ModifierEvent{
				On:      domain.TextOpcode,
				Handler: handler,
			},
		)
	}

	return overridePayload, nil
}
