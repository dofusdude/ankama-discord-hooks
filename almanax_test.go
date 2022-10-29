package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dofusdude/dodugo"
	"github.com/steinfletcher/apitest"
	jsonpath "github.com/steinfletcher/apitest-jsonpath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestHourCheck(t *testing.T) {
	parsedTime, _ := time.Parse(time.RFC3339, "2021-01-01T00:02:35Z")
	assert.False(t, isNewHour(parsedTime))

	parsedTime, _ = time.Parse(time.RFC3339, "2021-01-01T00:00:35Z")
	assert.True(t, isNewHour(parsedTime))

	parsedTime, _ = time.Parse(time.RFC3339, "2021-01-01T06:59:35Z")
	assert.False(t, isNewHour(parsedTime))

	parsedTime, _ = time.Parse(time.RFC3339, "2021-01-01T00:00:59Z")
	assert.True(t, isNewHour(parsedTime))

	parsedTime, _ = time.Parse(time.RFC3339, "2021-01-01T00:00:00Z")
	assert.True(t, isNewHour(parsedTime))
}

func TestFire(t *testing.T) {
	testTz := "Europe/Paris"
	tzOffset := 1
	testhook1 := AlmanaxWebhook{
		DailySettings: WebhookDailySettings{
			Timezone:       &testTz,
			MidnightOffset: &tzOffset,
		},
	}

	triggerTime := time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC)
	assert.True(t, almHookIsSetToFireNow(testhook1, triggerTime))

	triggerTime = time.Date(2021, 1, 1, 23, 0, 0, 0, time.UTC)
	assert.False(t, almHookIsSetToFireNow(testhook1, triggerTime))
}

type AlmanaxTestSuite struct {
	suite.Suite
	db           Repository
	sut          *httptest.Server
	almBonusMock *apitest.Mock
	discordCheck []*apitest.Mock
}

func (suite *AlmanaxTestSuite) SetupSuite() {
	ReadEnvs()

	var repo Repository
	if err := repo.Init(context.Background()); err != nil {
		suite.T().Fatal(err)
	}

	suite.db = repo
	suite.sut = httptest.NewServer(Router())
	suite.discordCheck = append(suite.discordCheck, apitest.NewMock().
		Get("https://discord.com/api/webhooks/123/abc").
		RespondWith().
		Status(http.StatusOK).
		End())
	for i := 1; i < 7; i++ {
		suite.discordCheck = append(suite.discordCheck, apitest.NewMock().
			Get(fmt.Sprintf("https://discord.com/api/webhooks/123/abc%d", i)).
			RespondWith().
			Status(http.StatusOK).
			End())
	}
	suite.almBonusMock = apitest.NewMock().
		Get("https://api.dofusdu.de/dofus2/meta/en/almanax/bonuses").
		RespondWith().
		Body(`[
		  {
			"id": "rewardbonus",
			"name": "Reward Bonus"
		  },
		  {
			"id": "loot",
			"name": "Loot"
		  }
		]`).
		Status(http.StatusOK).
		End()
}

func (suite *AlmanaxTestSuite) TearDownSuite() {
	suite.sut.Close()
}

func (suite *AlmanaxTestSuite) SetupTest() {
}

func (suite *AlmanaxTestSuite) TearDownTest() {
	if err := testutilCleartables(); err != nil {
		suite.T().Fatal(err)
	}
}

func (suite *AlmanaxTestSuite) Test_Feeds() {
	apitest.New().
		Mocks(suite.almBonusMock).
		Handler(Router()).
		Get("/meta/webhooks/almanax").
		Expect(suite.T()).
		Status(http.StatusOK).
		Assert(jsonpath.Chain().
			Contains("$.subscriptions", "dofus2_en").
			Contains("$.subscriptions", "dofus2_fr").
			Contains("$.subscriptions", "dofus2_de").
			Contains("$.subscriptions", "dofus2_es").
			Contains("$.subscriptions", "dofus2_it").
			End(),
		).
		End()
}

func (suite *AlmanaxTestSuite) TestRepository_GetAllWeekdayTranslations() {
	translations, err := suite.db.GetAllWeekdayTranslations()
	assert.Nil(suite.T(), err)

	assert.NotNil(suite.T(), translations)
	assert.Equal(suite.T(), "Samstag", translations["de"]["Saturday"])
}

func (suite *AlmanaxTestSuite) Test_CRUD_Create() {
	tz := "Europe/Paris"
	tzOffset := 1

	apitest.New().
		Mocks(suite.almBonusMock, suite.discordCheck[0]).
		Handler(Router()).
		Post("/webhooks/almanax").
		JSON(AlmanaxHookPost{
			BonusBlacklist: nil,
			BonusWhitelist: nil,
			DailySettings: &WebhookDailySettings{
				Timezone:       &tz,
				MidnightOffset: &tzOffset,
			},
			Callback: "https://discord.com/api/webhooks/123/abc",
			Subscriptions: []string{
				"dofus2_en",
			},
			Format: "discord",
		}).
		Expect(suite.T()).
		Status(http.StatusCreated).
		Assert(jsonpath.Chain().
			Present("$.id").
			NotPresent("$.last_fired_at").
			Equal("$.daily_settings.timezone", "Europe/Paris").
			Equal("$.daily_settings.midnight_offset", float64(1)).
			NotPresent("$.callback").
			Equal("$.subscriptions[0].id", "dofus2_en").
			Equal("$.bonus_whitelist", nil).
			Equal("$.bonus_blacklist", nil).
			Equal("$.mentions", nil).
			Present("$.created_at").
			Present("$.updated_at").
			Equal("$.iso_date", false).
			Equal("$.format", "discord").
			End(),
		).
		End()

	apitest.New().
		Mocks(suite.almBonusMock, suite.discordCheck[3]).
		Handler(Router()).
		Post("/webhooks/almanax").
		JSON(AlmanaxHookPost{
			BonusBlacklist: []string{
				"loot",
			},
			BonusWhitelist: []string{
				"rewardbonus",
			},
			DailySettings: &WebhookDailySettings{
				Timezone:       &tz,
				MidnightOffset: &tzOffset,
			},
			Callback: "https://discord.com/api/webhooks/123/abc3",
			Subscriptions: []string{
				"dofus2_en",
			},
			Format: "discord",
		}).
		Expect(suite.T()).
		Status(http.StatusBadRequest).
		End()

	apitest.New().
		Mocks(suite.almBonusMock, suite.discordCheck[3]).
		Handler(Router()).
		Post("/webhooks/almanax").
		JSON(AlmanaxHookPost{
			BonusBlacklist: []string{
				"loot",
			},
			BonusWhitelist: []string{
				"rewardbonus",
			},
			DailySettings: &WebhookDailySettings{
				Timezone:       &tz,
				MidnightOffset: &tzOffset,
			},
			Callback: "https://discord.com/api/webhooks/123/abc2",
			Subscriptions: []string{
				"dofus2_en",
			}, // missing format
		}).
		Expect(suite.T()).
		Status(http.StatusBadRequest).
		End()

	apitest.New().
		Mocks(suite.almBonusMock, suite.discordCheck[2]).
		Handler(Router()).
		Post("/webhooks/almanax").
		JSON(AlmanaxHookPost{
			BonusBlacklist: []string{},
			Callback:       "https://discord.com/api/webhooks/123/abc2",
			Subscriptions: []string{
				"dofus2_en",
			},
			Format: "discord",
		}).
		Expect(suite.T()).
		Status(http.StatusCreated).
		Assert(jsonpath.Chain().
			Equal("$.subscriptions[0].id", "dofus2_en").
			Equal("$.bonus_whitelist", nil).
			Equal("$.bonus_blacklist", nil).
			End()).
		End()
}

func (suite *AlmanaxTestSuite) Test_CRUD_Create_Defaults() {
	apitest.New().
		Mocks(suite.almBonusMock, suite.discordCheck[0]).
		Handler(Router()).
		Post("/webhooks/almanax").
		JSON(AlmanaxHookPost{
			Callback: "https://discord.com/api/webhooks/123/abc",
			Subscriptions: []string{
				"dofus2_fr",
			},
			Format: "discord",
		}).
		Expect(suite.T()).
		Status(http.StatusCreated).
		Assert(jsonpath.Chain().
			Present("$.id").
			NotPresent("$.last_fired_at").
			Equal("$.daily_settings.timezone", "Europe/Paris").
			Equal("$.daily_settings.midnight_offset", float64(0)).
			NotPresent("$.callback").
			Equal("$.subscriptions[0].id", "dofus2_fr").
			Equal("$.bonus_whitelist", nil).
			Equal("$.bonus_blacklist", nil).
			Equal("$.mentions", nil).
			Present("$.created_at").
			Present("$.updated_at").
			Equal("$.iso_date", false).
			Equal("$.format", "discord").
			End(),
		).
		End()
}

func (suite *AlmanaxTestSuite) Test_CRUD_Create_Mentions() {
	wantIsoDate := true
	apitest.New().
		Mocks(suite.almBonusMock, suite.discordCheck[0]).
		Handler(Router()).
		Post("/webhooks/almanax").
		JSON(AlmanaxHookPost{
			Callback: "https://discord.com/api/webhooks/123/abc",
			Subscriptions: []string{
				"dofus2_fr",
			},
			Mentions: &map[string][]MentionDTO{
				"loot": {
					MentionDTO{
						DiscordId: 123,
						IsRole:    false,
					},
					MentionDTO{
						DiscordId: 124,
						IsRole:    true,
					},
				},
			},
			WantsIsoDate: &wantIsoDate,
			Format:       "discord",
		}).
		Expect(suite.T()).
		Status(http.StatusCreated).
		Assert(jsonpath.Chain().
			Present("$.id").
			NotPresent("$.last_fired_at").
			Equal("$.daily_settings.timezone", "Europe/Paris").
			Equal("$.daily_settings.midnight_offset", float64(0)).
			NotPresent("$.callback").
			Equal("$.subscriptions[0].id", "dofus2_fr").
			Equal("$.bonus_whitelist", nil).
			Equal("$.bonus_blacklist", nil).
			Equal("$.mentions.loot[0].discord_id", float64(123)).
			Equal("$.mentions.loot[0].is_role", false).
			Equal("$.mentions.loot[1].discord_id", float64(124)).
			Equal("$.mentions.loot[1].is_role", true).
			Present("$.created_at").
			Present("$.updated_at").
			Equal("$.iso_date", true).
			Equal("$.format", "discord").
			End(),
		).
		End()

	apitest.New().
		Mocks(suite.almBonusMock, suite.discordCheck[1]).
		Handler(Router()).
		Post("/webhooks/almanax").
		JSON(AlmanaxHookPost{
			Callback: "https://discord.com/api/webhooks/123/abc1",
			Subscriptions: []string{
				"dofus2_fr",
			},
			Mentions: &map[string][]MentionDTO{
				"loot": {
					MentionDTO{
						DiscordId: 123,
						IsRole:    false,
					},
				},
				"rewardbonus": {
					MentionDTO{
						DiscordId: 123,
						IsRole:    true,
					},
				},
			},
			Format: "discord",
		}).
		Expect(suite.T()).
		Status(http.StatusCreated).
		Assert(jsonpath.Chain().
			Equal("$.mentions.loot[0].discord_id", float64(123)).
			Equal("$.mentions.loot[0].is_role", false).
			Equal("$.mentions.rewardbonus[0].discord_id", float64(123)).
			Equal("$.mentions.rewardbonus[0].is_role", true).
			End(),
		).
		End()

	pingDaysAhead := 5
	apitest.New().
		Mocks(suite.almBonusMock, suite.discordCheck[2]).
		Handler(Router()).
		Post("/webhooks/almanax").
		JSON(AlmanaxHookPost{
			Callback: "https://discord.com/api/webhooks/123/abc2",
			Subscriptions: []string{
				"dofus2_fr",
			},
			Mentions: &map[string][]MentionDTO{
				"loot": {
					MentionDTO{
						DiscordId:      123,
						IsRole:         false,
						PingDaysBefore: &pingDaysAhead,
					},
				},
				"rewardbonus": {
					MentionDTO{
						DiscordId: 123,
						IsRole:    true,
					},
				},
			},
			Format: "discord",
		}).
		Expect(suite.T()).
		Status(http.StatusCreated).
		Assert(jsonpath.Chain().
			Equal("$.mentions.loot[0].discord_id", float64(123)).
			Equal("$.mentions.loot[0].is_role", false).
			Equal("$.mentions.loot[0].ping_days_before", float64(pingDaysAhead)).
			Equal("$.mentions.rewardbonus[0].discord_id", float64(123)).
			Equal("$.mentions.rewardbonus[0].is_role", true).
			End(),
		).
		End()
}

func (suite *AlmanaxTestSuite) Test_CRUD_Delete() {
	apitest.New().
		Mocks(suite.almBonusMock, suite.discordCheck[0]).
		Handler(Router()).
		Post("/webhooks/almanax").
		JSON(AlmanaxHookPost{
			Callback: "https://discord.com/api/webhooks/123/abc",
			Subscriptions: []string{
				"dofus2_fr",
			},
			Mentions: &map[string][]MentionDTO{
				"loot": {
					MentionDTO{
						DiscordId: 123,
						IsRole:    false,
					},
					MentionDTO{
						DiscordId: 124,
						IsRole:    true,
					},
				},
			},
			Format: "discord",
		}).
		Expect(suite.T()).
		Status(http.StatusCreated).
		End()

	uid, err := testutilGetlastinsertedwebhookid()
	assert.Nil(suite.T(), err)

	apitest.New().
		Mocks(suite.almBonusMock).
		Handler(Router()).
		Delete("/webhooks/almanax/" + uid.String()).
		Expect(suite.T()).
		Status(http.StatusNoContent).
		End()

	apitest.New().
		Mocks(suite.almBonusMock).
		Handler(Router()).
		Delete("/webhooks/almanax/" + uid.String()).
		Expect(suite.T()).
		Status(http.StatusNotFound).
		End()
}

func (suite *AlmanaxTestSuite) Test_CRUD_Create_UnknownTz() {
	tz := "Europe/Paris123"
	tzOffset := 1
	body := AlmanaxHookPost{
		BonusBlacklist: nil,
		BonusWhitelist: nil,
		DailySettings: &WebhookDailySettings{
			Timezone:       &tz,
			MidnightOffset: &tzOffset,
		},
		Callback: "https://discord.com/api/webhooks/123/abc",
		Subscriptions: []string{
			"dofus2_en",
		},
		Format: "discord",
	}

	apitest.New().
		Mocks(suite.almBonusMock, suite.discordCheck[0]).
		Handler(Router()).
		Post("/webhooks/almanax").
		JSON(body).
		Expect(suite.T()).
		Status(http.StatusBadRequest).
		End()
}

func (suite *AlmanaxTestSuite) Test_CRUD_Create_LargeOffset() {
	tz := "Europe/Paris"
	tzOffset := 24

	apitest.New().
		Mocks(suite.almBonusMock, suite.discordCheck[0]).
		Handler(Router()).
		Post("/webhooks/almanax").
		JSON(AlmanaxHookPost{
			BonusBlacklist: nil,
			BonusWhitelist: nil,
			DailySettings: &WebhookDailySettings{
				Timezone:       &tz,
				MidnightOffset: &tzOffset,
			},
			Callback: "https://discord.com/api/webhooks/123/abc",
			Subscriptions: []string{
				"dofus2_en",
			},
			Format: "discord",
		}).
		Expect(suite.T()).
		Status(http.StatusBadRequest).
		End()
}

func (suite *AlmanaxTestSuite) Test_CRUD_Create_And_Get() {
	tz := "Europe/Berlin"
	tzOffset := 1

	apitest.New().
		Mocks(suite.almBonusMock, suite.discordCheck[0]).
		Handler(Router()).
		Post("/webhooks/almanax").
		JSON(AlmanaxHookPost{
			BonusBlacklist: nil,
			BonusWhitelist: nil,
			DailySettings: &WebhookDailySettings{
				Timezone:       &tz,
				MidnightOffset: &tzOffset,
			},
			Callback: "https://discord.com/api/webhooks/123/abc",
			Subscriptions: []string{
				"dofus2_en",
			},
			Format: "discord",
		}).
		Expect(suite.T()).
		Status(http.StatusCreated).
		End()

	lastId, err := testutilGetlastinsertedwebhookid()
	if err != nil {
		suite.T().Fatal(err)
	}

	apitest.New().
		Handler(Router()).
		Get("/webhooks/almanax/" + lastId.String()).
		Expect(suite.T()).
		Status(http.StatusOK).
		Assert(jsonpath.Chain().
			Present("$.id").
			NotPresent("$.last_fired_at").
			Equal("$.daily_settings.timezone", "Europe/Berlin").
			Equal("$.daily_settings.midnight_offset", float64(1)).
			NotPresent("$.callback").
			Equal("$.subscriptions[0].id", "dofus2_en").
			Equal("$.bonus_whitelist", nil).
			Equal("$.bonus_blacklist", nil).
			Equal("$.mentions", nil).
			Present("$.created_at").
			Present("$.updated_at").
			Equal("$.iso_date", false).
			Equal("$.format", "discord").
			End(),
		).
		End()
}

func (suite *AlmanaxTestSuite) Test_CRUD_Create_And_Update() {
	apitest.New().
		Mocks(suite.almBonusMock, suite.discordCheck[0]).
		Handler(Router()).
		Post("/webhooks/almanax").
		JSON(AlmanaxHookPost{
			Callback: "https://discord.com/api/webhooks/123/abc",
			Subscriptions: []string{
				"dofus2_fr",
			},
			Mentions: &map[string][]MentionDTO{
				"loot": {
					MentionDTO{
						DiscordId: 123,
						IsRole:    false,
					},
					MentionDTO{
						DiscordId: 124,
						IsRole:    true,
					},
				},
			},
			Format: "discord",
		}).
		Expect(suite.T()).
		Status(http.StatusCreated).
		End()

	uid, err := testutilGetlastinsertedwebhookid()
	if err != nil {
		suite.T().Fatal(err)
	}

	hook, err := suite.db.GetAlmanaxHook(uid)
	if err != nil {
		suite.T().Fatal(err)
	}

	apitest.New().
		Mocks(suite.almBonusMock).
		Handler(Router()).
		Put("/webhooks/almanax/" + uid.String()).
		JSON(AlmanaxHookPut{
			BonusBlacklist: []string{},
			Subscriptions: []string{
				"dofus2_en",
				"dofus2_fr",
			},
		}).
		Expect(suite.T()).
		Status(http.StatusOK).
		Assert(jsonpath.Chain().
			Present("$.id").
			NotPresent("$.last_fired_at").
			Equal("$.daily_settings.timezone", "Europe/Paris").
			Equal("$.daily_settings.midnight_offset", float64(0)).
			NotPresent("$.callback").
			Equal("$.subscriptions[0].id", "dofus2_en").
			Equal("$.subscriptions[1].id", "dofus2_fr").
			Equal("$.bonus_whitelist", nil).
			Equal("$.bonus_blacklist", nil).
			Present("$.mentions").
			Present("$.created_at").
			NotEqual("$.updated_at", hook.UpdatedAt).
			Equal("$.iso_date", false).
			Equal("$.format", "discord").
			End(),
		).
		End()

	apitest.New().
		Mocks(suite.almBonusMock).
		Handler(Router()).
		Put("/webhooks/almanax/" + uid.String()).
		JSON(AlmanaxHookPut{
			BonusWhitelist: []string{
				"loot",
			},
		}).
		Expect(suite.T()).
		Status(http.StatusOK).
		Assert(jsonpath.Chain().
			Present("$.id").
			NotPresent("$.last_fired_at").
			Equal("$.daily_settings.timezone", "Europe/Paris").
			Equal("$.daily_settings.midnight_offset", float64(0)).
			NotPresent("$.callback").
			Equal("$.subscriptions[0].id", "dofus2_en").
			Equal("$.subscriptions[1].id", "dofus2_fr").
			Equal("$.bonus_whitelist[0]", "loot").
			Equal("$.bonus_blacklist", nil).
			Present("$.mentions").
			Present("$.created_at").
			Present("$.updated_at").
			Equal("$.iso_date", false).
			Equal("$.format", "discord").
			End(),
		).
		End()

	apitest.New().
		Mocks(suite.almBonusMock).
		Handler(Router()).
		Put("/webhooks/almanax/" + uid.String()).
		JSON(AlmanaxHookPut{
			BonusWhitelist: []string{
				"reward-xp",
			},
		}).
		Expect(suite.T()).
		Status(http.StatusBadRequest).
		End()

	apitest.New().
		Mocks(suite.almBonusMock).
		Handler(Router()).
		Put("/webhooks/almanax/" + uid.String()).
		JSON(AlmanaxHookPut{
			BonusBlacklist: []string{
				"rewardbonus",
			},
		}).
		Expect(suite.T()).
		Status(http.StatusOK).
		Assert(jsonpath.Chain().
			Equal("$.bonus_blacklist[0]", "rewardbonus").
			End(),
		).
		End()

	apitest.New().
		Mocks(suite.almBonusMock).
		Handler(Router()).
		Put("/webhooks/almanax/" + uid.String()).
		JSON(AlmanaxHookPut{
			BonusBlacklist: []string{
				"rewardbonus",
			},
			BonusWhitelist: []string{
				"loot",
			},
		}).
		Expect(suite.T()).
		Status(http.StatusBadRequest).
		End()

	apitest.New().
		Mocks(suite.almBonusMock).
		Handler(Router()).
		Put("/webhooks/almanax/" + uid.String()).
		JSON(AlmanaxHookPut{
			BonusBlacklist: []string{
				"unknownBonus",
			},
		}).
		Expect(suite.T()).
		Status(http.StatusBadRequest).
		End()

	putTz1 := "Europe/Berlin1"
	apitest.New().
		Mocks(suite.almBonusMock).
		Handler(Router()).
		Put("/webhooks/almanax/" + uid.String()).
		JSON(AlmanaxHookPut{
			DailySettings: &WebhookDailySettings{
				Timezone: &putTz1,
			},
		}).
		Expect(suite.T()).
		Status(http.StatusBadRequest).
		End()

	putTz1 = "Europe/Paris"
	tzOffset := 2
	apitest.New().
		Mocks(suite.almBonusMock).
		Handler(Router()).
		Put("/webhooks/almanax/" + uid.String()).
		JSON(AlmanaxHookPut{
			DailySettings: &WebhookDailySettings{
				Timezone:       &putTz1,
				MidnightOffset: &tzOffset,
			},
		}).
		Expect(suite.T()).
		Status(http.StatusOK).
		Assert(jsonpath.Chain().
			Equal("$.daily_settings.timezone", "Europe/Paris").
			Equal("$.daily_settings.midnight_offset", float64(2)).
			End(),
		).
		End()

	changeWantsIsoDate := true
	apitest.New().
		Mocks(suite.almBonusMock).
		Handler(Router()).
		Put("/webhooks/almanax/" + uid.String()).
		JSON(AlmanaxHookPut{
			WantsIsoDate: &changeWantsIsoDate,
			Mentions: &map[string][]MentionDTO{
				"loot": {
					{
						DiscordId: 42,
						IsRole:    false,
					},
				},
			},
		}).
		Expect(suite.T()).
		Status(http.StatusOK).
		Assert(jsonpath.Chain().
			Equal("$.iso_date", true).
			Equal("$.mentions.loot[0].discord_id", float64(42)).
			Equal("$.mentions.loot[0].is_role", false).
			End(),
		).
		End()

	changeWantsIsoDate = false
	apitest.New().
		Mocks(suite.almBonusMock).
		Handler(Router()).
		Put("/webhooks/almanax/" + uid.String()).
		JSON(AlmanaxHookPut{
			WantsIsoDate: &changeWantsIsoDate,
		}).
		Expect(suite.T()).
		Status(http.StatusOK).
		Assert(jsonpath.Chain().
			Equal("$.iso_date", false).
			End(),
		).
		End()
}

func (suite *AlmanaxTestSuite) Test_GetFeeds() {
	apitest.New().
		Mocks(suite.almBonusMock, suite.discordCheck[0]).
		Handler(Router()).
		Post("/webhooks/almanax").
		JSON(AlmanaxHookPost{
			Callback: "https://discord.com/api/webhooks/123/abc",
			Subscriptions: []string{
				"dofus2_fr",
			},
			Mentions: &map[string][]MentionDTO{
				"loot": {
					MentionDTO{
						DiscordId: 123,
						IsRole:    false,
					},
					MentionDTO{
						DiscordId: 124,
						IsRole:    true,
					},
				},
			},
			Format: "discord",
		}).
		Expect(suite.T()).
		Status(http.StatusCreated).
		End()

	webhookId, err := testutilGetlastinsertedwebhookid()
	assert.Nil(suite.T(), err)

	feeds, err := suite.db.GetAlmanaxFeeds([]uint64{27})
	assert.Nil(suite.T(), err)

	assert.Len(suite.T(), feeds, 1)
	dofus2Fr := feeds[0]
	hooks, err := suite.db.GetAlmanaxSubsForFeed(dofus2Fr)
	assert.Nil(suite.T(), err)

	assert.Len(suite.T(), hooks, 1)
	assert.Equal(suite.T(), webhookId, hooks[0].Id)
}

func (suite *AlmanaxTestSuite) Test_FilterBonus() {
	actualBonusId := "loot"
	actualBonusName := "Loot"
	assert.False(suite.T(), filterAlmanaxBonusWhiteBlacklist(AlmanaxWebhook{
		BonusWhitelist: []string{
			"loot",
		}},
		dodugo.GetMetaAlmanaxBonuses200ResponseInner{
			Id:   &actualBonusId,
			Name: &actualBonusName,
		}))

	actualBonusId = "rewardxp"
	actualBonusName = "Rewardxp"
	assert.True(suite.T(), filterAlmanaxBonusWhiteBlacklist(AlmanaxWebhook{
		BonusWhitelist: []string{
			"loot",
		}},
		dodugo.GetMetaAlmanaxBonuses200ResponseInner{
			Id:   &actualBonusId,
			Name: &actualBonusName,
		}))

	actualBonusId = "rewardxp"
	actualBonusName = "Rewardxp"
	assert.False(suite.T(), filterAlmanaxBonusWhiteBlacklist(AlmanaxWebhook{
		BonusBlacklist: []string{
			"loot",
		}},
		dodugo.GetMetaAlmanaxBonuses200ResponseInner{
			Id:   &actualBonusId,
			Name: &actualBonusName,
		}))

	actualBonusId = "rewardxp"
	actualBonusName = "Rewardxp"
	assert.True(suite.T(), filterAlmanaxBonusWhiteBlacklist(AlmanaxWebhook{
		BonusBlacklist: []string{
			"rewardxp",
		}},
		dodugo.GetMetaAlmanaxBonuses200ResponseInner{
			Id:   &actualBonusId,
			Name: &actualBonusName,
		}))

	actualBonusId = "loot"
	actualBonusName = "Loot"
	assert.True(suite.T(), filterAlmanaxBonusWhiteBlacklist(AlmanaxWebhook{
		BonusWhitelist: []string{}},
		dodugo.GetMetaAlmanaxBonuses200ResponseInner{
			Id:   &actualBonusId,
			Name: &actualBonusName,
		}))
}

func TestAlmanaxTestSuite(t *testing.T) {
	suite.Run(t, new(AlmanaxTestSuite))
}
