package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	requestsCRUDTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "webhooks_requestsCRUDTotal",
		Help: "The total number of CRUD requests for all types.",
	})

	requestsCRUDTwitter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "webhooks_requestsCRUDTwitter",
		Help: "The number of CRUD requests for twitter hooks.",
	})

	requestsCRUDRss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "webhooks_requestsCRUDRss",
		Help: "The number of CRUD requests for rss hooks.",
	})

	requestsCRUDAlmanax = promauto.NewCounter(prometheus.CounterOpts{
		Name: "webhooks_requestsCRUDAlmanax",
		Help: "The number of CRUD requests for almanax hooks.",
	})

	sendHooksTwitter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "webhooks_sendTwitter",
		Help: "The number of sent twitter hooks.",
	})

	sendHooksRss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "webhooks_sendRss",
		Help: "The number of sent rss hooks.",
	})

	sendHooksAlmanax = promauto.NewCounter(prometheus.CounterOpts{
		Name: "webhooks_sendAlmanax",
		Help: "The number of sent almanax hooks.",
	})

	sendHooksTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "webhooks_sendTotal",
		Help: "The total number of hooks sent.",
	})
)
