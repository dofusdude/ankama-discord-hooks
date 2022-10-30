package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

func handleGetMetaTwitterSubscriptions(w http.ResponseWriter, r *http.Request) {
	HandleGenGetMetaSubscriptions(w, r, GetTwitterFeeds)
}

// CRUD

func handleGetTwitter(w http.ResponseWriter, r *http.Request) {
	metricsIncSocialCRUD(TwitterWebhookType)
	handleGetSocial(r.Context().Value("id").(string), TwitterWebhookType, w, r)
}

func handleDeleteTwitter(w http.ResponseWriter, r *http.Request) {
	metricsIncSocialCRUD(TwitterWebhookType)
	handleDeleteSocial(TwitterWebhookType, w, r)
}

func handlePutTwitter(w http.ResponseWriter, r *http.Request) {
	metricsIncSocialCRUD(TwitterWebhookType)
	handlePutSocial(TwitterWebhookType, w, r)
}

func handleCreateTwitterHook(w http.ResponseWriter, r *http.Request) {
	metricsIncSocialCRUD(TwitterWebhookType)
	handleCreateSocial(TwitterWebhookType, w, r)
}

// utils for filter and fire hooks

func getLatestTweets(userId uint64, lastCheck time.Time, baseUrl string) ([]Tweet, error) {
	var err error
	if userId == 0 {
		return []Tweet{}, fmt.Errorf("invalid user id")
	}

	var bearer = "Bearer " + TwitterToken

	url := fmt.Sprintf("%s/2/users/%d/tweets", baseUrl, userId)

	// Create a new request using http
	var req *http.Request
	if req, err = http.NewRequest("GET", url, nil); err != nil {
		return nil, err
	}

	// add authorization header to the req
	req.Header.Add("Authorization", bearer)
	params := req.URL.Query()
	params.Add("max_results", "5")
	params.Add("start_time", lastCheck.Format(time.RFC3339))
	params.Add("tweet.fields", "created_at,in_reply_to_user_id")
	params.Add("expansions", "author_id,attachments.media_keys")
	params.Add("user.fields", "name,username,profile_image_url")
	params.Add("media.fields", "url")
	req.URL.RawQuery = params.Encode()

	// Send req using http Client
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error on response.\n[ERROR] -", err)
		return []Tweet{}, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Println(err)
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error while reading the response bytes:", err)
		return []Tweet{}, err
	}

	var jsonData ApiUserTweetResult
	err = json.Unmarshal(body, &jsonData)
	if err != nil {
		return []Tweet{}, err
	}

	if jsonData.Meta.ResultCount < 1 || len(jsonData.Includes.Users) < 1 {
		return []Tweet{}, nil
	}

	var tweets []Tweet
	for _, apiTweet := range jsonData.Data {
		if apiTweet.InReplyTo != nil {
			continue
		}

		var tweet Tweet
		tweet.Text = apiTweet.Text
		tweet.CreatedAt = apiTweet.CreatedAt
		tweet.Author = jsonData.Includes.Users[0]
		tweet.IsNew = true

		if len(apiTweet.Attachments.MediaKeys) > 0 {
			for _, mediaKey := range apiTweet.Attachments.MediaKeys {
				for _, media := range jsonData.Includes.Media {
					if media.MediaKey == mediaKey && media.Type == "photo" {
						tweet.Attachments = append(tweet.Attachments, media.Url)
					}
				}
			}
		}

		tweets = append(tweets, tweet)
	}

	return tweets, nil
}

// fire hook handlers

func HandleTimeTwitter(socialFeed IFeed, _ TwitterState, tickTime time.Time, tickRate time.Duration, repo Repository) ([]TwitterSend, error) {
	var twitterSends []TwitterSend
	tweets, err := getLatestTweets(socialFeed.GetTwitterId(), tickTime.Add(tickRate*-1), "https://api.twitter.com")
	if err != nil {
		log.Println(err)
		return nil, err
	}

	for _, tweet := range tweets {
		if !tweet.IsNew {
			return nil, nil
		}

		var subbedWebhooks []HasIdBlackWhiteList[string]
		if subbedWebhooks, err = repo.GetTwitterSubsForFeed(socialFeed); err != nil {
			log.Printf("error getting subbed webhooks for feed %d: %v", socialFeed.GetId(), err)
			return nil, err
		}
		if len(subbedWebhooks) == 0 {
			return nil, nil
		}

		webhooksToSend := filterByBlackWhitelist(subbedWebhooks, tweet.Text)
		sendHooksTotal.Add(float64(len(webhooksToSend)))
		sendHooksTwitter.Add(float64(len(webhooksToSend)))

		twitterSends = append(twitterSends, TwitterSend{
			Tweet:    tweet,
			Webhooks: webhooksToSend,
		})
	}

	return twitterSends, nil
}

func BuildDiscordHookTwitter(twitterHook TwitterSend) ([]PreparedHook, error) {
	var res []PreparedHook
	for _, webhook := range twitterHook.Webhooks {
		tweetText := TruncateText(twitterHook.Tweet.Text, webhook.GetPreviewLength())
		discordWebhook := DiscordWebhook{
			Content:   &tweetText,
			Username:  "@" + twitterHook.Tweet.Author.Username,
			AvatarUrl: twitterHook.Tweet.Author.ProfileImageURL,
		}

		if twitterHook.Tweet.Attachments != nil && len(twitterHook.Tweet.Attachments) > 0 {
			discordWebhook.Embeds = []DiscordEmbed{
				{
					Image: &DiscordImage{
						Url: twitterHook.Tweet.Attachments[0],
					},
					Color: 3684408,
				},
			}
		}

		jsonBody, err := json.Marshal(discordWebhook)
		if err != nil {
			return nil, err
		}

		res = append(res, PreparedHook{
			Callback: webhook.GetCallback(),
			Body:     string(jsonBody),
		})
	}

	return res, nil
}

func ListenTwitter(ctx context.Context, feed IFeed) {
	var state TwitterState
	Listen(ctx, TwitterPollingRate, feed, state, HandleTimeTwitter, BuildDiscordHookTwitter)
}
