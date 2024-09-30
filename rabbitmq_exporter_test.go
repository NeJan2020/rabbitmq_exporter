package rabbitmq_exporter

import (
	"io"

	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetOutput(io.Discard)
}
