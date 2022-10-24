package main

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/mmcdole/gofeed"
	"github.com/steinfletcher/apitest"
	jsonpath "github.com/steinfletcher/apitest-jsonpath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestFilterMarkdownImages(t *testing.T) {
	images := []string{
		`![image](https://static.ankama.com/ankama/cms/images/273/2022/09/22/1513063.jpg)`,
		`![](https://static.ankama.com/ankama/cms/images/273/2022/09/22/1513063.jpg)`,
	}

	for _, image := range images {
		assert.Equal(t, "", filterMarkdownImageStrings(image))
	}

	noImages := []string{
		`[billet de devblog](https://www.dofus.com/fr/mmorpg/actualites/devblog/billets/1510962-devblog-fusion-serveurs-fusionnes)`,
	}

	for _, noImage := range noImages {
		assert.Equal(t, noImage, filterMarkdownImageStrings(noImage))
	}
}

func TestFilterByBlackWhitelist(t *testing.T) {
	var input []HasIdBlackWhiteList[string]
	var filtered []IHook

	input = []HasIdBlackWhiteList[string]{}
	input = append(input, RssWebhook{
		Id:        uuid.New(),
		Callback:  "https://discord.com/api/webhooks/123/abc",
		Whitelist: []string{"whitelisted"},
	})

	filtered = filterByBlackWhitelist(input, "loot is nice")
	assert.Len(t, filtered, 0)

	input = []HasIdBlackWhiteList[string]{}
	input = append(input, RssWebhook{
		Id:        uuid.New(),
		Callback:  "https://discord.com/api/webhooks/123/abc",
		Whitelist: []string{"loot"},
	})

	filtered = filterByBlackWhitelist(input, "loot is nice")
	assert.Len(t, filtered, 1)
	assert.Equal(t, input[0].GetId().String(), filtered[0].GetId().String())

	input = []HasIdBlackWhiteList[string]{}
	input = append(input, RssWebhook{
		Id:        uuid.New(),
		Callback:  "https://discord.com/api/webhooks/123/abc",
		Blacklist: []string{"loot"},
	})

	filtered = filterByBlackWhitelist(input, "loot is nice")
	assert.Len(t, filtered, 0)

	input = []HasIdBlackWhiteList[string]{}
	input = append(input, RssWebhook{
		Id:        uuid.New(),
		Callback:  "https://discord.com/api/webhooks/123/abc",
		Blacklist: []string{"hello"},
	})

	filtered = filterByBlackWhitelist(input, "loot is nice")
	assert.Len(t, filtered, 1)
	assert.Equal(t, input[0].GetId().String(), filtered[0].GetId().String())

	input = []HasIdBlackWhiteList[string]{}
	input = append(input, RssWebhook{
		Id:        uuid.New(),
		Callback:  "https://discord.com/api/webhooks/123/abc",
		Whitelist: []string{"loot"},
		Blacklist: []string{"hello"},
	})

	filtered = filterByBlackWhitelist(input, "hello is nice")
	assert.Len(t, filtered, 0)

	input = []HasIdBlackWhiteList[string]{}
	input = append(input, RssWebhook{
		Id:        uuid.New(),
		Callback:  "https://discord.com/api/webhooks/123/abc",
		Whitelist: []string{"loot"},
		Blacklist: []string{"hello"},
	})

	filtered = filterByBlackWhitelist(input, "hello is nice and loot is nice")
	assert.Len(t, filtered, 1)
	assert.Equal(t, input[0].GetId().String(), filtered[0].GetId().String())

	input = []HasIdBlackWhiteList[string]{}
	input = append(input, RssWebhook{
		Id:        uuid.New(),
		Callback:  "https://discord.com/api/webhooks/123/abc",
		Whitelist: []string{"loot"},
		Blacklist: []string{"hello"},
	})

	filtered = filterByBlackWhitelist(input, "dofus")
	assert.Len(t, filtered, 1)
	assert.Equal(t, input[0].GetId().String(), filtered[0].GetId().String())

	input = []HasIdBlackWhiteList[string]{}
	input = append(input, RssWebhook{
		Id:       uuid.New(),
		Callback: "https://discord.com/api/webhooks/123/abc",
	})

	filtered = filterByBlackWhitelist(input, "dofus")
	assert.Len(t, filtered, 1)
	assert.Equal(t, input[0].GetId().String(), filtered[0].GetId().String())

	input = []HasIdBlackWhiteList[string]{}
	input = append(input, RssWebhook{
		Id:        uuid.New(),
		Callback:  "https://discord.com/api/webhooks/123/abc",
		Whitelist: []string{"loot"},
		Blacklist: []string{"hello"},
	})

	filtered = filterByBlackWhitelist(input, "loot")
	assert.Len(t, filtered, 1)
	assert.Equal(t, input[0].GetId().String(), filtered[0].GetId().String())
}

func TestFindImageUrl(t *testing.T) {
	file, err := os.ReadFile("testdata/fusionNewsItem.xml")
	fp := gofeed.NewParser()
	rssFeed, err := fp.ParseString(string(file))
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, "https://static.ankama.com/ankama/cms/images/273/2022/09/22/1513063.jpg", findImageUrl(rssFeed.Items[0].Description))
}

func TestShortenAndRenderDescription(t *testing.T) {
	file, err := os.ReadFile("testdata/fusionNewsItem.xml")
	fp := gofeed.NewParser()
	rssFeed, err := fp.ParseString(string(file))
	if err != nil {
		t.Error(err)
	}

	markdown, err := shortenAndRenderDescription(rssFeed.Items[0].Description, 200)
	assert.Nil(t, err)
	assert.True(t, len(markdown) < 200+10+4) // finish word and ' ...'
	assert.Equal(t, " ...", markdown[len(markdown)-4:])
}

type RssTestSuite struct {
	suite.Suite
	db           Repository
	sut          *httptest.Server
	discordCheck []*apitest.Mock
}

func (suite *RssTestSuite) Test_Feeds_Db() {
	feeds, err := suite.db.GetRssFeeds([]uint64{})
	assert.Nil(suite.T(), err)
	assert.Len(suite.T(), feeds, 12)
	assert.NotEqual(suite.T(), feeds[0].GetFeedName(), "")
}

func (suite *RssTestSuite) SetupSuite() {
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
}

func (suite *RssTestSuite) TearDownSuite() {
	suite.db.conn.Close()
}

func (suite *RssTestSuite) SetupTest() {
}

func (suite *RssTestSuite) TearDownTest() {
	if err := testutilCleartables(); err != nil {
		suite.T().Fatal(err)
	}
}

func (suite *RssTestSuite) Test_Feeds() {
	apitest.New().
		Handler(Router()).
		Get("/meta/webhooks/rss").
		Expect(suite.T()).
		Status(http.StatusOK).
		Assert(jsonpath.Chain().
			Contains("$.subscriptions", "dofus2-fr-official-news").
			Contains("$.subscriptions", "dofus2-en-official-news").
			Contains("$.subscriptions", "dofus2-es-official-news").
			Contains("$.subscriptions", "dofus2-pt-official-news").
			Contains("$.subscriptions", "dofus2-fr-official-changelog").
			Contains("$.subscriptions", "dofus2-en-official-changelog").
			Contains("$.subscriptions", "dofus2-es-official-changelog").
			Contains("$.subscriptions", "dofus2-pt-official-changelog").
			Contains("$.subscriptions", "dofus2-fr-official-devblog").
			Contains("$.subscriptions", "dofus2-es-official-devblog").
			Contains("$.subscriptions", "dofus2-en-official-devblog").
			Contains("$.subscriptions", "dofus2-pt-official-devblog").
			End(),
		).
		End()
}

func (suite *RssTestSuite) Test_GetFeeds() {
	apitest.New().
		Mocks(suite.discordCheck[0]).
		Handler(Router()).
		Post("/webhooks/rss").
		JSON(SocialHookCreate{
			Callback: "https://discord.com/api/webhooks/123/abc",
			Subscriptions: []string{
				"dofus2-fr-official-news",
			},
			Format: "discord",
		}).
		Expect(suite.T()).
		Status(http.StatusCreated).
		End()

	id, err := testutilGetlastinsertedwebhookid()
	assert.Nil(suite.T(), err)

	feeds, err := suite.db.GetRssFeeds([]uint64{1})
	assert.Nil(suite.T(), err)

	assert.Len(suite.T(), feeds, 1)
	frOfficialNews := feeds[0]

	subbedFeeds, err := suite.db.GetRSSSubsForFeed(frOfficialNews)
	assert.Nil(suite.T(), err)
	assert.Len(suite.T(), subbedFeeds, 1)
	assert.Equal(suite.T(), subbedFeeds[0].GetId(), id)
}

func (suite *RssTestSuite) Test_CRUD_Create() {
	apitest.New().
		Mocks(suite.discordCheck[0]).
		Handler(Router()).
		Post("/webhooks/rss").
		JSON(SocialHookCreate{
			Callback: "https://discord.com/api/webhooks/123/abc",
			Subscriptions: []string{
				"dofus2-fr-official-news",
			},
			Format: "discord",
		}).
		Expect(suite.T()).
		Status(http.StatusCreated).
		Assert(jsonpath.Chain().
			Present("$.id").
			Contains("$.subscriptions", "dofus2-fr-official-news").
			End(),
		).
		End()

	apitest.New().
		Mocks(suite.discordCheck[1]).
		Handler(Router()).
		Post("/webhooks/rss").
		JSON(SocialHookCreate{
			Callback: "https://discord.com/api/webhooks/123/abc1", // callback ok but no subscriptions
			Format:   "discord",
		}).
		Expect(suite.T()).
		Status(http.StatusBadRequest).
		End()

	apitest.New().
		Mocks(suite.discordCheck[1]).
		Handler(Router()).
		Post("/webhooks/rss").
		JSON(SocialHookCreate{
			Callback: "https://discord.com/api/webhooks/123/abc3",
			Subscriptions: []string{
				"dofus2-fr-official-news",
			}, // no format
		}).
		Expect(suite.T()).
		Status(http.StatusBadRequest).
		End()

	apitest.New().
		Mocks(suite.discordCheck[0]).
		Handler(Router()).
		Post("/webhooks/rss").
		JSON(SocialHookCreate{
			Callback: "https://discord.com/api/webhooks/123/abc", // double callback
			Subscriptions: []string{
				"dofus2-fr-official-news",
			},
			Format: "discord",
		}).
		Expect(suite.T()).
		Status(http.StatusConflict).
		End()

	apitest.New().
		Mocks(suite.discordCheck[3]).
		Handler(Router()).
		Post("/webhooks/rss").
		JSON(SocialHookCreate{
			Callback: "https://discord.com/api/webhooks/123/abc3",
			Subscriptions: []string{
				"fr-official--news", // invalid name
			},
			Format: "discord",
		}).
		Expect(suite.T()).
		Status(http.StatusBadRequest).
		End()

	apitest.New().
		Handler(Router()).
		Post("/webhooks/rss").
		Expect(suite.T()).
		Status(http.StatusBadRequest). // no body
		End()

	apitest.New().
		Mocks(suite.discordCheck[5]).
		Handler(Router()).
		Post("/webhooks/rss").
		JSON(SocialHookCreate{
			Callback: "https://discord.com/api/webhooks/123/abc5",
			Subscriptions: []string{
				"dofus2-fr-official-news",
			},
			Whitelist: []string{
				"dofus touch",
			},
			Blacklist: []string{
				"ankama",
			},
			Format: "discord",
		}).
		Expect(suite.T()).
		Status(http.StatusCreated).
		Assert(jsonpath.Chain().
			Present("$.id").
			Contains("$.subscriptions", "dofus2-fr-official-news").
			Contains("$.whitelist", "dofus touch").
			Contains("$.blacklist", "ankama").
			Present("$.created_at").
			NotEqual("$.updated_at", nil).
			NotPresent("$.last_fired_at").
			NotPresent("$.callback").
			Equal("$.preview_length", float64(2000)).
			End(),
		).
		End()

	apitest.New().
		Mocks(suite.discordCheck[6]).
		Handler(Router()).
		Post("/webhooks/rss").
		JSON(SocialHookCreate{
			Callback: "https://discord.com/api/webhooks/123/abc6",
			Subscriptions: []string{
				"dofus2-fr-official-news",
			},
			Whitelist: []string{
				"dofus touch",
			},
			Format: "discord",
		}).
		Expect(suite.T()).
		Status(http.StatusCreated).
		Assert(jsonpath.Chain().
			Present("$.id").
			Contains("$.subscriptions", "dofus2-fr-official-news").
			Contains("$.whitelist", "dofus touch").
			Present("$.created_at").
			NotEqual("$.updated_at", nil).
			NotPresent("$.last_fired_at").
			NotPresent("$.callback").
			Equal("$.preview_length", float64(2000)).
			End(),
		).
		End()
}

func (suite *RssTestSuite) Test_CRUD_Create_And_Get() {
	apitest.New().
		Mocks(suite.discordCheck[0]).
		Handler(Router()).
		Post("/webhooks/rss").
		JSON(SocialHookCreate{
			Callback: "https://discord.com/api/webhooks/123/abc",
			Subscriptions: []string{
				"dofus2-fr-official-news",
			},
			Format: "discord",
		}).
		Expect(suite.T()).
		Status(http.StatusCreated).
		Assert(jsonpath.Chain().
			Present("$.id").
			Contains("$.subscriptions", "dofus2-fr-official-news").
			End(),
		).
		End()

	id, err := testutilGetlastinsertedwebhookid()
	assert.Nil(suite.T(), err)

	apitest.New().
		Handler(Router()).
		Get("/webhooks/rss/" + id.String()).
		Expect(suite.T()).
		Status(http.StatusOK).
		Assert(jsonpath.Chain().
			Present("$.id").
			Contains("$.subscriptions", "dofus2-fr-official-news").
			Present("$.created_at").
			Present("$.updated_at").
			NotPresent("$.last_fired_at").
			End(),
		).
		End()
}

func (suite *RssTestSuite) Test_CRUD_Delete() {
	apitest.New().
		Mocks(suite.discordCheck[0]).
		Handler(Router()).
		Post("/webhooks/rss").
		JSON(SocialHookCreate{
			Callback: "https://discord.com/api/webhooks/123/abc",
			Subscriptions: []string{
				"dofus2-fr-official-news",
			},
			Format: "discord",
		}).
		Expect(suite.T()).
		Status(http.StatusCreated).
		Assert(jsonpath.Chain().
			Present("$.id").
			Contains("$.subscriptions", "dofus2-fr-official-news").
			End(),
		).
		End()

	id, err := testutilGetlastinsertedwebhookid()
	assert.Nil(suite.T(), err)

	apitest.New().
		Handler(Router()).
		Delete("/webhooks/rss/" + id.String()).
		Expect(suite.T()).
		Status(http.StatusNoContent).
		End()

	apitest.New().
		Handler(Router()).
		Get("/webhooks/rss/" + id.String()).
		Expect(suite.T()).
		Status(http.StatusNotFound).
		End()
}

func (suite *RssTestSuite) Test_CRUD_Create_And_Update() {
	apitest.New().
		Mocks(suite.discordCheck[0]).
		Handler(Router()).
		Post("/webhooks/rss").
		JSON(SocialHookCreate{
			Callback: "https://discord.com/api/webhooks/123/abc",
			Subscriptions: []string{
				"dofus2-fr-official-news",
			},
			Format: "discord",
		}).
		Expect(suite.T()).
		Status(http.StatusCreated).
		Assert(jsonpath.Chain().
			Present("$.id").
			Contains("$.subscriptions", "dofus2-fr-official-news").
			End(),
		).
		End()

	id, err := testutilGetlastinsertedwebhookid()
	assert.Nil(suite.T(), err)

	hook, err := suite.db.GetSocialHook(RSSWebhookType, id)
	assert.Nil(suite.T(), err)

	apitest.New().
		Handler(Router()).
		Put("/webhooks/rss/" + id.String()).
		JSON(SocialWebhookPut{
			Subscriptions: []string{
				"dofus2-en-official-news",
			},
			Whitelist: []string{
				"dofus",
			},
			Blacklist: []string{
				"ankama",
			},
		}).
		Expect(suite.T()).
		Status(http.StatusOK).
		Assert(jsonpath.Chain().
			Present("$.id").
			Contains("$.subscriptions", "dofus2-en-official-news").
			Contains("$.whitelist", "dofus").
			Contains("$.blacklist", "ankama").
			Present("$.created_at").
			NotEqual("$.updated_at", hook.GetUpdatedAt()).
			NotPresent("$.last_fired_at").
			End(),
		).
		End()

	apitest.New().
		Handler(Router()).
		Put("/webhooks/rss/" + id.String()).
		JSON(SocialWebhookPut{
			Subscriptions: []string{
				"ede-official-news", // invalid subscription
			},
			Whitelist: []string{
				"dofus",
			},
			Blacklist: []string{
				"ankama",
			},
		}).
		Expect(suite.T()).
		Status(http.StatusBadRequest).
		End()
}

func TestRssTestSuite(t *testing.T) {
	suite.Run(t, new(RssTestSuite))
}
