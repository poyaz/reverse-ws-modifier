package cmd

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"

	"github.com/poyaz/reverse-ws-modifier/config"
	httpDeliveryProxy "github.com/poyaz/reverse-ws-modifier/internal/app/proxy/delivery/http"
	adapterUsecaseProxy "github.com/poyaz/reverse-ws-modifier/internal/app/proxy/usecase/adapter"
	wsUsecaseProxy "github.com/poyaz/reverse-ws-modifier/internal/app/proxy/usecase/ws"
	"github.com/poyaz/reverse-ws-modifier/internal/domain"
	infraWs "github.com/poyaz/reverse-ws-modifier/internal/infra/ws"
)

var wsInfraProxyImp adapterUsecaseProxy.WsAdapter
var wsUsecaseProxyImp domain.WsProxyTableUsecase
var logger *logrus.Logger

var shutdownHandlers []ShutdownBootstrap

func Run(cfg *config.Config) error {
	var err error
	gracefulShutdown := make(chan os.Signal, 1)
	signal.Notify(gracefulShutdown, syscall.SIGINT, syscall.SIGTERM)

	logger = logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	switch cfg.Data.Global.LogLevel {
	case "panic":
		logger.SetLevel(logrus.PanicLevel)
	case "fatal":
		logger.SetLevel(logrus.FatalLevel)
	case "error", "err":
		logger.SetLevel(logrus.ErrorLevel)
	case "warning", "warn":
		logger.SetLevel(logrus.WarnLevel)
	case "info":
		logger.SetLevel(logrus.InfoLevel)
	case "debugging", "debug":
		logger.SetLevel(logrus.DebugLevel)
	case "tracing", "trace":
		logger.SetLevel(logrus.TraceLevel)
	}

	wsInfraProxyImp, err = infraWs.NewWsInfra()
	if err != nil {
		return err
	}

	if err := runWebsocketProxyUsecase(cfg); err != nil {
		return err
	}

	if err := runDelivery(cfg); err != nil {
		return err
	}

	<-gracefulShutdown
	_, _ = os.Stdout.Write([]byte{'\n'})

	for _, sh := range shutdownHandlers {
		if err := sh.Shutdown(); err != nil {
			logger.Error(err)
		}
	}

	return nil
}

func runDelivery(cfg *config.Config) error {
	listenerConfig := make(map[httpDeliveryProxy.Config]bool)
	for _, server := range cfg.Data.Servers {
		listenerConfig[httpDeliveryProxy.Config{ListenIP: server.Ip, ListenPort: server.Port}] = true
	}

	for k, _ := range listenerConfig {
		httpConfig := httpDeliveryProxy.Config{
			ListenIP:   k.ListenIP,
			ListenPort: k.ListenPort,
		}
		httpDelivery, err := httpDeliveryProxy.NewHandler(wsUsecaseProxyImp, logger, httpConfig)
		if err != nil {
			return err
		}
		shutdownHandlers = append(shutdownHandlers, httpDelivery)

		go func(handler RunBootstrap) {
			if err := handler.Run(); err != nil {
				logger.Fatal(err)
			}
		}(httpDelivery)
	}

	return nil
}

func runWebsocketProxyUsecase(cfg *config.Config) (err error) {
	wsConfig := wsUsecaseProxy.Config{}
	for _, server := range cfg.Data.Servers {
		var matchPaths []wsUsecaseProxy.MatchPathConfig
		for _, smp := range server.Match.Path {
			mp := wsUsecaseProxy.MatchPathConfig{Value: smp.Value}
			switch smp.Type {
			case "exact":
				mp.Type = domain.ExactMatch
			case "prefix":
				mp.Type = domain.PrefixMatch
			case "regex":
				mp.Type = domain.RegexMatch
			}
			matchPaths = append(matchPaths, mp)
		}

		upstreamConf := wsUsecaseProxy.UpstreamConfig{
			Ip:   server.Upstream.Ip,
			Port: server.Upstream.Port,
			Override: wsUsecaseProxy.OverrideConfig{
				Host: server.Upstream.Override.Host,
			},
		}
		for _, header := range server.Upstream.Override.Headers {
			upstreamConf.Override.Header = append(
				upstreamConf.Override.Header,
				wsUsecaseProxy.HeaderOverrideConfig{Key: header.Key, Value: header.Value},
			)
		}
		for _, wsPayload := range server.Upstream.Override.WebsocketPayload {
			wsPayloadConf := wsUsecaseProxy.WebsocketPayloadOverrideConfig{
				Match: wsPayload.Match,
				Value: wsPayload.Value,
			}
			switch wsPayload.Type {
			case "exact":
				wsPayloadConf.Type = domain.ExactMatch
			case "regex":
				wsPayloadConf.Type = domain.RegexMatch
			}
			upstreamConf.Override.WebsocketPayload = append(upstreamConf.Override.WebsocketPayload, wsPayloadConf)
		}

		serverConf := wsUsecaseProxy.ServersConfig{
			MatchPath: matchPaths,
			Upstream:  upstreamConf,
		}
		wsConfig.Servers = append(wsConfig.Servers, serverConf)
	}
	wsUsecaseProxyImp, err = wsUsecaseProxy.NewWs(wsInfraProxyImp, wsConfig)
	if err != nil {
		return err
	}

	return nil
}
