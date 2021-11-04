package flow

import "github.com/prometheus/client_golang/prometheus"

var (
	_metric = newMetrics()
)

type CoreMetrics struct {
	counter     *prometheus.CounterVec
	performance *prometheus.HistogramVec
}

func Metrics() *CoreMetrics {
	return _metric
}

func (m *CoreMetrics) NewTimer(stage, action string) *prometheus.Timer {
	return prometheus.NewTimer(m.NewObserver(stage, action))
}

func (m *CoreMetrics) NewObserver(stage, action string) prometheus.Observer {
	return m.performance.WithLabelValues(stage, action)
}

func (m *CoreMetrics) NewCounter(source, action string) prometheus.Counter {
	return m.counter.WithLabelValues(source, action)
}

func newMetrics() *CoreMetrics {
	const ns = "flow"
	return &CoreMetrics{
		counter: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Subsystem: "core",
			Name:      "count",
			Help:      "核心事件统计",
		}, []string{"source", "eventType"}),
		performance: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: ns,
			Subsystem: "core",
			Name:      "action_duration",
			Help:      "核心处理性能",
			Buckets:   prometheus.LinearBuckets(1, 0.5, 50),
		}, []string{"stage", "action"}),
	}
}

func init() {
	prometheus.MustRegister(_metric.counter, _metric.performance)
}
