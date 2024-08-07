// Copyright 2021 The Cockroach Authors.
//
// Licensed as a CockroachDB Enterprise file under the Cockroach Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//     https://github.com/cockroachdb/cockroach/blob/master/licenses/CCL.txt

package tenantcostclient

import (
	"github.com/cockroachdb/cockroach/pkg/kv/kvpb"
	"github.com/cockroachdb/cockroach/pkg/util/metric"
)

var (
	metaCurrentBlocked = metric.Metadata{
		Name:        "tenant.cost_client.blocked_requests",
		Help:        "Number of requests currently blocked by the rate limiter",
		Measurement: "Requests",
		Unit:        metric.Unit_COUNT,
	}

	// SQL usage related metrics.
	metaTotalRU = metric.Metadata{
		Name:        "tenant.sql_usage.request_units",
		Help:        "RU consumption",
		Measurement: "Request Units",
		Unit:        metric.Unit_COUNT,
	}
	metaTotalKVRU = metric.Metadata{
		Name:        "tenant.sql_usage.kv_request_units",
		Help:        "RU consumption attributable to KV",
		Measurement: "Request Units",
		Unit:        metric.Unit_COUNT,
	}
	metaTotalReadBatches = metric.Metadata{
		Name:        "tenant.sql_usage.read_batches",
		Help:        "Total number of KV read batches",
		Measurement: "Requests",
		Unit:        metric.Unit_COUNT,
	}
	metaTotalReadRequests = metric.Metadata{
		Name:        "tenant.sql_usage.read_requests",
		Help:        "Total number of KV read requests",
		Measurement: "Requests",
		Unit:        metric.Unit_COUNT,
	}
	metaTotalReadBytes = metric.Metadata{
		Name:        "tenant.sql_usage.read_bytes",
		Help:        "Total number of bytes read from KV",
		Measurement: "Bytes",
		Unit:        metric.Unit_COUNT,
	}
	metaTotalWriteBatches = metric.Metadata{
		Name:        "tenant.sql_usage.write_batches",
		Help:        "Total number of KV write batches",
		Measurement: "Requests",
		Unit:        metric.Unit_COUNT,
	}
	metaTotalWriteRequests = metric.Metadata{
		Name:        "tenant.sql_usage.write_requests",
		Help:        "Total number of KV write requests",
		Measurement: "Requests",
		Unit:        metric.Unit_COUNT,
	}
	metaTotalWriteBytes = metric.Metadata{
		Name:        "tenant.sql_usage.write_bytes",
		Help:        "Total number of bytes written to KV",
		Measurement: "Bytes",
		Unit:        metric.Unit_COUNT,
	}
	metaTotalSQLPodsCPUSeconds = metric.Metadata{
		Name:        "tenant.sql_usage.sql_pods_cpu_seconds",
		Help:        "Total amount of CPU used by SQL pods",
		Measurement: "CPU Seconds",
		Unit:        metric.Unit_SECONDS,
	}
	metaTotalPGWireEgressBytes = metric.Metadata{
		Name:        "tenant.sql_usage.pgwire_egress_bytes",
		Help:        "Total number of bytes transferred from a SQL pod to the client",
		Measurement: "Bytes",
		Unit:        metric.Unit_COUNT,
	}
	metaTotalExternalIOIngressBytes = metric.Metadata{
		Name:        "tenant.sql_usage.external_io_ingress_bytes",
		Help:        "Total number of bytes read from external services such as cloud storage providers",
		Measurement: "Bytes",
		Unit:        metric.Unit_COUNT,
	}
	metaTotalExternalIOEgressBytes = metric.Metadata{
		Name:        "tenant.sql_usage.external_io_egress_bytes",
		Help:        "Total number of bytes written to external services such as cloud storage providers",
		Measurement: "Bytes",
		Unit:        metric.Unit_COUNT,
	}
	metaTotalCrossRegionNetworkRU = metric.Metadata{
		Name:        "tenant.sql_usage.cross_region_network_ru",
		Help:        "Total number of RUs charged for cross-region network traffic",
		Measurement: "Request Units",
		Unit:        metric.Unit_COUNT,
	}
	metaTotalEstimatedKVCPUSeconds = metric.Metadata{
		Name:        "tenant.sql_usage.estimated_kv_cpu_seconds",
		Help:        "Estimated amount of CPU consumed by a virtual cluster, in the KV layer",
		Measurement: "CPU Seconds",
		Unit:        metric.Unit_SECONDS,
	}
	metaTotalEstimatedCPUSeconds = metric.Metadata{
		Name:        "tenant.sql_usage.estimated_cpu_seconds",
		Help:        "Estimated amount of CPU consumed by a virtual cluster",
		Measurement: "CPU Seconds",
		Unit:        metric.Unit_SECONDS,
	}
)

// metrics manage the metrics used by the tenant cost client.
type metrics struct {
	CurrentBlocked              *metric.Gauge
	TotalRU                     *metric.CounterFloat64
	TotalKVRU                   *metric.CounterFloat64
	TotalReadBatches            *metric.Counter
	TotalReadRequests           *metric.Counter
	TotalReadBytes              *metric.Counter
	TotalWriteBatches           *metric.Counter
	TotalWriteRequests          *metric.Counter
	TotalWriteBytes             *metric.Counter
	TotalSQLPodsCPUSeconds      *metric.CounterFloat64
	TotalPGWireEgressBytes      *metric.Counter
	TotalExternalIOEgressBytes  *metric.Counter
	TotalExternalIOIngressBytes *metric.Counter
	TotalCrossRegionNetworkRU   *metric.CounterFloat64
	TotalEstimatedKVCPUSeconds  *metric.CounterFloat64
	TotalEstimatedCPUSeconds    *metric.CounterFloat64
}

var _ metric.Struct = (*metrics)(nil)

// MetricStruct indicates that Metrics is a metric.Struct.
func (m *metrics) MetricStruct() {}

// Init initializes the tenant cost client metrics.
func (m *metrics) Init() {
	m.CurrentBlocked = metric.NewGauge(metaCurrentBlocked)
	m.TotalRU = metric.NewCounterFloat64(metaTotalRU)
	m.TotalKVRU = metric.NewCounterFloat64(metaTotalKVRU)
	m.TotalReadBatches = metric.NewCounter(metaTotalReadBatches)
	m.TotalReadRequests = metric.NewCounter(metaTotalReadRequests)
	m.TotalReadBytes = metric.NewCounter(metaTotalReadBytes)
	m.TotalWriteBatches = metric.NewCounter(metaTotalWriteBatches)
	m.TotalWriteRequests = metric.NewCounter(metaTotalWriteRequests)
	m.TotalWriteBytes = metric.NewCounter(metaTotalWriteBytes)
	m.TotalSQLPodsCPUSeconds = metric.NewCounterFloat64(metaTotalSQLPodsCPUSeconds)
	m.TotalPGWireEgressBytes = metric.NewCounter(metaTotalPGWireEgressBytes)
	m.TotalExternalIOEgressBytes = metric.NewCounter(metaTotalExternalIOEgressBytes)
	m.TotalExternalIOIngressBytes = metric.NewCounter(metaTotalExternalIOIngressBytes)
	m.TotalCrossRegionNetworkRU = metric.NewCounterFloat64(metaTotalCrossRegionNetworkRU)
	m.TotalEstimatedKVCPUSeconds = metric.NewCounterFloat64(metaTotalEstimatedKVCPUSeconds)
	m.TotalEstimatedCPUSeconds = metric.NewCounterFloat64(metaTotalEstimatedCPUSeconds)
}

func (m *metrics) getConsumption(consumption *kvpb.TenantConsumption) {
	consumption.RU = m.TotalRU.Count()
	consumption.KVRU = m.TotalKVRU.Count()
	consumption.ReadBatches = uint64(m.TotalReadBatches.Count())
	consumption.ReadRequests = uint64(m.TotalReadRequests.Count())
	consumption.ReadBytes = uint64(m.TotalReadBytes.Count())
	consumption.WriteBatches = uint64(m.TotalWriteBatches.Count())
	consumption.WriteRequests = uint64(m.TotalWriteRequests.Count())
	consumption.WriteBytes = uint64(m.TotalWriteBytes.Count())
	consumption.SQLPodsCPUSeconds = m.TotalSQLPodsCPUSeconds.Count()
	consumption.PGWireEgressBytes = uint64(m.TotalPGWireEgressBytes.Count())
	consumption.ExternalIOEgressBytes = uint64(m.TotalExternalIOEgressBytes.Count())
	consumption.ExternalIOIngressBytes = uint64(m.TotalExternalIOIngressBytes.Count())
	consumption.CrossRegionNetworkRU = m.TotalCrossRegionNetworkRU.Count()
	consumption.EstimatedCPUSeconds = m.TotalEstimatedCPUSeconds.Count()
}
