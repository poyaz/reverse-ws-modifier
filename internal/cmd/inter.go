package cmd

type RunBootstrap interface {
	Run() error
}

type ShutdownBootstrap interface {
	Shutdown() error
}
