package main

import (
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"net/http"
	"strings"
)

// filterByBlackWhitelist applies the blacklist and whitelist to the given slice of webhooks.
// If only a blacklist is provided, it will filter a tweet out if a substring from the blacklist is found in the tweet
// If only a whitelist is provided, it will filter a tweet out if a substring from the whitelist is not found in the tweet
// If both are provided, it will filter out a tweet if only the blacklist wants to filter it out. The whitelist can override
// the blacklist with other substrings.
func filterByBlackWhitelist(webhooks []HasIdBlackWhiteList[string], text string) []IHook {
	lowerText := strings.ToLower(text)
	var webhooksToSend []IHook
	for _, subbedHook := range webhooks {
		isWhitelisted := false
		whitelistExists := len(subbedHook.GetWhitelist()) > 0

		if whitelistExists {
			for _, word := range subbedHook.GetWhitelist() {
				if strings.Contains(lowerText, word) {
					isWhitelisted = true
				}
			}
		}

		isBlacklisted := false
		blacklistExists := len(subbedHook.GetBlacklist()) > 0

		if blacklistExists {
			for _, word := range subbedHook.GetBlacklist() {
				if strings.Contains(lowerText, word) {
					isBlacklisted = true
				}
			}
		}

		if !blacklistExists && !whitelistExists {
			webhooksToSend = append(webhooksToSend, subbedHook)
			continue
		}

		if blacklistExists {
			if whitelistExists && !isBlacklisted && !isWhitelisted {
				webhooksToSend = append(webhooksToSend, subbedHook)
				continue
			}
			if isBlacklisted && whitelistExists && isWhitelisted {
				webhooksToSend = append(webhooksToSend, subbedHook)
				continue
			} else if !isBlacklisted {
				if whitelistExists {
					if isWhitelisted {
						webhooksToSend = append(webhooksToSend, subbedHook)
					}
					continue
				} else {
					webhooksToSend = append(webhooksToSend, subbedHook)
					continue
				}
			}
		} else { // !blacklistExists
			if whitelistExists && isWhitelisted {
				webhooksToSend = append(webhooksToSend, subbedHook)
			}
			continue
		}
	}

	return webhooksToSend
}

func getSocial(socialWebhookType string, parsedId uuid.UUID, repo Repository) (SocialWebhookDTO, error) {
	var err error
	var found bool
	if found, err = repo.HasSocialWebhook(socialWebhookType, parsedId); err != nil {
		return SocialWebhookDTO{}, err
	}

	if !found {
		return SocialWebhookDTO{}, errors.New("not found")
	}

	var foundWebhook ISocialHook
	if foundWebhook, err = repo.GetSocialHook(socialWebhookType, parsedId); err != nil {
		return SocialWebhookDTO{}, err
	}

	hookOut := SocialWebhookDTO{
		Id:            foundWebhook.GetId(),
		Blacklist:     foundWebhook.GetBlacklist(),
		Whitelist:     foundWebhook.GetWhitelist(),
		PreviewLength: foundWebhook.GetPreviewLength(),
		Format:        foundWebhook.GetFormat(),
		CreatedAt:     foundWebhook.GetCreatedAt(),
		LastFiredAt:   foundWebhook.GetLastFiredAt(),
		UpdatedAt:     foundWebhook.GetUpdatedAt(),
	}

	var subbedFeeds []IFeed
	if subbedFeeds, err = repo.GetSocialHookSubscriptions(socialWebhookType, parsedId); err != nil {
		return SocialWebhookDTO{}, err
	}

	for _, feed := range subbedFeeds {
		hookOut.Subscriptions = append(hookOut.Subscriptions, feed.GetFeedName())
	}

	return hookOut, nil
}

func handleGetSocial(id string, socialWebhookType string, w http.ResponseWriter, r *http.Request) {
	parsedId, err := uuid.Parse(id)
	if err != nil {
		http.Error(w, "Invalid id.", http.StatusBadRequest)
		return
	}

	var repo Repository
	if err = repo.Init(r.Context()); err != nil {
		http.Error(w, "Internal error.", http.StatusInternalServerError)
		return
	}
	defer repo.Deinit()

	var hookOut SocialWebhookDTO
	hookOut, err = getSocial(socialWebhookType, parsedId, repo)
	if err != nil {
		if err.Error() == "not found" {
			http.Error(w, "Not found.", http.StatusNotFound)
			return
		} else {
			http.Error(w, "Internal error.", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err = json.NewEncoder(w).Encode(hookOut); err != nil {
		http.Error(w, "Error encoding the response.", http.StatusInternalServerError)
		return
	}
}

func metricsIncSocialCRUD(socialWebhookType string) {
	requestsCRUDTotal.Inc()
	switch socialWebhookType {
	case TwitterWebhookType:
		requestsCRUDTwitter.Inc()
	case RSSWebhookType:
		requestsCRUDRss.Inc()
	}
}

func handleDeleteSocial(socialWebhookType string, w http.ResponseWriter, r *http.Request) {
	id := r.Context().Value("id").(string)

	parsedId, err := uuid.Parse(id)
	if err != nil {
		http.Error(w, "Invalid id.", http.StatusBadRequest)
		return
	}

	var repo Repository
	if err = repo.Init(r.Context()); err != nil {
		http.Error(w, "Internal error.", http.StatusInternalServerError)
		return
	}
	defer repo.Deinit()

	var found bool
	if found, err = repo.HasSocialWebhook(socialWebhookType, parsedId); err != nil {
		http.Error(w, "Internal error.", http.StatusInternalServerError)
		return
	}

	if !found {
		http.Error(w, "Not found.", http.StatusNotFound)
		return
	}

	if err = repo.DeleteHook(parsedId); err != nil {
		http.Error(w, "Could not delete webhook.", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func handlePutSocial(socialWebhookType string, w http.ResponseWriter, r *http.Request) {
	id := r.Context().Value("id").(string)

	var err error
	var updateSocialWebhook SocialWebhookPut
	if err = json.NewDecoder(r.Body).Decode(&updateSocialWebhook); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	parsedId, err := uuid.Parse(id)
	if err != nil {
		http.Error(w, "Invalid id.", http.StatusBadRequest)
		return
	}

	var repo Repository
	if err = repo.Init(r.Context()); err != nil {
		http.Error(w, "Could not connect to internal services.", http.StatusInternalServerError)
		return
	}
	defer repo.Deinit()

	var found bool
	if found, err = repo.HasSocialWebhook(socialWebhookType, parsedId); err != nil {
		http.Error(w, "Internal error.", http.StatusInternalServerError)
		return
	}

	if !found {
		http.Error(w, "Not found.", http.StatusNotFound)
		return
	}

	updateHook := SocialWebhookPutDb{
		Id:            parsedId,
		Blacklist:     updateSocialWebhook.Blacklist,
		Whitelist:     updateSocialWebhook.Whitelist,
		PreviewLength: updateSocialWebhook.PreviewLength,
		Subscriptions: updateSocialWebhook.Subscriptions,
	}

	if err = repo.UpdateSocialHook(socialWebhookType, updateHook); err != nil {
		if err.Error() == "feed not found" {
			http.Error(w, "Invalid subscription.", http.StatusBadRequest)
			return
		} else {
			http.Error(w, "Could not update webhook.", http.StatusInternalServerError)
			return
		}
	}

	handleGetSocial(id, socialWebhookType, w, r)
}

func handleCreateSocial(socialWebhookType string, w http.ResponseWriter, r *http.Request) {
	var err error
	var newSocialWebhook SocialHookCreate
	if err = json.NewDecoder(r.Body).Decode(&newSocialWebhook); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if newSocialWebhook.Callback == "" {
		http.Error(w, "Callback is required.", http.StatusBadRequest)
		return
	}

	if newSocialWebhook.Subscriptions == nil {
		http.Error(w, "Subscriptions are required.", http.StatusBadRequest)
		return
	}

	if newSocialWebhook.Format != "discord" {
		http.Error(w, "Callback must have a known format.", http.StatusBadRequest)
		return
	}

	if !isDiscordWebhook(newSocialWebhook.Callback) {
		http.Error(w, "Callback is not a valid Discord URL.", http.StatusBadRequest)
		return
	}

	var repo Repository
	if err = repo.Init(r.Context()); err != nil {
		http.Error(w, "Could not connect to internal services.", http.StatusInternalServerError)
		return
	}
	defer repo.Deinit()

	var hasCallback bool
	switch socialWebhookType {
	case TwitterWebhookType:
		if newSocialWebhook.PreviewLength == nil {
			defaultTwitterPreviewLength := 280
			newSocialWebhook.PreviewLength = &defaultTwitterPreviewLength
		}
		hasCallback, err = repo.HasTwitterWebhookCallback(newSocialWebhook.Callback)
	case RSSWebhookType:
		if newSocialWebhook.PreviewLength == nil {
			defaultRssPreviewLength := 2000
			newSocialWebhook.PreviewLength = &defaultRssPreviewLength
		}
		hasCallback, err = repo.HasRssWebhookCallback(newSocialWebhook.Callback)
	default:
		http.Error(w, "Invalid webhook type.", http.StatusBadRequest)
	}

	if err != nil {
		http.Error(w, "Internal error.", http.StatusInternalServerError)
		return
	}

	if hasCallback {
		http.Error(w, "Callback already exists.", http.StatusConflict)
		return
	}

	var id uuid.UUID
	id, err = repo.CreateSocialHook(socialWebhookType, newSocialWebhook)
	if err != nil {
		if err.Error() == "some feeds not found" {
			http.Error(w, "Some feeds not found.", http.StatusBadRequest)
			return
		} else {
			http.Error(w, "Internal error.", http.StatusInternalServerError)
			return
		}
	}

	var hookOut SocialWebhookDTO
	hookOut, err = getSocial(socialWebhookType, id, repo)
	if err != nil {
		if err.Error() == "not found" {
			http.Error(w, "Not found.", http.StatusNotFound)
			return
		} else {
			http.Error(w, "Internal error.", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err = json.NewEncoder(w).Encode(hookOut); err != nil {
		http.Error(w, "Error encoding the response.", http.StatusInternalServerError)
		return
	}
}

func HandleGenGetMetaSubscriptions[T IFeed](w http.ResponseWriter, r *http.Request, getFeeds func([]uint64, Repository) ([]T, error)) {
	var err error
	var repo Repository
	if err = repo.Init(r.Context()); err != nil {
		http.Error(w, "Could not connect to internal services.", http.StatusInternalServerError)
		return
	}
	defer repo.Deinit()

	var feeds []T
	if feeds, err = getFeeds([]uint64{}, repo); err != nil {
		http.Error(w, "Could not connect to internal services.", http.StatusInternalServerError)
		return
	}

	if len(feeds) == 0 {
		http.Error(w, "No feeds found.", http.StatusNotFound)
		return
	}

	subscriptions := Map(feeds, func(feed T) string {
		return feed.GetFeedName()
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err = json.NewEncoder(w).Encode(HookMeta{
		Subscriptions: subscriptions,
	}); err != nil {
		http.Error(w, "Error encoding the response.", http.StatusInternalServerError)
		return
	}
}
