package main

import "github.com/Shopify/sarama"

func newProducer() (sarama.SyncProducer, error) {
	clients := []string{
		"redpanda-0.redpanda.redpanda.svc.cluster.local.:9093",
		"redpanda-1.redpanda.redpanda.svc.cluster.local.:9093",
		"redpanda-2.redpanda.redpanda.svc.cluster.local.:9093",
	}

	return sarama.NewSyncProducer(clients, nil)
}
