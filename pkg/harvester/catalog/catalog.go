package catalog

import (
	"context"
	"github.com/HewlettPackard/galadriel/pkg/common/telemetry"
	"github.com/sirupsen/logrus"
)

type Catalog interface {
	// methods for accessing pluggable interfaces (e.g., NodeAttestors)
	// no needed for a PoC.
}

type Repository struct {
	log logrus.FieldLogger
}

func (r *Repository) Close() {
	// TODO: close repository
}

type Config struct {
	Log     logrus.FieldLogger
	Metrics telemetry.MetricServer
}

func Load(ctx context.Context, config Config) (*Repository, error) {
	re := &Repository{
		log: config.Log,
	}

	return re, nil
}
