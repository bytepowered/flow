package flow

import "github.com/prometheus/client_golang/prometheus"

var _metric = newMetrics()

type coremetrics struct {
	counter     *prometheus.CounterVec
	performance *prometheus.HistogramVec
}

func metrics() *coremetrics {
	return _metric
}

func (m *coremetrics) NewTimer(stage, action string) *prometheus.Timer {
	return prometheus.NewTimer(m.NewObserver(stage, action))
}

func (m *coremetrics) NewObserver(stage, action string) prometheus.Observer {
	return m.performance.WithLabelValues(stage, action)
}

func (m *coremetrics) NewCounter(tag, action string) prometheus.Counter {
	return m.counter.WithLabelValues(tag, action)
}

func newMetrics() *coremetrics {
	const ns = "flow"
	return &coremetrics{
		counter: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Subsystem: "core",
			Name:      "count",
			Help:      "Source action with tag",
		}, []string{"tag", "action"}),
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
