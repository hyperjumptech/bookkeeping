package logger

import (
	"strings"

	"github.com/hyperjumptech/hyperwallet/internal/config"
	log "github.com/sirupsen/logrus"
)

// ConfigureLogging set logging lever from config
func ConfigureLogging() {
	lLevel := config.Get("server.log.level")
	log.SetFormatter(&log.JSONFormatter{})
	log.Info("Setting log level to: ", lLevel)
	switch strings.ToUpper(lLevel) {
	default:
		log.Info("Unknown level [", lLevel, "]. Log level set to ERROR")
		log.SetLevel(log.ErrorLevel)
	case "TRACE":
		log.SetLevel(log.TraceLevel)
	case "DEBUG":
		log.SetLevel(log.DebugLevel)
	case "INFO":
		log.SetLevel(log.InfoLevel)
	case "WARN":
		log.SetLevel(log.WarnLevel)
	case "ERROR":
		log.SetLevel(log.ErrorLevel)
	case "FATAL":
		log.SetLevel(log.FatalLevel)
	}
}
