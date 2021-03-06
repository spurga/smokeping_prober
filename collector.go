// Copyright 2018 Ben Kochie <superq@gmail.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"github.com/superq/smokeping_prober/ping"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
)

const (
	namespace = "smokeping"
)

var (
	labelNames = []string{"ip", "host"}

	pingResponseSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "response_duration_seconds",
			Help:      "A histogram of latencies for ping responses.",
			Buckets:   prometheus.ExponentialBuckets(0.00005, 2, 20),
		},
		labelNames,
	)
)

func init() {
	prometheus.MustRegister(pingResponseSeconds)
}

// SmokepingCollector collects metrics from the pinger.
type SmokepingCollector struct {
	pingers *[]*ping.Pinger

	requestsSent *prometheus.Desc
}

func NewSmokepingCollector(pingers *[]*ping.Pinger) *SmokepingCollector {
	for _, pinger := range *pingers {
		pinger.OnRecv = func(pkt *ping.Packet) {
			pingResponseSeconds.WithLabelValues(pkt.IPAddr.String(), pkt.Addr).Observe(pkt.Rtt.Seconds())
			log.Debugf("%d bytes from %s: icmp_seq=%d time=%v\n",
				pkt.Nbytes, pkt.IPAddr, pkt.Seq, pkt.Rtt)
		}
		pinger.OnFinish = func(stats *ping.Statistics) {
			log.Debugf("\n--- %s ping statistics ---\n", stats.Addr)
			log.Debugf("%d packets transmitted, %d packets received, %v%% packet loss\n",
				stats.PacketsSent, stats.PacketsRecv, stats.PacketLoss)
			log.Debugf("round-trip min/avg/max/stddev = %v/%v/%v/%v\n",
				stats.MinRtt, stats.AvgRtt, stats.MaxRtt, stats.StdDevRtt)
		}
	}

	return &SmokepingCollector{
		pingers: pingers,
		requestsSent: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "requests_total"),
			"Number of ping requests sent",
			labelNames,
			nil,
		),
	}
}

func (s *SmokepingCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- s.requestsSent
}

func (s *SmokepingCollector) Collect(ch chan<- prometheus.Metric) {
	for _, pinger := range *s.pingers {
		stats := pinger.Statistics()

		ch <- prometheus.MustNewConstMetric(
			s.requestsSent,
			prometheus.CounterValue,
			float64(stats.PacketsSent),
			stats.IPAddr.String(),
			stats.Addr,
		)
	}
}
