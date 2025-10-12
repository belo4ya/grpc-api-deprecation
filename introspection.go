package apideprecation

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Internal metrics for interceptor runtime/limits.
var (
	hitMaxItemsPerCollection = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_deprecated_field_usage_hit_max_items_per_collection_total",
			Help: "Number of times element iteration was cut due to maxItemsPerCollection constant.",
		},
		[]string{"grpc_type", "grpc_service", "grpc_method", "field", "collection_type", "max_items"},
	)
)

func hitMaxItemsPerCollectionInc(typ, service, method, field, collectionType string, maxItems int) {
	hitMaxItemsPerCollection.WithLabelValues(typ, service, method, field, collectionType, strconv.Itoa(maxItems)).Inc()
}
