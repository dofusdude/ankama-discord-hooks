package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

func isDiscordWebhook(url string) bool {
	prefixCheck := strings.HasPrefix(url, "https://discord.com/api/webhooks/")
	if !prefixCheck {
		return false
	}

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false
	}

	getDiscordHook, err := http.DefaultClient.Do(request)
	if err != nil {
		return false
	}

	return getDiscordHook.StatusCode == http.StatusOK
}

type PreparedHook struct {
	Callback string
	Body     string
}

func tick[CustomObj any, Feed IFeed, State any](ctx context.Context, tickTime time.Time, state State,
	feed Feed,
	tickRate time.Duration,
	handleTime func(socialFeed Feed, state State, tickTime time.Time, tickRate time.Duration, repo Repository) ([]CustomObj, error),
	buildDiscordWebhook func(customObj CustomObj) ([]PreparedHook, error)) error {
	var err error
	var repo Repository
	if err = repo.Init(ctx); err != nil {
		return err
	}
	defer repo.Deinit()

	topicSends, err := handleTime(feed, state, tickTime, tickRate, repo)
	if err != nil {
		return err
	}

	if topicSends == nil {
		return nil
	}

	var failedUrls []chan string
	for _, webhook := range topicSends {
		failedUrls = append(failedUrls, make(chan string))
		go func(topicHooks CustomObj, res chan string) {
			var preparedHooks []PreparedHook
			preparedHooks, err = buildDiscordWebhook(topicHooks)
			if err != nil {
				log.Println("Error while buildDiscordWebhook in feed ", feed.GetFeedName(), err)
				res <- "ok" // TODO handle more verbose
				return
			}

			var callbackReturns []chan string
			for _, preparedHook := range preparedHooks {
				channel := make(chan string)
				callbackReturns = append(callbackReturns, channel)
				go func(preparedHook PreparedHook, channel chan string) {
					var resp *http.Response
					resp, err = http.Post(preparedHook.Callback, "application/json", bytes.NewBuffer([]byte(preparedHook.Body)))
					if err != nil {
						log.Println("error posting callback ", err)
						channel <- preparedHook.Callback
						return
					}

					defer func(Body io.ReadCloser) {
						if err = Body.Close(); err != nil {
							log.Println("could not close body io ", err)
						}
					}(resp.Body)

					if resp.StatusCode == http.StatusNotFound {
						log.Println("returned not found for ", preparedHook.Callback)
						channel <- preparedHook.Callback
						return
					}

					if resp.StatusCode != http.StatusNoContent {
						log.Println("strange return from discord hook ", resp.StatusCode)
						channel <- "ok" + preparedHook.Callback
						return
					}
				}(preparedHook, callbackReturns[len(callbackReturns)-1])
			}
		}(webhook, failedUrls[len(failedUrls)-1])
	}

	for _, failedCallback := range failedUrls {
		callback := <-failedCallback
		if strings.HasPrefix(callback, "ok") {
			if callback != "ok" {
				if err = repo.FireStampWebhook(callback[2:]); err != nil {
					log.Println("could not stamp webhook ", callback[2:], err)
				}
			}
		} else {
			if err = repo.DeleteHooksByCallback(callback); err != nil {
				log.Println("error deleting webhook ", err)
			}
		}
	}

	return nil
}

func Listen[CustomObj any, Feed IFeed, State any](ctx context.Context, tickRate time.Duration,
	feed Feed,
	state State,
	handleTime func(socialFeed Feed, state State, tickTime time.Time, tickRate time.Duration, repo Repository) ([]CustomObj, error),
	buildDiscordWebhook func(customObj CustomObj) ([]PreparedHook, error),
) {
	ticker := time.NewTicker(tickRate)

	for {
		select {
		case tickTime := <-ticker.C:
			if err := tick(ctx, tickTime, state, feed, tickRate, handleTime, buildDiscordWebhook); err != nil {
				log.Println("Error in tick ", err)
				continue
			}
		case <-ctx.Done():
			ticker.Stop()
			return
		}
	}
}

func SendAndCleanFailed(pack WebhookJobs, deleteByCallbackFunc func(callback string) error) error {
	notWorked, err := send(pack)
	if err != nil {
		return err
	}
	for _, job := range notWorked {
		if err := deleteByCallbackFunc(job); err != nil {
			log.Println(err)
			continue
		}
	}

	return nil
}

func send(pack WebhookJobs) ([]string, error) {
	marshal, err := json.Marshal(pack)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	var resp *http.Response
	resp, err = http.Post(ServerlessSenderUrl, "application/json", bytes.NewBuffer(marshal))
	if err != nil {
		log.Println(err)
		return nil, err
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Println(err)
		}
	}(resp.Body)

	if resp.StatusCode != 200 {
		log.Println("SocialWebhook worker returned non 200 status code:", resp.StatusCode)
		return nil, err
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	if len(string(bodyBytes)) == 0 {
		log.Println("SocialWebhook worker returned empty body")
		return nil, err
	}

	var notWorkedUrls []string
	err = json.Unmarshal(bodyBytes, &notWorkedUrls)
	if err != nil {
		return nil, err
	}

	return notWorkedUrls, nil
}
