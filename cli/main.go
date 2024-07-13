package main

import (
	"log"
	"os"

	_ "go.uber.org/automaxprocs"

	"github.com/poyaz/reverse-ws-modifier/config"
	"github.com/poyaz/reverse-ws-modifier/internal/cmd"
)

func main() {
	cfg := config.NewConfig()
	if err := cfg.ParseFlags(os.Args[1:]); err != nil {
		log.Fatalf("flag parsing error: %v", err)
	}

	if err := cmd.Run(cfg); err != nil {
		log.Fatal(err)
	}
}
