package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	"strconv"
	"time"
)

var httpMetricsServer *http.Server

func main() {
	prometheusFlag := flag.Bool("prom", false, "Toggle Prometheus metrics export.")
	batchFlag := flag.Bool("batch", false, "Toggle batch sending with external service.")
	flag.Parse()
	SendBatchEnabled = *batchFlag

	var err error
	ReadEnvs()
	if err != nil {
		log.Fatal(err)
		return
	}

	ctx := context.Background()

	var repo Repository
	if err = repo.Init(ctx); err != nil {
		log.Fatal(err)
	}

	// almanax
	var almFeeds []AlmanaxFeed
	if almFeeds, err = repo.GetAlmanaxFeeds([]uint64{}); err != nil {
		log.Fatal(err)
	}
	go func(ctx context.Context) {
		for _, feed := range almFeeds {
			time.Sleep(time.Second * 1)
			go ListenAlmanax(ctx, feed)
		}
	}(ctx)

	// twitter
	var twitterFeeds []TwitterFeed
	twitterFeeds, err = repo.GetTwitterFeeds([]uint64{})
	if err != nil {
		log.Fatal(err)
	}
	for _, feed := range twitterFeeds {
		go ListenTwitter(ctx, feed)
	}

	// rss
	var rssFeeds []RssFeed
	rssFeeds, err = repo.GetRssFeeds([]uint64{})
	if err != nil {
		log.Fatal(err)
	}
	for _, feed := range rssFeeds {
		go ListenRss(ctx, feed)
	}

	repo.Deinit()

	httpDataServer := &http.Server{
		Addr:    fmt.Sprintf(":%s", ApiPort),
		Handler: Router(),
	}

	if *prometheusFlag {
		apiPort, _ := strconv.Atoi(ApiPort)
		httpMetricsServer = &http.Server{
			Addr:    fmt.Sprintf(":%d", apiPort+1),
			Handler: promhttp.Handler(),
		}

		go func() {
			log.Printf("metrics on port %d\n", apiPort+1)
			if err := httpMetricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatal(err)
			}
		}()
	}

	log.Printf("listen on port %s\n", ApiPort)
	if err := httpDataServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}

}
