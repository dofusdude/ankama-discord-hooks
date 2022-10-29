package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dofusdude/dodugo"
	"github.com/google/uuid"
)

func handleGetMetaAlmanaxSubscriptions(w http.ResponseWriter, r *http.Request) {
	HandleGenGetMetaSubscriptions(w, r, GetAlmanaxFeeds)
}

// CRUD

func toDTO(webhook AlmanaxWebhook) AlmanaxHookDTO {
	prep := AlmanaxHookDTO{
		Id: webhook.Id,
		DailySettings: DailySettings{
			Timezone:       *webhook.DailySettings.Timezone,
			MidnightOffset: *webhook.DailySettings.MidnightOffset,
		},
		Subscriptions: webhook.Subscriptions,
		WantsIsoDate:  webhook.WantsIsoDate,
		Format:        webhook.Format,
		CreatedAt:     webhook.CreatedAt,
		UpdatedAt:     webhook.UpdatedAt,
		LastFiredAt:   webhook.LastFiredAt,
	}

	if webhook.BonusWhitelist != nil && len(webhook.BonusWhitelist) > 0 {
		prep.BonusWhitelist = webhook.BonusWhitelist
	}

	if webhook.BonusBlacklist != nil && len(webhook.BonusBlacklist) > 0 {
		prep.BonusBlacklist = webhook.BonusBlacklist
	}

	if webhook.Mentions != nil && len(*webhook.Mentions) > 0 {
		prep.Mentions = webhook.Mentions
	}

	return prep
}

func handleDeleteAlmanaxHook(w http.ResponseWriter, r *http.Request) {
	requestsCRUDTotal.Inc()
	requestsCRUDAlmanax.Inc()
	id := r.Context().Value("id").(string)
	var err error = nil

	var parsedId uuid.UUID
	parsedId, err = uuid.Parse(id)
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

	var hasWebhook bool
	if hasWebhook, err = repo.HasAlmanaxWebhook(parsedId); err != nil {
		http.Error(w, "Internal error.", http.StatusInternalServerError)
		return
	}

	if !hasWebhook {
		http.Error(w, "Not found.", http.StatusNotFound)
		return
	}

	if err = repo.DeleteHook(parsedId); err != nil {
		http.Error(w, "Internal error.", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func getPossibleAlmanaxBonuses(ctx context.Context) (*Set[string], error) {
	almClient := dodugo.NewAPIClient(dodugo.NewConfiguration())
	almBonuses, _, err := almClient.MetaApi.GetMetaAlmanaxBonuses(ctx, "en").Execute()
	if err != nil {
		return nil, err
	}

	possibleBonuses := NewSet[string]()
	for _, bonus := range almBonuses {
		possibleBonuses.Add(bonus.GetId())
	}

	return possibleBonuses, nil
}

func handleCreateAlmanax(w http.ResponseWriter, r *http.Request) {
	requestsCRUDTotal.Inc()
	requestsCRUDAlmanax.Inc()
	var err error = nil
	var createWebhook AlmanaxHookPost
	if err = json.NewDecoder(r.Body).Decode(&createWebhook); err != nil {
		http.Error(w, "Invalid request.", http.StatusBadRequest)
		return
	}

	if createWebhook.Callback == "" {
		http.Error(w, "Callback is required.", http.StatusBadRequest)
		return
	}

	if createWebhook.Subscriptions == nil {
		http.Error(w, "Subscriptions are required.", http.StatusBadRequest)
		return
	}

	defaultTz := "Europe/Paris"
	defaultTzOffset := 0
	if createWebhook.DailySettings == nil {
		createWebhook.DailySettings = &WebhookDailySettings{
			Timezone:       &defaultTz,
			MidnightOffset: &defaultTzOffset,
		}
	}

	if createWebhook.DailySettings.Timezone == nil {
		createWebhook.DailySettings.Timezone = &defaultTz
	}

	if createWebhook.DailySettings.MidnightOffset == nil {
		createWebhook.DailySettings.MidnightOffset = &defaultTzOffset
	}

	if createWebhook.Format != "discord" {
		http.Error(w, "Callback must have a known format.", http.StatusBadRequest)
		return
	}

	if !isDiscordWebhook(createWebhook.Callback) {
		http.Error(w, "Callback is not a valid Discord URL.", http.StatusBadRequest)
		return
	}

	if createWebhook.WantsIsoDate == nil {
		defaultIsoDate := false
		createWebhook.WantsIsoDate = &defaultIsoDate
	}

	if createWebhook.BonusBlacklist != nil && createWebhook.BonusWhitelist != nil {
		http.Error(w, "You can't have both a bonus whitelist and a bonus blacklist.", http.StatusBadRequest)
		return
	}

	_, err = time.LoadLocation(*createWebhook.DailySettings.Timezone)
	if err != nil {
		http.Error(w, "Timezone not valid.", http.StatusBadRequest)
		return
	}

	if *createWebhook.DailySettings.MidnightOffset < 0 || *createWebhook.DailySettings.MidnightOffset > 23 {
		http.Error(w, "Offset should be between 0 and 23 valid.", http.StatusBadRequest)
		return
	}

	possibleBonuses, err := getPossibleAlmanaxBonuses(r.Context())
	if err != nil {
		http.Error(w, "Could not reach Almanax API.", http.StatusBadGateway)
		return
	}

	for _, whitelistEntry := range createWebhook.BonusWhitelist {
		if !possibleBonuses.Has(whitelistEntry) {
			http.Error(w, "Unknown almanax bonus id: "+whitelistEntry+".", http.StatusBadRequest)
			return
		}
	}

	for _, blacklistEntry := range createWebhook.BonusBlacklist {
		if !possibleBonuses.Has(blacklistEntry) {
			http.Error(w, "Unknown almanax bonus id: "+blacklistEntry+".", http.StatusBadRequest)
			return
		}
	}

	var repo Repository
	if err = repo.Init(r.Context()); err != nil {
		http.Error(w, "Internal error.", http.StatusInternalServerError)
		return
	}
	defer repo.Deinit()

	var hasAlm bool
	if hasAlm, err = repo.HasAlmanaxWebhookCallback(createWebhook.Callback); err != nil {
		http.Error(w, "Internal error.", http.StatusInternalServerError)
		return
	}

	if hasAlm {
		http.Error(w, "Callback already exists.", http.StatusConflict)
		return
	}

	requestedBonuses := map[string]*Set[string]{}
	requestedMentions := map[string]*Set[uint64]{}

	if createWebhook.Mentions != nil {

		if len(*createWebhook.Mentions) > 150 {
			http.Error(w, "Too many mentions.", http.StatusBadRequest)
			return
		}

		for bonusId, mentions := range *createWebhook.Mentions {
			if _, ok := requestedBonuses[bonusId]; !ok {
				requestedBonuses[bonusId] = NewSet[string]()
			}
			if _, ok := requestedMentions[bonusId]; !ok {
				requestedMentions[bonusId] = NewSet[uint64]()
			}
			if !possibleBonuses.Has(bonusId) {
				http.Error(w, "Unknown almanax bonus id: "+bonusId+".", http.StatusBadRequest)
				return
			}
			if requestedBonuses[bonusId].Has(bonusId) {
				http.Error(w, "Duplicate bonus id: "+bonusId+".", http.StatusBadRequest)
				return
			}
			requestedBonuses[bonusId].Add(bonusId)
			for _, mention := range mentions {
				if mention.DiscordId < 0 {
					http.Error(w, "Invalid mention id.", http.StatusBadRequest)
					return
				}

				if mention.PingDaysBefore != nil {
					if *mention.PingDaysBefore < 1 || *mention.PingDaysBefore > 30 {
						http.Error(w, "PingDaysBefore should be between 1 and 30.", http.StatusBadRequest)
						return
					}
				}
				requestedMentions[bonusId].Add(mention.DiscordId)
			}
		}
	}

	var uid uuid.UUID
	if uid, err = repo.CreateAlmanaxHook(CreateAlmanaxHook{
		Callback:       createWebhook.Callback,
		Subscriptions:  createWebhook.Subscriptions,
		Format:         createWebhook.Format,
		WantsIsoDate:   *createWebhook.WantsIsoDate,
		DailySettings:  *createWebhook.DailySettings,
		BonusWhitelist: createWebhook.BonusWhitelist,
		BonusBlacklist: createWebhook.BonusBlacklist,
		Mentions:       createWebhook.Mentions,
	}); err != nil {
		if err.Error() == "some feeds not found" {
			http.Error(w, "Some feeds not found.", http.StatusBadRequest)
			return
		} else {
			http.Error(w, "Internal error.", http.StatusInternalServerError)
			return
		}
	}

	alm, err := getAlm(uid, repo)
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
	if err = json.NewEncoder(w).Encode(toDTO(alm)); err != nil {
		http.Error(w, "Error encoding the response.", http.StatusInternalServerError)
		return
	}
}

func getAlm(parsedId uuid.UUID, repo Repository) (AlmanaxWebhook, error) {
	var err error = nil
	var hasWebhook bool
	hasWebhook, err = repo.HasAlmanaxWebhook(parsedId)
	if err != nil {
		return AlmanaxWebhook{}, err
	}

	if !hasWebhook {
		return AlmanaxWebhook{}, errors.New("not found")
	}

	hook, err := repo.GetAlmanaxHook(parsedId)
	if err != nil {
		return AlmanaxWebhook{}, err
	}

	subbedFeeds, err := repo.GetAlmanaxHookSubscriptions(parsedId)
	if err != nil {
		return AlmanaxWebhook{}, err
	}

	for _, feed := range subbedFeeds {
		hook.Subscriptions = append(hook.Subscriptions, Subscription{Id: feed.GetFeedName()})
	}

	return hook, nil
}

func handleGetAlmanax(w http.ResponseWriter, r *http.Request) {
	requestsCRUDTotal.Inc()
	requestsCRUDAlmanax.Inc()
	id := r.Context().Value("id").(string)
	var err error = nil

	var parsedId uuid.UUID
	parsedId, err = uuid.Parse(id)
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

	hook, err := getAlm(parsedId, repo)
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
	if err = json.NewEncoder(w).Encode(toDTO(hook)); err != nil {
		http.Error(w, "Error encoding the response.", http.StatusInternalServerError)
		return
	}
}

func handlePutAlmanax(w http.ResponseWriter, r *http.Request) {
	requestsCRUDTotal.Inc()
	requestsCRUDAlmanax.Inc()
	id := r.Context().Value("id").(string)
	var err error

	var parsedId uuid.UUID
	parsedId, err = uuid.Parse(id)
	if err != nil {
		http.Error(w, "Invalid id.", http.StatusBadRequest)
		return
	}

	var updateHook AlmanaxHookPut
	if err = json.NewDecoder(r.Body).Decode(&updateHook); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if updateHook.BonusBlacklist != nil && updateHook.BonusWhitelist != nil {
		http.Error(w, "Cannot set both bonus blacklist and whitelist.", http.StatusBadRequest)
		return
	}

	possibleBonuses, err := getPossibleAlmanaxBonuses(r.Context())
	if err != nil {
		http.Error(w, "Could not reach Almanax API.", http.StatusBadGateway)
		return
	}

	if updateHook.DailySettings != nil && updateHook.DailySettings.Timezone != nil {
		_, err = time.LoadLocation(*updateHook.DailySettings.Timezone)
		if err != nil {
			http.Error(w, "Timezone not valid.", http.StatusBadRequest)
			return
		}
	}

	if updateHook.DailySettings != nil && updateHook.DailySettings.MidnightOffset != nil {
		if *updateHook.DailySettings.MidnightOffset < 0 || *updateHook.DailySettings.MidnightOffset > 23 {
			http.Error(w, "Offset should be between 0 and 23 valid.", http.StatusBadRequest)
			return
		}
	}

	if updateHook.BonusBlacklist != nil && len(updateHook.BonusBlacklist) > 0 {
		for _, blacklistEntry := range updateHook.BonusBlacklist {
			if !possibleBonuses.Has(blacklistEntry) {
				http.Error(w, "Unknown almanax bonus id: "+blacklistEntry+".", http.StatusBadRequest)
				return
			}
		}
	}

	if updateHook.BonusWhitelist != nil && len(updateHook.BonusWhitelist) > 0 {
		for _, blacklistEntry := range updateHook.BonusWhitelist {
			if !possibleBonuses.Has(blacklistEntry) {
				http.Error(w, "Unknown almanax bonus id: "+blacklistEntry+".", http.StatusBadRequest)
				return
			}
		}
	}

	if updateHook.Mentions != nil {
		for bonusId, mentions := range *updateHook.Mentions {
			if !possibleBonuses.Has(bonusId) {
				http.Error(w, "Unknown almanax bonus id: "+bonusId+".", http.StatusBadRequest)
				return
			}
			for _, mention := range mentions {
				if mention.DiscordId < 0 {
					http.Error(w, "Invalid mention id.", http.StatusBadRequest)
					return
				}
			}
		}
	}

	var repo Repository
	if err = repo.Init(r.Context()); err != nil {
		http.Error(w, "Internal error.", http.StatusInternalServerError)
		return
	}
	defer repo.Deinit()

	var found bool
	found, err = repo.HasAlmanaxWebhook(parsedId)
	if err != nil {
		http.Error(w, "Internal error.", http.StatusInternalServerError)
		return
	}

	if !found {
		http.Error(w, "Not found.", http.StatusNotFound)
		return
	}

	if err = repo.UpdateAlmanaxHook(updateHook, parsedId); err != nil {
		if err.Error() == "some feeds not found" {
			http.Error(w, "Some feeds not found.", http.StatusBadRequest)
			return
		} else {
			http.Error(w, "Internal error.", http.StatusInternalServerError)
			return
		}
	}

	alm, err := getAlm(parsedId, repo)
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
	if err = json.NewEncoder(w).Encode(toDTO(alm)); err != nil {
		http.Error(w, "Error encoding the response.", http.StatusInternalServerError)
		return
	}
}

// utils for filter and fire hooks

func isNewHour(tick time.Time) bool {
	return tick.Minute() == 0
}

func almHookIsSetToFireNow(webhook AlmanaxWebhook, currTime time.Time) bool {
	location, err := time.LoadLocation(*webhook.DailySettings.Timezone)
	if err != nil {
		return false
	}
	return currTime.In(location).Hour() == *webhook.DailySettings.MidnightOffset
}

func localTimeFormat(lang string, almDateString string, translations map[string]map[string]string) (string, error) {
	parsedAlmTime, err := time.Parse("2006-01-02", almDateString)
	if err != nil {
		return "", err
	}

	var out string
	translatedWeekday := translations[lang][parsedAlmTime.Weekday().String()]
	out += translatedWeekday

	switch lang {
	case "fr":
		out += ", " + parsedAlmTime.Format("02/01/2006")
	case "en":
		out += ", " + parsedAlmTime.Format("02/01/2006")
	case "de":
		out += ", " + parsedAlmTime.Format("02.01.2006")
	case "es":
		out += ", " + parsedAlmTime.Format("02/01/2006")
	case "it":
		out += ", " + parsedAlmTime.Format("02/01/2006")
	default:
		return "", nil
	}

	return out, nil
}

func getFutureAlmData(almData map[string]dodugo.AlmanaxEntry, timezone string, daysAhead int) (dodugo.AlmanaxEntry, error) {
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return dodugo.AlmanaxEntry{}, err
	}
	localDate := time.Now().In(location).Add(time.Hour * 24 * time.Duration(daysAhead)).Format("2006-01-02")
	return almData[localDate], nil
}

func getLocalAlmData(almData map[string]dodugo.AlmanaxEntry, timezone string) (dodugo.AlmanaxEntry, error) {
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return dodugo.AlmanaxEntry{}, err
	}
	localDate := time.Now().In(location).Format("2006-01-02")
	return almData[localDate], nil
}

func atLeastOneWebhookIsSetToFireNow(webhooks []AlmanaxWebhook, currTime time.Time) bool {
	for _, webhook := range webhooks {
		if almHookIsSetToFireNow(webhook, currTime) {
			return true
		}
	}
	return false
}

func filterAlmanaxBonusWhiteBlacklist(webhook AlmanaxWebhook, almBonusType dodugo.GetMetaAlmanaxBonuses200ResponseInner) bool {
	isWhitelisted := false
	isBlacklisted := false
	if webhook.BonusWhitelist != nil && len(webhook.BonusWhitelist) > 0 {
		for _, bonus := range webhook.BonusWhitelist {
			if bonus == almBonusType.GetId() {
				isWhitelisted = true
				break
			}
		}
	} else if webhook.BonusBlacklist != nil && len(webhook.BonusBlacklist) > 0 {
		for _, bonus := range webhook.BonusBlacklist {
			if bonus == almBonusType.GetId() {
				isBlacklisted = true
				break
			}
		}
	}

	if webhook.BonusBlacklist != nil && isBlacklisted {
		return true
	}

	if webhook.BonusWhitelist != nil && !isWhitelisted {
		return true
	}

	return false
}

// fire hook handlers

func HandleTimeAlmanax(almFeed AlmanaxFeed, state AlmanaxState, tickTime time.Time, tickRate time.Duration, repo Repository) ([]AlmanaxSend, error) {
	var err error
	if !isNewHour(tickTime) {
		return nil, nil
	}

	var subbedWebhooks []AlmanaxWebhook
	if subbedWebhooks, err = repo.GetAlmanaxSubsForFeed(almFeed); err != nil {
		return nil, err
	}

	if len(subbedWebhooks) == 0 || !atLeastOneWebhookIsSetToFireNow(subbedWebhooks, tickTime) {
		return nil, nil
	}

	parisTz, err := time.LoadLocation("Europe/Paris") // default dofus time
	if err != nil {
		return nil, err
	}

	var dodugoClient = dodugo.NewAPIClient(dodugo.NewConfiguration())
	ctxRangeFrom := context.WithValue(context.Background(), "range[from]", tickTime.In(parisTz).Add(-24*time.Hour).Format("2006-01-02"))
	ctxRangeTo := context.WithValue(ctxRangeFrom, "range[size]", 31)
	almRes, _, err := dodugoClient.AlmanaxApi.GetAlmanaxRange(ctxRangeTo, almFeed.Language).Execute()
	if err != nil {
		return nil, err
	}

	almData := make(map[string]dodugo.AlmanaxEntry)
	for _, entry := range almRes {
		almData[entry.GetDate()] = entry
	}

	var sendWebhooks []IHook
	var onlyPres []bool
	for _, webhook := range subbedWebhooks {
		var preMentions map[int][]MentionDTO
		if webhook.Mentions != nil {
			preMentions, err = buildPreviewMentions(*webhook.Mentions, almData, *webhook.DailySettings.Timezone)
			if err != nil {
				return nil, err
			}
		}

		if !almHookIsSetToFireNow(webhook, tickTime) {
			continue
		}

		var localAlmData dodugo.AlmanaxEntry
		localAlmData, err = getLocalAlmData(almData, *webhook.DailySettings.Timezone)
		if err != nil {
			return nil, err
		}

		almBonus := localAlmData.GetBonus()
		almBonusType := almBonus.GetType()

		filterOut := filterAlmanaxBonusWhiteBlacklist(webhook, almBonusType)
		if filterOut && len(preMentions) == 0 {
			continue
		}
		onlyPres = append(onlyPres, filterOut)

		sendHooksTotal.Inc()
		sendHooksAlmanax.Inc()

		sendWebhooks = append(sendWebhooks, webhook)
	}

	if len(sendWebhooks) == 0 {
		return nil, nil
	}

	translations, err := repo.GetAllWeekdayTranslations()
	if err != nil {
		return nil, err
	}

	return []AlmanaxSend{
		{
			Feed: almFeed,
			BuildInfo: AlmanaxHookBuildInfo{
				almData:      almData,
				translations: translations,
			},
			Webhooks:        sendWebhooks,
			OnlyPreMentions: onlyPres,
		},
	}, nil
}

func buildPreviewMentions(hookMentions map[string][]MentionDTO, almData map[string]dodugo.AlmanaxEntry, tz string) (map[int][]MentionDTO, error) {
	var mentionsAcc map[int][]MentionDTO
	mentionsAcc = make(map[int][]MentionDTO) // daysAhead => mentions
	for bonus, mentions := range hookMentions {
		for _, mention := range mentions {
			if mention.PingDaysBefore == nil {
				continue
			}

			futureAlmData, err := getFutureAlmData(almData, tz, *mention.PingDaysBefore)
			if err != nil {
				return nil, err
			}

			futureBonus := futureAlmData.GetBonus()
			futureBonusType := futureBonus.GetType()
			if futureBonusType.GetId() == bonus {
				if _, ok := mentionsAcc[*mention.PingDaysBefore]; ok {
					mentionsAcc[*mention.PingDaysBefore] = append(mentionsAcc[*mention.PingDaysBefore], mention)
				} else {
					mentionsAcc[*mention.PingDaysBefore] = []MentionDTO{mention}
				}
			}
		}
	}

	return mentionsAcc, nil
}

func buildDiscordHookAlmanax(almanaxSend AlmanaxSend) ([]PreparedHook, error) {
	var res []PreparedHook
	for webhookIdx, webhook := range almanaxSend.Webhooks {
		localAlmData, err := getLocalAlmData(almanaxSend.BuildInfo.almData, webhook.GetTimezone())
		if err != nil {
			return nil, err
		}

		var almLocalDate string
		if webhook.IsWantIsoDate() {
			almLocalDate = localAlmData.GetDate()
		} else {
			var err error
			almLocalDate, err = localTimeFormat(almanaxSend.Feed.Language, localAlmData.GetDate(), almanaxSend.BuildInfo.translations)
			if err != nil {
				return nil, err
			}
		}

		var imgBestResolution string
		tribute := localAlmData.GetTribute()
		almItem := tribute.GetItem()
		itemImageUrls := almItem.GetImageUrls()
		if itemImageUrls.HasSd() {
			urls := almItem.GetImageUrls()
			imgBestResolution = urls.GetSd()
		} else {
			imgBestResolution = itemImageUrls.GetIcon()
		}

		almBonus := localAlmData.GetBonus()
		almBonusType := almBonus.GetType()

		mentionString := ""
		var beforeMentions []DiscordEmbedField
		if webhook.GetMentions() != nil {
			hookMentions := *webhook.GetMentions()
			if mentions, ok := hookMentions[almBonusType.GetId()]; ok {
				var mentionStrings []string
				for _, mention := range mentions {
					idStr := strconv.FormatUint(mention.DiscordId, 10)
					found := false
					for _, alreadyInsertedMention := range mentionStrings {
						if strings.Contains(alreadyInsertedMention, idStr) {
							found = true // skip already inserted mentions (when using multiple ones for multiple days in advance)
							break
						}
					}
					if found {
						continue
					}

					if mention.IsRole {
						mentionStrings = append(mentionStrings, "<@&"+idStr+">")
					}
					mentionStrings = append(mentionStrings, "<@"+idStr+">")
				}
				mentionString = strings.Join(mentionStrings, " ")
			}

			var mentionsAcc map[int][]MentionDTO
			mentionsAcc, err = buildPreviewMentions(hookMentions, almanaxSend.BuildInfo.almData, webhook.GetTimezone())

			for daysBefore, mentions := range mentionsAcc {
				futureAlmData, err := getFutureAlmData(almanaxSend.BuildInfo.almData, webhook.GetTimezone(), daysBefore)
				if err != nil {
					return nil, err
				}

				futureBonus := futureAlmData.GetBonus()
				futureBonusType := futureBonus.GetType()

				var mentionStrings []string
				for _, mention := range mentions {
					idStr := strconv.FormatUint(mention.DiscordId, 10)
					if mention.IsRole {
						mentionStrings = append(mentionStrings, "<@&"+idStr+">")
					} else {
						mentionStrings = append(mentionStrings, "<@"+idStr+">")
					}
				}

				langCode := almanaxSend.Feed.GetFeedName()[len(almanaxSend.Feed.GetFeedName())-2:] // TODO query db for lang code
				var almTitle string
				switch langCode {
				case "fr":
					if daysBefore == 1 {
						almTitle = fmt.Sprintf("%s demain !", futureBonusType.GetName())
					} else {
						almTitle = fmt.Sprintf("%s dans %d jours !", futureBonusType.GetName(), daysBefore)
					}
				case "es":
					if daysBefore == 1 {
						almTitle = fmt.Sprintf("%s mañana!", futureBonusType.GetName())
					} else {
						almTitle = fmt.Sprintf("%s en %d días!", futureBonusType.GetName(), daysBefore)
					}
				case "de":
					if daysBefore == 1 {
						almTitle = fmt.Sprintf("%s morgen!", futureBonusType.GetName())
					} else {
						almTitle = fmt.Sprintf("%s in %d Tagen!", futureBonusType.GetName(), daysBefore)
					}
				case "it":
					if daysBefore == 1 {
						almTitle = fmt.Sprintf("%s domani!", futureBonusType.GetName())
					} else {
						almTitle = fmt.Sprintf("%s in %d giorni!", futureBonusType.GetName(), daysBefore)
					}
				default:
					if daysBefore == 1 {
						almTitle = fmt.Sprintf("%s tomorrow!", futureBonusType.GetName())
					} else {
						almTitle = fmt.Sprintf("%s in %d days!", futureBonusType.GetName(), daysBefore)
					}
				}

				beforeMentions = append(beforeMentions, DiscordEmbedField{
					Name:  almTitle,
					Value: fmt.Sprintf("%s\n%s", strings.Join(mentionStrings, " "), futureBonus.GetDescription()),
				})
			}
		}

		var discordWebhook DiscordWebhook
		discordWebhook.Username = "Almanax"
		discordWebhook.AvatarUrl = "https://discord.dofusdude.com/almanax_daily.jpg"
		if almanaxSend.OnlyPreMentions[webhookIdx] {
			discordWebhook.Content = nil
			langCode := almanaxSend.Feed.GetFeedName()[len(almanaxSend.Feed.GetFeedName())-2:] // TODO query db for lang code
			var previewTranslation string
			switch langCode {
			case "fr":
				previewTranslation = "Remarque"
			case "es":
				previewTranslation = "Pista"
			case "de":
				previewTranslation = "Hinweis"
			case "it":
				previewTranslation = "Suggerimento"
			default:
				previewTranslation = "Hint"
			}

			discordWebhook.Embeds = []DiscordEmbed{
				{
					Title:  &previewTranslation,
					Color:  16777215,
					Fields: beforeMentions,
				},
			}
		} else {
			if mentionString == "" {
				discordWebhook.Content = nil
			} else {
				discordWebhook.Content = &mentionString
			}

			discordWebhook.Embeds = []DiscordEmbed{
				{
					Title: &almLocalDate,
					Color: 16777215,
					Thumbnail: &DiscordImage{
						Url: imgBestResolution,
					},
					Fields: []DiscordEmbedField{
						{
							Name:  ":zap: " + almBonusType.GetName(),
							Value: fmt.Sprintf("%s\n\n:moneybag: %d %s", almBonus.GetDescription(), tribute.GetQuantity(), almItem.GetName()),
						},
					},
				},
			}

			if len(beforeMentions) > 0 {
				discordWebhook.Embeds[0].Fields = append(discordWebhook.Embeds[0].Fields, beforeMentions...)
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

func ListenAlmanax(ctx context.Context, feed AlmanaxFeed) {
	var almanaxState AlmanaxState
	Listen(ctx, AlmanaxPollingRate, feed, almanaxState, HandleTimeAlmanax, buildDiscordHookAlmanax)
}
