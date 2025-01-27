package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
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
		WeeklyWeekday: webhook.WeeklyWeekday,
		Intervals:     webhook.Intervals,
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
	var err error

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
	almBonuses, _, err := almClient.MetaAPI.GetMetaAlmanaxBonuses(ctx, "en").Execute()
	if err != nil {
		return nil, err
	}

	possibleBonuses := NewSet[string]()
	for _, bonus := range almBonuses {
		possibleBonuses.Add(bonus.GetId())
	}

	return possibleBonuses, nil
}

func validateIntervals(intervals []string) ([]string, bool) {
	intervalSet := NewSet[string]()
	for _, interval := range intervals {
		lowerInterval := strings.ToLower(interval)
		intervalSet.Add(lowerInterval)
		switch lowerInterval {
		case "daily":
		case "weekly":
		case "monthly":
		default:
			return nil, false
		}
	}
	return intervalSet.Slice(), true
}

func validateWeekday(weekday string) (string, bool) {
	lowerWeekday := strings.ToLower(weekday)
	switch lowerWeekday {
	case "monday":
	case "tuesday":
	case "wednesday":
	case "thursday":
	case "friday":
	case "saturday":
	case "sunday":
	default:
		return lowerWeekday, false
	}
	return lowerWeekday, true
}

func handleCreateAlmanax(w http.ResponseWriter, r *http.Request) {
	requestsCRUDTotal.Inc()
	requestsCRUDAlmanax.Inc()
	var err error
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

	if createWebhook.Intervals == nil || len(createWebhook.Intervals) == 0 {
		createWebhook.Intervals = []string{"daily"}
	}

	var ok bool
	if createWebhook.Intervals, ok = validateIntervals(createWebhook.Intervals); !ok {
		http.Error(w, "An interval must be one of daily, weekly or monthly.", http.StatusBadRequest)
	}

	if createWebhook.WeeklyWeekday == nil && sliceContains(createWebhook.Intervals, "sunday") {
		defaultWeekday := "monday"
		createWebhook.WeeklyWeekday = &defaultWeekday
	}

	if createWebhook.WeeklyWeekday != nil {
		if *createWebhook.WeeklyWeekday, ok = validateWeekday(*createWebhook.WeeklyWeekday); !ok {
			http.Error(w, "Unknown weekly weekday: "+*createWebhook.WeeklyWeekday+".", http.StatusBadRequest)
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
	requestedMentions := map[string]*Set[json.Number]{}

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
				requestedMentions[bonusId] = NewSet[json.Number]()
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
				if mention.PingDaysBefore != nil {
					if *mention.PingDaysBefore < 1 || *mention.PingDaysBefore > 31 {
						http.Error(w, "PingDaysBefore should be between 1 and 31.", http.StatusBadRequest)
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
		Intervals:      createWebhook.Intervals,
		WeeklyWeekday:  createWebhook.WeeklyWeekday,
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
	var err error
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
	var err error

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

	var ok bool
	if updateHook.Intervals, ok = validateIntervals(updateHook.Intervals); !ok {
		http.Error(w, "An interval must be one of daily, weekly or monthly.", http.StatusBadRequest)
	}

	if updateHook.WeeklyWeekday != nil {
		if *updateHook.WeeklyWeekday, ok = validateWeekday(*updateHook.WeeklyWeekday); !ok {
			http.Error(w, "Unknown weekly weekday: "+*updateHook.WeeklyWeekday+".", http.StatusBadRequest)
		}
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

	if updateHook.BonusBlacklist != nil {
		for _, blacklistEntry := range updateHook.BonusBlacklist {
			if !possibleBonuses.Has(blacklistEntry) {
				http.Error(w, "Unknown almanax bonus id: "+blacklistEntry+".", http.StatusBadRequest)
				return
			}
		}
	}

	if updateHook.BonusWhitelist != nil {
		for _, blacklistEntry := range updateHook.BonusWhitelist {
			if !possibleBonuses.Has(blacklistEntry) {
				http.Error(w, "Unknown almanax bonus id: "+blacklistEntry+".", http.StatusBadRequest)
				return
			}
		}
	}

	if updateHook.Mentions != nil {
		for bonusId := range *updateHook.Mentions {
			if !possibleBonuses.Has(bonusId) {
				http.Error(w, "Unknown almanax bonus id: "+bonusId+".", http.StatusBadRequest)
				return
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

func endOfMonth(date time.Time) time.Time {
	return date.AddDate(0, 1, -date.Day())
}

func almHookIsSetToFireNow(webhook AlmanaxWebhook, currTime time.Time) ([]string, error) {
	var toFire []string
	location, err := time.LoadLocation(*webhook.DailySettings.Timezone)
	if err != nil {
		return nil, err
	}

	localeTime := currTime.In(location)

	if sliceContains(webhook.Intervals, "daily") && localeTime.Hour() == *webhook.DailySettings.MidnightOffset {
		toFire = append(toFire, "daily")
	}

	if sliceContains(webhook.Intervals, "weekly") && strings.ToLower(localeTime.Weekday().String()) == *webhook.WeeklyWeekday && localeTime.Hour() == *webhook.DailySettings.MidnightOffset {
		toFire = append(toFire, "weekly")
	}

	if sliceContains(webhook.Intervals, "monthly") && endOfMonth(localeTime).Day() == localeTime.Day() && localeTime.Hour() == *webhook.DailySettings.MidnightOffset {
		toFire = append(toFire, "monthly")
	}

	return toFire, nil
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

func getFutureAlmData(almData map[string]dodugo.Almanax, timezone string, daysAhead int) (dodugo.Almanax, error) {
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return dodugo.Almanax{}, err
	}
	localDate := time.Now().In(location).Add(time.Hour * 24 * time.Duration(daysAhead)).Format("2006-01-02")
	return almData[localDate], nil
}

func getLocalAlmData(almData map[string]dodugo.Almanax, timezone string) (dodugo.Almanax, error) {
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return dodugo.Almanax{}, err
	}
	localDate := time.Now().In(location).Format("2006-01-02")
	return almData[localDate], nil
}

func getLocalAlmDataRange(almData map[string]dodugo.Almanax, timezone string, start time.Time, end time.Time) ([]dodugo.Almanax, error) {
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, err
	}
	var out []dodugo.Almanax
	for i := 0; i <= int(end.Sub(start).Hours()/24); i++ {
		localDate := start.Add(time.Hour * 24 * time.Duration(i)).In(location).Format("2006-01-02")
		out = append(out, almData[localDate])
	}
	return out, nil
}

func atLeastOneWebhookIsSetToFireNow(webhooks []AlmanaxWebhook, currTime time.Time) (bool, error) {
	var toFire []string
	var err error
	for _, webhook := range webhooks {
		if toFire, err = almHookIsSetToFireNow(webhook, currTime); err != nil {
			return false, err
		}
		if len(toFire) > 0 {
			return true, nil
		}
	}
	return false, nil
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

func buildAlmSpan(tickTime time.Time, intervalType string, tz string, almData map[string]dodugo.Almanax) ([]dodugo.Almanax, error) {
	var err error
	var location *time.Location
	location, err = time.LoadLocation(tz)
	if err != nil {
		return nil, err
	}

	var start time.Time
	var end time.Time

	start = tickTime.In(location).Add(time.Hour * 24)

	switch intervalType {
	case "weekly":
		end = tickTime.In(location).Add(time.Hour * 24 * 7)
	case "monthly":
		end = endOfMonth(start)
	}

	var localAlmData []dodugo.Almanax
	localAlmData, err = getLocalAlmDataRange(almData, tz, start, end)
	if err != nil {
		return nil, err
	}

	return localAlmData, nil
}

// fire hook handlers

func HandleTimeAlmanax(almFeed AlmanaxFeed, _ any, tickTime time.Time, _ time.Duration, repo Repository) ([]AlmanaxSend, error) {
	var err error
	if !isNewHour(tickTime) {
		return nil, nil
	}

	var subbedWebhooks []AlmanaxWebhook
	if subbedWebhooks, err = repo.GetAlmanaxSubsForFeed(almFeed); err != nil {
		return nil, err
	}

	var atLeastFireOne bool
	if atLeastFireOne, err = atLeastOneWebhookIsSetToFireNow(subbedWebhooks, tickTime); err != nil {
		return nil, err
	}

	if len(subbedWebhooks) == 0 || !atLeastFireOne {
		return nil, nil
	}

	parisTz, err := time.LoadLocation("Europe/Paris") // default dofus time
	if err != nil {
		return nil, err
	}

	dodugoCfg := &dodugo.Configuration{
		DefaultHeader: make(map[string]string),
		UserAgent:     "ankama-discord-hooks",
		Debug:         false,
		Servers: dodugo.ServerConfigurations{
			{
				URL:         "https://api.dofusdu.de",
				Description: "API",
			},
		},
		OperationServers: map[string]dodugo.ServerConfigurations{},
	}
	var dodugoClient = dodugo.NewAPIClient(dodugoCfg)
	options := dodugoClient.AlmanaxAPI.GetAlmanaxRange(context.Background(), almFeed.Language)
	options = options.Timezone(parisTz.String()).RangeFrom(tickTime.In(parisTz).Add(-24 * time.Hour).Format("2006-01-02")).RangeSize(33)
	almRes, _, err := options.Execute()
	if err != nil {
		return nil, err
	}

	almData := make(map[string]dodugo.Almanax)
	for _, entry := range almRes {
		almData[entry.GetDate()] = entry
	}

	var sendWebhooks []IHook
	var onlyPres []bool
	var intervals []string
	for _, webhook := range subbedWebhooks {
		var preMentions map[int][]MentionDTO
		if webhook.Mentions != nil {
			preMentions, err = buildPreviewMentions(*webhook.Mentions, almData, *webhook.DailySettings.Timezone)
			if err != nil {
				return nil, err
			}
		}

		var toFire []string
		if toFire, err = almHookIsSetToFireNow(webhook, tickTime); err != nil {
			return nil, err
		}
		if len(toFire) == 0 {
			continue
		}

		for _, intervalType := range toFire {
			// check if filters will hide the hook completely
			if intervalType == "daily" {
				var localAlmData dodugo.Almanax
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
			} else { // weekly or monthly
				var localAlmData []dodugo.Almanax
				if localAlmData, err = buildAlmSpan(tickTime, intervalType, webhook.GetTimezone(), almData); err != nil {
					return nil, err
				}

				var filteredAlmData []dodugo.Almanax
				for _, almEntry := range localAlmData {
					almBonus := almEntry.GetBonus()
					almBonusType := almBonus.GetType()

					filterOut := filterAlmanaxBonusWhiteBlacklist(webhook, almBonusType)
					if filterOut && len(preMentions) == 0 {
						continue
					}
					filteredAlmData = append(filteredAlmData, almEntry)
				}

				if len(filteredAlmData) == 0 {
					continue
				}

				onlyPres = append(onlyPres, false)
			}

			sendHooksTotal.Inc()
			sendHooksAlmanax.Inc()
			sendWebhooks = append(sendWebhooks, webhook)
			intervals = append(intervals, intervalType)
		}
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
			IntervalType:    intervals,
			TickTime:        tickTime,
		},
	}, nil
}

func buildPreviewMentions(hookMentions map[string][]MentionDTO, almData map[string]dodugo.Almanax, tz string) (map[int][]MentionDTO, error) {
	mentionsAcc := make(map[int][]MentionDTO) // daysAhead => mentions
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
				mentionsAcc[*mention.PingDaysBefore] = append(mentionsAcc[*mention.PingDaysBefore], mention)
			}
		}
	}

	return mentionsAcc, nil
}

func formatKamas(kamas int32) string {
	kamasStr := fmt.Sprintf("%d", kamas)
	n := len(kamasStr)
	formatted := make([]byte, 0, n+(n-1)/3)
	for i := 0; i < n; i++ {
		if i > 0 && (n-i)%3 == 0 {
			formatted = append(formatted, ' ')
		}
		formatted = append(formatted, kamasStr[i])
	}

	return string(formatted) + " K"
}

func buildDiscordHookAlmanax(almanaxSend AlmanaxSend) ([]PreparedHook, error) {
	var res []PreparedHook
	var err error
	for webhookIdx, webhook := range almanaxSend.Webhooks {
		var discordWebhook DiscordWebhook
		if almanaxSend.IntervalType[webhookIdx] == "daily" {
			var localAlmData dodugo.Almanax
			localAlmData, err = getLocalAlmData(almanaxSend.BuildInfo.almData, webhook.GetTimezone())
			if err != nil {
				return nil, err
			}

			var almLocalDate string
			if webhook.IsWantIsoDate() {
				almLocalDate = localAlmData.GetDate()
			} else {
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
						idStr := mention.DiscordId.String()
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
				if err != nil {
					return nil, err
				}

				for daysBefore, mentions := range mentionsAcc {
					var futureAlmData dodugo.Almanax
					futureAlmData, err = getFutureAlmData(almanaxSend.BuildInfo.almData, webhook.GetTimezone(), daysBefore)
					if err != nil {
						return nil, err
					}

					futureBonus := futureAlmData.GetBonus()
					futureBonusType := futureBonus.GetType()

					var mentionStrings []string
					for _, mention := range mentions {
						idStr := mention.DiscordId.String()
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
						Name:   almTitle,
						Value:  fmt.Sprintf("%s\n%s", strings.Join(mentionStrings, " "), futureBonus.GetDescription()),
						Inline: false,
					})
				}
			}

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
						Color:  3684408,
						Fields: beforeMentions,
					},
				}
			} else {
				if mentionString == "" {
					discordWebhook.Content = nil
				} else {
					discordWebhook.Content = &mentionString
				}

				kamas := formatKamas(localAlmData.GetRewardKamas())

				discordWebhook.Embeds = []DiscordEmbed{
					{
						Title: &almLocalDate,
						Color: 3684408,
						Thumbnail: &DiscordImage{
							Url: imgBestResolution,
						},
						Fields: []DiscordEmbedField{
							{
								Name:   ":zap: " + almBonusType.GetName(),
								Value:  fmt.Sprintf("*%s*\n\n:moneybag: %s\n\n:pray: %dx **%s**", almBonus.GetDescription(), kamas, tribute.GetQuantity(), almItem.GetName()),
								Inline: false,
							},
						},
					},
				}

				if len(beforeMentions) > 0 {
					discordWebhook.Embeds[0].Fields = append(discordWebhook.Embeds[0].Fields, beforeMentions...)
				}
			}
		} else {
			var localAlmData []dodugo.Almanax
			if localAlmData, err = buildAlmSpan(almanaxSend.TickTime, almanaxSend.IntervalType[webhookIdx], webhook.GetTimezone(), almanaxSend.BuildInfo.almData); err != nil {
				log.Printf("Error building almanax span: %s", err)
				continue
			}

			discordWebhook.Username = "Almanax"
			discordWebhook.AvatarUrl = "https://discord.dofusdude.com/almanax_daily.jpg"
			var almLocalDateStart string
			var almLocalDateEnd string
			if webhook.IsWantIsoDate() {
				almLocalDateStart = localAlmData[0].GetDate()
				almLocalDateEnd = localAlmData[len(localAlmData)-1].GetDate()
			} else {
				if almLocalDateStart, err = localTimeFormat(almanaxSend.Feed.Language, localAlmData[0].GetDate(), almanaxSend.BuildInfo.translations); err != nil {
					return nil, err
				}
				if almLocalDateEnd, err = localTimeFormat(almanaxSend.Feed.Language, localAlmData[len(localAlmData)-1].GetDate(), almanaxSend.BuildInfo.translations); err != nil {
					return nil, err
				}
			}

			var content string
			if almanaxSend.IntervalType[webhookIdx] == "weekly" {
				switch almanaxSend.Feed.GetFeedName() {
				case "almanax_fr":
					content = "Voici les bonus de la semaine !"
				case "almanax_es":
					content = "¡Aquí están los bonos de la semana!"
				case "almanax_de":
					content = "Hier sind die Boni der Woche!"
				case "almanax_it":
					content = "Ecco i bonus della settimana!"
				default:
					content = "Here are the bonuses for the week!"
				}
			} else if almanaxSend.IntervalType[webhookIdx] == "monthly" {
				switch almanaxSend.Feed.GetFeedName() {
				case "almanax_fr":
					content = "Voici les bonus du mois !"
				case "almanax_es":
					content = "¡Aquí están los bonos del mes!"
				case "almanax_de":
					content = "Hier sind die Boni des Monats!"
				case "almanax_it":
					content = "Ecco i bonus del mese!"
				default:
					content = "Here are the bonuses for the month!"
				}
			}
			discordWebhook.Content = &content

			localeWeekSpan := almLocalDateStart + " - " + almLocalDateEnd
			discordWebhook.Embeds = []DiscordEmbed{
				{
					Title: &localeWeekSpan,
					Color: 3684408,
				},
			}

			currentEmbed := 0
			itemsAgg := make(map[string]int32)
			for _, almEntry := range localAlmData {
				var almLocalDate string
				if webhook.IsWantIsoDate() {
					almLocalDate = almEntry.GetDate()
				} else {
					if almLocalDate, err = localTimeFormat(almanaxSend.Feed.Language, almEntry.GetDate(), almanaxSend.BuildInfo.translations); err != nil {
						return nil, err
					}
				}
				kamas := formatKamas(almEntry.GetRewardKamas())
				tribute := almEntry.GetTribute()
				almItem := tribute.GetItem()
				almBonus := almEntry.GetBonus()
				almBonusType := almBonus.GetType()
				discordWebhook.Embeds[currentEmbed].Fields = append(discordWebhook.Embeds[currentEmbed].Fields, DiscordEmbedField{
					Name:   fmt.Sprintf("%s – %s", almLocalDate, almBonusType.GetName()),
					Value:  fmt.Sprintf("*%s*\n%s\n%dx **%s**", almBonus.GetDescription(), kamas, tribute.GetQuantity(), almItem.GetName()),
					Inline: len(discordWebhook.Embeds[currentEmbed].Fields)%2 != 0,
				})

				if len(discordWebhook.Embeds[currentEmbed].Fields) == 16 {
					*discordWebhook.Embeds[currentEmbed].Title = almLocalDateStart + " - " + almLocalDate + " (1/2)"
					var parsedLastDate time.Time
					if parsedLastDate, err = time.Parse("2006-01-02", almEntry.GetDate()); err != nil {
						return nil, err
					}
					secondStartTime := parsedLastDate.Add(time.Hour * 24).Format("2006-01-02")
					var secondStart string
					if webhook.IsWantIsoDate() {
						almLocalDateStart = secondStartTime
					} else {
						if secondStart, err = localTimeFormat(almanaxSend.Feed.Language, secondStartTime, almanaxSend.BuildInfo.translations); err != nil {
							return nil, err
						}
					}

					localeWeekSpanPart2 := secondStart + " - " + almLocalDateEnd + " (2/2)"
					discordWebhook.Embeds = append(discordWebhook.Embeds, DiscordEmbed{
						Title: &localeWeekSpanPart2,
						Color: 3684408,
					})
					currentEmbed++
				}

				itemsAgg[almItem.GetName()] += tribute.GetQuantity()
			}

			var totalTranslation string
			switch almanaxSend.Feed.GetFeedName() {
			case "almanax_fr":
				totalTranslation = "Total"
			case "almanax_es":
				totalTranslation = "Total"
			case "almanax_de":
				totalTranslation = "Gesamt"
			case "almanax_it":
				totalTranslation = "Totale"
			default:
				totalTranslation = "Total"
			}

			var totalItems string
			for itemName, itemQuantity := range itemsAgg {
				totalItems += fmt.Sprintf("%dx **%s**\n", itemQuantity, itemName)
			}

			discordWebhook.Embeds[currentEmbed].Fields = append(discordWebhook.Embeds[currentEmbed].Fields, DiscordEmbedField{
				Name:   totalTranslation,
				Value:  totalItems,
				Inline: false,
			})
		}

		var jsonBody []byte
		if jsonBody, err = json.Marshal(discordWebhook); err != nil {
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
	Listen(ctx, AlmanaxPollingRate, feed, nil, HandleTimeAlmanax, buildDiscordHookAlmanax)
}
