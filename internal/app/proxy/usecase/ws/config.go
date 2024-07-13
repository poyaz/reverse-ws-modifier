package ws

import "github.com/poyaz/reverse-ws-modifier/internal/domain"

type Config struct {
	Servers []ServersConfig
}

type ServersConfig struct {
	MatchPath []MatchPathConfig
	Upstream  UpstreamConfig
}

type MatchPathConfig struct {
	Type  domain.FindMatch
	Value string
}

type UpstreamConfig struct {
	Ip       string
	Port     int
	Override OverrideConfig
}

type OverrideConfig struct {
	Host             string
	Header           []HeaderOverrideConfig
	WebsocketPayload []WebsocketPayloadOverrideConfig
}

type HeaderOverrideConfig struct {
	Key   string
	Value string
}

type WebsocketPayloadOverrideConfig struct {
	Type  domain.FindMatch
	Match string
	Value string
}
