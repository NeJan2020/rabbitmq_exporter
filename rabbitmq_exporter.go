package rabbitmq_exporter

import (
	"bytes"
	"context"
	"flag"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

const (
	defaultLogLevel = logrus.InfoLevel
)

var log *logrus.Logger = logrus.New()

func SetLogger(l *logrus.Logger) {
	log = l
}

func InitLogger(config *RabbitExporterConfig) *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(getLogLevel())
	if strings.ToUpper(config.OutputFormat) == "JSON" {
		logger.SetFormatter(&logrus.JSONFormatter{})
	} else {
		// The TextFormatter is default, you don't actually have to do this.
		logger.SetFormatter(&logrus.TextFormatter{})
	}

	return logger
}

func main() {
	var checkURL = flag.String("check-url", "", "Curl url and return exit code (http: 200 => 0, otherwise 1)")
	var configFile = flag.String("config-file", "conf/rabbitmq.conf", "path to json config")
	flag.Parse()

	if *checkURL != "" { // do a single http get request. Used in docker healthckecks as curl is not inside the image
		curl(*checkURL)
		return
	}

	var config *RabbitExporterConfig

	config, err := initConfigFromFile(*configFile)          //Try parsing config file
	if _, isPathError := err.(*os.PathError); isPathError { // No file => use environment variables
		config = initConfig()
	} else if err != nil {
		panic(err)
	}

	log = InitLogger(config)
	exporter := NewExporter(config)
	prometheus.MustRegister(exporter)

	log.WithFields(logrus.Fields{
		"VERSION":    Version,
		"REVISION":   Revision,
		"BRANCH":     Branch,
		"BUILD_DATE": BuildDate,
		//		"RABBIT_PASSWORD": config.RABBIT_PASSWORD,
	}).Info("Starting RabbitMQ exporter")

	log.WithFields(logrus.Fields{
		"PUBLISH_ADDR":        config.PublishAddr,
		"PUBLISH_PORT":        config.PublishPort,
		"RABBIT_URL":          config.RabbitURL,
		"RABBIT_USER":         config.RabbitUsername,
		"RABBIT_CONNECTION":   config.RabbitConnection,
		"OUTPUT_FORMAT":       config.OutputFormat,
		"RABBIT_CAPABILITIES": formatCapabilities(config.RabbitCapabilities),
		"RABBIT_EXPORTERS":    config.EnabledExporters,
		"CAFILE":              config.CAFile,
		"CERTFILE":            config.CertFile,
		"KEYFILE":             config.KeyFile,
		"SKIPVERIFY":          config.InsecureSkipVerify,
		"EXCLUDE_METRICS":     config.ExcludeMetrics,
		"SKIP_EXCHANGES":      config.SkipExchanges.String(),
		"INCLUDE_EXCHANGES":   config.IncludeExchanges.String(),
		"SKIP_QUEUES":         config.SkipQueues.String(),
		"INCLUDE_QUEUES":      config.IncludeQueues.String(),
		"SKIP_VHOST":          config.SkipVHost.String(),
		"INCLUDE_VHOST":       config.IncludeVHost.String(),
		"RABBIT_TIMEOUT":      config.Timeout,
		"MAX_QUEUES":          config.MaxQueues,
		//		"RABBIT_PASSWORD": config.RABBIT_PASSWORD,
	}).Info("Active Configuration")

	handler := http.NewServeMux()
	handler.Handle("/metrics", promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{}))
	handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html>
             <head><title>RabbitMQ Exporter</title></head>
             <body>
             <h1>RabbitMQ Exporter</h1>
             <p><a href='/metrics'>Metrics</a></p>
             </body>
             </html>`))
	})
	handler.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if exporter.LastScrapeOK() {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusGatewayTimeout)
		}
	})

	server := &http.Server{Addr: config.PublishAddr + ":" + config.PublishPort, Handler: handler}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()

	<-runService()
	log.Info("Shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := server.Shutdown(ctx); err != nil {
		log.Fatal(err)
	}
	cancel()
}

func getLogLevel() logrus.Level {
	lvl := strings.ToLower(os.Getenv("LOG_LEVEL"))
	level, err := logrus.ParseLevel(lvl)
	if err != nil {
		level = defaultLogLevel
	}
	return level
}

func formatCapabilities(caps RabbitCapabilitySet) string {
	var buffer bytes.Buffer
	first := true
	for k := range caps {
		if !first {
			buffer.WriteString(",")
		}
		first = false
		buffer.WriteString(string(k))
	}
	return buffer.String()
}
