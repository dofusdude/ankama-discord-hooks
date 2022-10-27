package main

import (
	"context"
	"encoding/json"
	"github.com/mitchellh/hashstructure/v2"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"

	"github.com/mmcdole/gofeed"
)

func handleGetMetaRssSubscriptions(w http.ResponseWriter, r *http.Request) {
	HandleGenGetMetaSubscriptions(w, r, GetRssFeeds)
}

// CRUD

func handleGetRss(w http.ResponseWriter, r *http.Request) {
	metricsIncSocialCRUD(RSSWebhookType)
	handleGetSocial(r.Context().Value("id").(string), RSSWebhookType, w, r)
}

func handleDeleteRss(w http.ResponseWriter, r *http.Request) {
	metricsIncSocialCRUD(RSSWebhookType)
	handleDeleteSocial(RSSWebhookType, w, r)
}

func handlePutRss(w http.ResponseWriter, r *http.Request) {
	metricsIncSocialCRUD(RSSWebhookType)
	handlePutSocial(RSSWebhookType, w, r)
}

func handleCreateRssHook(w http.ResponseWriter, r *http.Request) {
	metricsIncSocialCRUD(RSSWebhookType)
	handleCreateSocial(RSSWebhookType, w, r)
}

// utils for filter and fire hooks

func findImageUrl(html string) string {
	re := regexp.MustCompile(`(?i)<img[^>]+src="([^">]+)"`)
	matches := re.FindStringSubmatch(html)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func filterMarkdownImageStrings(markdown string) string {
	re := regexp.MustCompile(`(?i)!\[.*]\(.*\)`)

	return re.ReplaceAllString(markdown, "")
}

func shortenAndRenderDescription(description string, maxLength int) (string, error) {
	converter := md.NewConverter("", true, nil)
	markdown, err := converter.ConvertString(description)
	if err != nil {
		return "", err
	}

	markdown = filterMarkdownImageStrings(markdown)
	markdown = TruncateText(markdown, maxLength)

	return markdown, nil
}

// fire hook handlers

func HandleTimeRss(socialFeed IFeed, state *RssState, _ time.Time, _ time.Duration, repo Repository) ([]RssSend, error) {
	fp := gofeed.NewParser()
	var rssSends []RssSend
	rssFeed, err := fp.ParseURL(socialFeed.GetRSSUrl())
	if err != nil {
		return nil, err
	}

	var newItems []gofeed.Item
	if len(rssFeed.Items) == 0 {
		return nil, nil
	}

	// build initial hash
	if state.LastHash == 0 {
		var itemHash uint64
		if itemHash, err = hashstructure.Hash(rssFeed.Items[0], hashstructure.FormatV2, nil); err != nil {
			log.Println("Error hashing item", err)
			return nil, err
		}
		state.LastHash = itemHash
		return nil, nil
	}

	var firstHash uint64 = 0
	for _, item := range rssFeed.Items {
		var itemHash uint64
		if itemHash, err = hashstructure.Hash(item, hashstructure.FormatV2, nil); err != nil {
			log.Println("Error hashing item", err)
			continue
		}

		if firstHash == 0 {
			firstHash = itemHash
		}

		if itemHash == state.LastHash {
			break
		}

		newItems = append(newItems, *item)
	}
	state.LastHash = firstHash

	var subbedWebhooks []HasIdBlackWhiteList[string]
	if subbedWebhooks, err = repo.GetRSSSubsForFeed(socialFeed); err != nil {
		log.Printf("error getting subbed webhooks for feed %d: %v", socialFeed.GetId(), err)
		return nil, err
	}

	if len(subbedWebhooks) == 0 {
		return nil, nil
	}

	for _, item := range newItems {
		webhooksToSend := filterByBlackWhitelist(subbedWebhooks, item.Description)
		sendHooksTotal.Add(float64(len(webhooksToSend)))
		sendHooksRss.Add(float64(len(webhooksToSend)))

		rssSends = append(rssSends, RssSend{
			Item:     item,
			Webhooks: webhooksToSend,
			Feed:     socialFeed,
		})
	}

	return rssSends, nil
}

func generateUsernameRss(feed IFeed) string {
	name := feed.GetFeedName()
	nameParts := strings.Split(name, "-")
	if len(nameParts) < 2 {
		return "Ankama"
	}
	caser := cases.Title(language.English)
	game := caser.String(nameParts[0])
	newsType := caser.String(nameParts[len(nameParts)-1])
	return game + " " + newsType
}

func BuildDiscordHookRss(rssHookBuild RssSend) ([]PreparedHook, error) {
	var discordWebhook DiscordWebhook
	var res []PreparedHook

	optImage := findImageUrl(rssHookBuild.Item.Description)
	for _, webhook := range rssHookBuild.Webhooks {
		shortenedText, err := shortenAndRenderDescription(rssHookBuild.Item.Description, webhook.GetPreviewLength())
		if err != nil {
			return nil, err
		}

		discordWebhook.AvatarUrl = "https://discord.dofusdude.com/ankama_rss_logo.jpg"
		discordWebhook.Username = generateUsernameRss(rssHookBuild.Feed)
		discordWebhook.Embeds = []DiscordEmbed{
			{
				Title: &rssHookBuild.Item.Title,
				Color: 16777215,
				Url:   &rssHookBuild.Item.Link,
			},
		}

		if optImage != "" {
			discordWebhook.Embeds[0].Image = &DiscordImage{
				Url: optImage,
			}
		}

		if len(shortenedText) != 0 {
			discordWebhook.Embeds[0].Description = &shortenedText
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

func ListenRss(ctx context.Context, feed IFeed) {
	var state RssState
	state.LastHash = 0

	Listen(ctx, RssPollingRate, feed, &state, HandleTimeRss, BuildDiscordHookRss)
}
