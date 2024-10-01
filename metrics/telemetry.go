// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 Canonical Ltd.

/*
 *  Metrics package is used to expose the metrics of the NSSF service.
 */

package metrics

import (
	"net/http"

	"github.com/omec-project/nssf/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// NssfStats captures NSSF stats
type NssfStats struct {
	nssfNsSelections *prometheus.CounterVec
}

var nssfStats *NssfStats

func initNssfStats() *NssfStats {
	return &NssfStats{
		nssfNsSelections: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "nssf_ns_selections",
			Help: "Counter of total NS selection queries",
		}, []string{"target_nf_type", "nf_id", "result"}),
	}
}

func (ps *NssfStats) register() error {
	if err := prometheus.Register(ps.nssfNsSelections); err != nil {
		return err
	}
	return nil
}

func init() {
	nssfStats = initNssfStats()

	if err := nssfStats.register(); err != nil {
		logger.InitLog.Errorln("NSSF Stats register failed")
	}
}

// InitMetrics initialises NSSF metrics
func InitMetrics() {
	http.Handle("/metrics", promhttp.Handler())
	if err := http.ListenAndServe(":8080", nil); err != nil {
		logger.InitLog.Errorf("Could not open metrics port: %v", err)
	}
}

// IncrementNssfNsSelectionsStats increments number of total NS selection queries
func IncrementNssfNsSelectionsStats(targetNfType, nfId, result string) {
	nssfStats.nssfNsSelections.WithLabelValues(targetNfType, nfId, result).Inc()
}
