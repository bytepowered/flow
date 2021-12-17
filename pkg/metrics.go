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

func (m *coremetrics) NewTimer(tag, action string) *prometheus.Timer {
	return prometheus.NewTimer(m.NewObserver(tag, action))
}

func (m *coremetrics) NewObserver(tag, action string) prometheus.Observer {
	return m.performance.WithLabelValues(tag, action)
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
			Name:      "event_count",
			Help:      "指定Tag的事件计数",
		}, []string{"tag", "action"}),
		performance: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: ns,
			Subsystem: "core",
			Name:      "action_duration",
			Help:      "指定Tag的事件计数处理性能",
			Buckets:   prometheus.LinearBuckets(1, 0.5, 50),
		}, []string{"tag", "action"}),
	}
}

func init() {
	prometheus.MustRegister(_metric.counter, _metric.performance)
}
