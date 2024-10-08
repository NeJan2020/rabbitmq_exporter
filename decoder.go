package rabbitmq_exporter

// MetricMap maps name to float64 metric
type MetricMap map[string]float64

// StatsInfo describes  one statistic (queue or exchange): its name, vhost it belongs to, and all associated metrics.
type StatsInfo struct {
	labels  map[string]string
	metrics MetricMap
}

// RabbitReply is an inteface responsible for extracting usable
// information from RabbitMQ HTTP API replies, independent of the
// actual transfer format used.
type RabbitReply interface {
	// MakeMap makes a flat map from string to float values from a
	// RabbitMQ reply. Processing happens recursively and nesting
	// is represented by '.'-separated keys. Entries are added
	// only for values that can be reasonably converted to float
	// (numbers and booleans). Failure to parse should result in
	// an empty result map.
	MakeMap() MetricMap

	// MakeStatsInfo parses a list of details about some named
	// RabbitMQ objects (i.e. list of queues, exchanges, etc.).
	// Failure to parse should result in an empty result list.
	MakeStatsInfo([]string) []StatsInfo

	// GetString returns the string value for the given key
	// If the key cannot be found the second return is false
	GetString(key string) (string, bool)
}

// MakeReply instantiates the apropriate reply parser for a given
// reply and the content-type header
func MakeReply(contentType string, body []byte) (RabbitReply, error) {
	if contentType == "application/bert" {
		return makeBERTReply(body)
	}
	return makeJSONReply(body)
}
