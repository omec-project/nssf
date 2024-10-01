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
	nssfNssaiAvailability              *prometheus.CounterVec
	nssfNssaiAvailabilitySubscriptions *prometheus.CounterVec
	nssfNsSelections                   *prometheus.CounterVec
}

var nrfStats *NssfStats

func initNssfStats() *NssfStats {
	return &NssfStats{
		nssfNssaiAvailability: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "nssf_nssai_availability",
			Help: "Counter of total NSSAI queries",
		}, []string{"query_type", "nf_id", "result"}),
		nssfNssaiAvailabilitySubscriptions: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "nssf_nssai_availability_subscription",
			Help: "Counter of total NSSAI subscription events",
		}, []string{"query_type", "request_nf_type", "nf_id", "result"}),
		nssfNsSelections: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "nssf_ns_selections",
			Help: "Counter of total NS selection queries",
		}, []string{"target_nf_type", "nf_id", "result"}),
	}
}

func (ps *NssfStats) register() error {
	if err := prometheus.Register(ps.nssfNssaiAvailability); err != nil {
		return err
	}
	if err := prometheus.Register(ps.nssfNssaiAvailabilitySubscriptions); err != nil {
		return err
	}
	if err := prometheus.Register(ps.nssfNsSelections); err != nil {
		return err
	}
	return nil
}

func init() {
	nrfStats = initNssfStats()

	if err := nrfStats.register(); err != nil {
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

// IncrementNrfRegistrationsStats increments number of total NRF registrations
func IncrementNssfNssaiAvailabilityStats(queryType, nfId, result string) {
	nrfStats.nssfNssaiAvailability.WithLabelValues(queryType, nfId, result).Inc()
}

// IncrementNrfSubscriptionsStats increments number of total NRF subscriptions
func IncrementNssfNssaiAvailabilitySubscriptionsStats(queryType, requestNfType, nfId, result string) {
	nrfStats.nssfNssaiAvailabilitySubscriptions.WithLabelValues(queryType, requestNfType, nfId, result).Inc()
}

// IncrementNrfNfInstancesStats increments number of total NRF queries
func IncrementNssfNsSelectionsStats(targetNfType, nfId, result string) {
	nrfStats.nssfNsSelections.WithLabelValues(targetNfType, nfId, result).Inc()
}
