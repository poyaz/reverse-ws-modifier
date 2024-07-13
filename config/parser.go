package config

import (
	"github.com/alecthomas/kingpin/v2"
	"github.com/gookit/config/v2"
	"github.com/gookit/config/v2/yaml"
	"strings"
)

var defaultConfig = &Config{
	Config: "config.yaml",
}

func NewConfig() *Config {
	return &Config{}
}

func (cfg *Config) ParseFlags(args []string) error {
	app := kingpin.New("reverse-ws-modifier", "This is a ws-modifier reverse proxy")
	app.Version(Version)
	app.DefaultEnvars()

	app.Flag("config", "The config file (Default: ./config.yaml)").Default(defaultConfig.Config).StringVar(&cfg.Config)

	_, err := app.Parse(args)
	if err != nil {
		return err
	}

	if err := cfg.parseConfig(); err != nil {
		return err
	}

	return nil
}

func (cfg *Config) parseConfig() error {
	config.WithOptions(config.ParseDefault)
	c := config.New("test").WithOptions(config.ParseDefault).WithDriver(yaml.Driver)
	if err := c.LoadFiles(cfg.Config); err != nil {
		return err
	}

	data := Data{}
	if err := c.Decode(&data); err != nil {
		return err
	}

	data.Global.LogLevel = strings.ToLower(data.Global.LogLevel)

	cfg.Data = data

	return nil
}
