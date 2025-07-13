package main

import (
	"fmt"
	"time"

	"github.com/LdDl/greenwave/app/configuration"
	"github.com/LdDl/greenwave/app/rest"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func init() {
	zerolog.TimeFieldFormat = time.RFC3339
}

// @title Microservice for green waves management for traffic lights
// @version 1.0.0

// @BasePath /

// @schemes https http
func main() {
	mainCfg, err := configuration.PrepareConfiguration()
	if err != nil {
		log.Error().Err(err).Msg("Can't prepare configuration")
		return
	}

	microservice := rest.Setup(mainCfg)
	microserviceAddr := fmt.Sprintf(":%d", mainCfg.ServerCfg.Port)
	log.Info().Str("scope", "api").Msg(fmt.Sprintf("Start microservice on URL: '%s'", microserviceAddr))
	err = microservice.Start(microserviceAddr)
	if err != nil {
		log.Error().Err(err).Str("scope", "api").Msg("Can't start REST API")
	}
}
