package config

var Version = "unknown"

type Config struct {
	Config string
	Data   Data
}

type Data struct {
	Global  GlobalConfig
	Servers []ServerConfig `default:""`
}

type GlobalConfig struct {
	LogLevel string `default:"info"`
}

type ServerConfig struct {
	Ip       string `default:"0.0.0.0"`
	Port     int    `default:"80"`
	Match    ServerMatchUrlConfig
	Upstream ServerUpstreamConfig
}

type ServerMatchUrlConfig struct {
	Path []ServerMatchConfig
}

type ServerMatchConfig struct {
	Type  string
	Value string
}

type ServerUpstreamConfig struct {
	Ip       string
	Port     int
	Override ServerUpstreamOverrideConfig
}

type ServerUpstreamOverrideConfig struct {
	Host             string
	Headers          []ServerUpstreamOverrideHeadersConfig
	WebsocketPayload []ServerUpstreamOverrideWebsocketPayloadConfig
}

type ServerUpstreamOverrideHeadersConfig struct {
	Key   string
	Value string
}

type ServerUpstreamOverrideWebsocketPayloadConfig struct {
	Type  string `default:"exact"`
	Match string
	Value string
}
