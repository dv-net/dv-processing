package metrics

import (
	"errors"

	"github.com/dv-net/dv-processing/internal/models"
	"github.com/prometheus/client_golang/prometheus"
)

// const metricsNamespace = "processing"

type PrometheusMetrics struct{}

func New(_ prometheus.Registerer) *PrometheusMetrics {
	pm := &PrometheusMetrics{}

	// registerer.MustRegister(pm.)

	return pm
}

func (m *PrometheusMetrics) ChangeValue(_ models.PrometheusMetricType, _ any) error {
	return errors.New("not implemented yet")
}

type PrometheusMetric interface {
	Set(float64)
}

type errorPrometheusMetric struct {
	prometheus.Gauge
}

func (epm *errorPrometheusMetric) Set(value float64) {
	epm.Gauge.Set(value)
}

func (epm *errorPrometheusMetric) Type() models.PrometheusMetricType {
	return models.ErrorPrometheusMetricType
}

func NewErrorPrometheusMetric(metricsNamespace string, name string, help string) PrometheusMetric {
	epm := &errorPrometheusMetric{
		Gauge: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: metricsNamespace,
				Name:      name,
				Help:      help,
			},
		),
	}
	return epm
}
