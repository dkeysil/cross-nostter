package main

import (
	"github.com/dkeysil/cross-nostter/internal/config"
	"github.com/dkeysil/cross-nostter/internal/service"
	"github.com/kelseyhightower/envconfig"
)

const (
	prefix = "CROSS_NOSTTER"
)

func main() {
	cfg := config.Config{}

	envconfig.MustProcess(prefix, &cfg)

	service.RunApplication(cfg)
}
