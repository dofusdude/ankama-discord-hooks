package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/steinfletcher/apitest"
	jsonpath "github.com/steinfletcher/apitest-jsonpath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

/*func TestTwitterFeedParsingInReply(t *testing.T) {
	latestTweetsEnLastInReply := `{
  "data": [
    {
      "text": "@Angeltt_tt \uD83D\uDC40",
      "in_reply_to_user_id": "1386997635152760839",
      "edit_history_tweet_ids": [
        "1575826132649263105"
      ],
      "created_at": "2022-09-30T12:33:22.000Z",
      "id": "1575826132649263105",
      "author_id": "83587596"
    }
  ],
  "includes": {
    "users": [
      {
        "username": "DOFUS_EN",
        "id": "83587596",
        "name": "DOFUS",
        "profile_image_url": "https://pbs.twimg.com/profile_images/1535256198752227333/uQVae-Ac_normal.jpg"
      }
    ],
    "media": [
      {
        "media_key": "13_1575493200847831041",
        "type": "video"
      },
      {
        "media_key": "3_1575138291984252929",
        "type": "photo",
        "url": "https://pbs.twimg.com/media/FdwD7l3WQAEefi_.jpg"
      },
      {
        "media_key": "3_1575123287826391040",
        "type": "photo",
        "url": "https://pbs.twimg.com/media/Fdv2SPBWYAAUNCe.jpg"
      }
    ]
  },
  "meta": {
    "result_count": 5,
    "newest_id": "1575826132649263105",
    "oldest_id": "1575123290166804481"
  }
}`

	inReplyMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := fmt.Fprintf(w, latestTweetsEnLastInReply)
		assert.Nil(t, err)
	}))
	defer inReplyMock.Close()

	tweets, err := getLatestTweets(1, time.Now().Add(-time.Minute), inReplyMock.URL)
	assert.Nil(t, err)
	assert.Len(t, tweets, 0)
}*/

/*func TestTwitterFeedParsingNoReply(t *testing.T) {
	latestTweetsEnFirstNoReply := `{
  "data": [
    {
      "text": "#DOFUS \uD83D\uDE36 https://t.co/Lock9a3GmX",
      "attachments": {
        "media_keys": [
          "13_1575493200847831041"
        ]
      },
      "edit_history_tweet_ids": [
        "1575493618651246598"
      ],
      "created_at": "2022-09-29T14:32:04.000Z",
      "id": "1575493618651246598",
      "author_id": "83587596"
    },
    {
      "text": "@ViperOnEcho We will leave clues over time for you to discover the name of all future servers ;)",
      "in_reply_to_user_id": "1397186706634416133",
      "edit_history_tweet_ids": [
        "1575459146920591360"
      ],
      "created_at": "2022-09-29T12:15:05.000Z",
      "id": "1575459146920591360",
      "author_id": "83587596"
    },
    {
      "text": "\uD83E\uDD29 Two new friends are making their way in the #AnkamaShop today! \uD83E\uDDF8 \nGood for cuddling and to channel all of your adorable hellspawn's demonic energy \uD83D\uDE08\n\n\uD83D\uDED2 Check it out now: https://t.co/GDF3lmpm10 https://t.co/Td8TmKG5UL",
      "attachments": {
        "media_keys": [
          "3_1575138291984252929"
        ]
      },
      "edit_history_tweet_ids": [
        "1575138294945521664"
      ],
      "created_at": "2022-09-28T15:00:08.000Z",
      "id": "1575138294945521664",
      "author_id": "83587596"
    }
  ],
  "includes": {
    "users": [
      {
        "username": "DOFUS_EN",
        "id": "83587596",
        "name": "DOFUS",
        "profile_image_url": "https://pbs.twimg.com/profile_images/1535256198752227333/uQVae-Ac_normal.jpg"
      }
    ],
    "media": [
      {
        "media_key": "13_1575493200847831041",
        "type": "video"
      },
      {
        "media_key": "3_1575138291984252929",
        "type": "photo",
        "url": "https://pbs.twimg.com/media/FdwD7l3WQAEefi_.jpg"
      },
      {
        "media_key": "3_1575123287826391040",
        "type": "photo",
        "url": "https://pbs.twimg.com/media/Fdv2SPBWYAAUNCe.jpg"
      }
    ]
  },
  "meta": {
    "result_count": 5,
    "newest_id": "1575826132649263105",
    "oldest_id": "1575123290166804481"
  }
}`

	tweetCreated, err := time.Parse("2006-01-02T15:04:05.000Z", "2022-09-29T14:32:04.000Z")
	assert.Nil(t, err)

	noReplyMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := fmt.Fprintf(w, latestTweetsEnFirstNoReply)
		assert.Nil(t, err)
	}))
	defer noReplyMock.Close()

	tweets, err := getLatestTweets(1, tweetCreated.Add(-time.Minute), noReplyMock.URL)
	assert.Nil(t, err)
	assert.True(t, len(tweets) != 0)
}*/

type TwitterTestSuite struct {
	suite.Suite
	db           Repository
	sut          *httptest.Server
	discordCheck []*apitest.Mock
}

func (suite *TwitterTestSuite) SetupSuite() {
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

func (suite *TwitterTestSuite) TearDownSuite() {
	suite.db.conn.Close()
}

func (suite *TwitterTestSuite) SetupTest() {
}

func (suite *TwitterTestSuite) TearDownTest() {
	if err := testutilCleartables(); err != nil {
		suite.T().Fatal(err)
	}
}

func (suite *TwitterTestSuite) Test_Feeds_Db() {
	feeds, err := suite.db.GetTwitterFeeds([]uint64{})
	assert.Nil(suite.T(), err)
	assert.Len(suite.T(), feeds, 13)
	assert.NotEqual(suite.T(), feeds[0].GetFeedName(), "")
}

func (suite *TwitterTestSuite) Test_Feeds() {
	apitest.New().
		Handler(Router()).
		Get("/meta/webhooks/twitter").
		Expect(suite.T()).
		Status(http.StatusOK).
		Assert(jsonpath.Chain().
			Contains("$.subscriptions", "DOFUS_EN").
			Contains("$.subscriptions", "ES_DOFUS").
			Contains("$.subscriptions", "DOFUSfr").
			Contains("$.subscriptions", "DOFUS_PTBR").
			Contains("$.subscriptions", "DOFUSTouch_EN").
			Contains("$.subscriptions", "AnkamaGames").
			Contains("$.subscriptions", "DOFUSTouch").
			Contains("$.subscriptions", "DOFUSTouch_ES").
			Contains("$.subscriptions", "KTA_En").
			Contains("$.subscriptions", "KTA_fr").
			Contains("$.subscriptions", "DPLNofficiel").
			Contains("$.subscriptions", "dofusbook_modo").
			Contains("$.subscriptions", "JOL_Dofus").
			End(),
		).
		End()
}

func (suite *TwitterTestSuite) Test_GetFeeds() {
	apitest.New().
		Mocks(suite.discordCheck[0]).
		Handler(Router()).
		Post("/webhooks/twitter").
		JSON(SocialHookCreate{
			Callback: "https://discord.com/api/webhooks/123/abc",
			Subscriptions: []string{
				"DOFUS_EN",
			},
			Format: "discord",
		}).
		Expect(suite.T()).
		Status(http.StatusCreated).
		Assert(jsonpath.Chain().
			Present("$.id").
			Contains("$.subscriptions", "DOFUS_EN").
			End(),
		).
		End()

	id, err := testutilGetlastinsertedwebhookid()
	assert.Nil(suite.T(), err)

	feeds, err := suite.db.GetTwitterFeeds([]uint64{13})
	assert.Nil(suite.T(), err)

	assert.Len(suite.T(), feeds, 1)
	dofusEn := feeds[0]

	subbedFeeds, err := suite.db.GetTwitterSubsForFeed(dofusEn)
	assert.Nil(suite.T(), err)
	assert.Len(suite.T(), subbedFeeds, 1)
	assert.Equal(suite.T(), subbedFeeds[0].GetId(), id)
}

func (suite *TwitterTestSuite) Test_CRUD_Create() {
	apitest.New().
		Mocks(suite.discordCheck[0]).
		Handler(Router()).
		Post("/webhooks/twitter").
		JSON(SocialHookCreate{
			Callback: "https://discord.com/api/webhooks/123/abc",
			Subscriptions: []string{
				"DOFUS_EN",
			},
			Format: "discord",
		}).
		Expect(suite.T()).
		Status(http.StatusCreated).
		Assert(jsonpath.Chain().
			Present("$.id").
			Contains("$.subscriptions", "DOFUS_EN").
			End(),
		).
		End()

	apitest.New().
		Mocks(suite.discordCheck[1]).
		Handler(Router()).
		Post("/webhooks/twitter").
		JSON(SocialHookCreate{
			Callback: "https://discord.com/api/webhooks/123/abc1", // callback ok but no subscriptions
			Format:   "discord",
		}).
		Expect(suite.T()).
		Status(http.StatusBadRequest).
		End()

	apitest.New().
		Mocks(suite.discordCheck[0]).
		Handler(Router()).
		Post("/webhooks/twitter").
		JSON(SocialHookCreate{
			Callback: "https://discord.com/api/webhooks/123/abc3",
			Subscriptions: []string{
				"DOFUS_EN",
			}, // missing format
		}).
		Expect(suite.T()).
		Status(http.StatusBadRequest).
		End()

	apitest.New().
		Mocks(suite.discordCheck[0]).
		Handler(Router()).
		Post("/webhooks/twitter").
		JSON(SocialHookCreate{
			Callback: "https://discord.com/api/webhooks/123/abc", // double callback
			Subscriptions: []string{
				"DOFUSfr",
			},
			Format: "discord",
		}).
		Expect(suite.T()).
		Status(http.StatusConflict).
		End()

	apitest.New().
		Mocks(suite.discordCheck[3]).
		Handler(Router()).
		Post("/webhooks/twitter").
		JSON(SocialHookCreate{
			Callback: "https://discord.com/api/webhooks/123/abc3",
			Subscriptions: []string{
				"DOFUS_fr", // invalid name
			},
			Format: "discord",
		}).
		Expect(suite.T()).
		Status(http.StatusBadRequest).
		End()

	apitest.New().
		Handler(Router()).
		Post("/webhooks/twitter").
		Expect(suite.T()).
		Status(http.StatusBadRequest). // no body
		End()

	apitest.New().
		Mocks(suite.discordCheck[5]).
		Handler(Router()).
		Post("/webhooks/twitter").
		JSON(SocialHookCreate{
			Callback: "https://discord.com/api/webhooks/123/abc5",
			Subscriptions: []string{
				"DOFUS_EN",
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
			Contains("$.subscriptions", "DOFUS_EN").
			Contains("$.whitelist", "dofus touch").
			Contains("$.blacklist", "ankama").
			Present("$.created_at").
			NotEqual("$.updated_at", nil).
			NotPresent("$.last_fired_at").
			NotPresent("$.callback").
			Equal("$.preview_length", float64(280)).
			End(),
		).
		End()

	apitest.New().
		Mocks(suite.discordCheck[6]).
		Handler(Router()).
		Post("/webhooks/twitter").
		JSON(SocialHookCreate{
			Callback: "https://discord.com/api/webhooks/123/abc6",
			Subscriptions: []string{
				"DOFUS_EN",
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
			Contains("$.subscriptions", "DOFUS_EN").
			Contains("$.whitelist", "dofus touch").
			Present("$.created_at").
			NotEqual("$.updated_at", nil).
			NotPresent("$.last_fired_at").
			NotPresent("$.callback").
			Equal("$.preview_length", float64(280)).
			End(),
		).
		End()
}

func (suite *TwitterTestSuite) Test_CRUD_Create_And_Get() {
	apitest.New().
		Mocks(suite.discordCheck[0]).
		Handler(Router()).
		Post("/webhooks/twitter").
		JSON(SocialHookCreate{
			Callback: "https://discord.com/api/webhooks/123/abc",
			Subscriptions: []string{
				"DOFUS_EN",
			},
			Format: "discord",
		}).
		Expect(suite.T()).
		Status(http.StatusCreated).
		Assert(jsonpath.Chain().
			Present("$.id").
			Contains("$.subscriptions", "DOFUS_EN").
			End(),
		).
		End()

	id, err := testutilGetlastinsertedwebhookid()
	assert.Nil(suite.T(), err)

	apitest.New().
		Handler(Router()).
		Get("/webhooks/twitter/" + id.String()).
		Expect(suite.T()).
		Status(http.StatusOK).
		Assert(jsonpath.Chain().
			Present("$.id").
			Contains("$.subscriptions", "DOFUS_EN").
			Present("$.created_at").
			Present("$.updated_at").
			NotPresent("$.last_fired_at").
			End(),
		).
		End()
}

func (suite *TwitterTestSuite) Test_CRUD_Delete() {
	apitest.New().
		Mocks(suite.discordCheck[0]).
		Handler(Router()).
		Post("/webhooks/twitter").
		JSON(SocialHookCreate{
			Callback: "https://discord.com/api/webhooks/123/abc",
			Subscriptions: []string{
				"DOFUS_EN",
			},
			Format: "discord",
		}).
		Expect(suite.T()).
		Status(http.StatusCreated).
		Assert(jsonpath.Chain().
			Present("$.id").
			Contains("$.subscriptions", "DOFUS_EN").
			End(),
		).
		End()

	id, err := testutilGetlastinsertedwebhookid()
	assert.Nil(suite.T(), err)

	apitest.New().
		Handler(Router()).
		Delete("/webhooks/twitter/" + id.String()).
		Expect(suite.T()).
		Status(http.StatusNoContent).
		End()

	apitest.New().
		Handler(Router()).
		Get("/webhooks/twitter/" + id.String()).
		Expect(suite.T()).
		Status(http.StatusNotFound).
		End()
}

func (suite *TwitterTestSuite) Test_CRUD_Create_And_Update() {
	apitest.New().
		Mocks(suite.discordCheck[0]).
		Handler(Router()).
		Post("/webhooks/twitter").
		JSON(SocialHookCreate{
			Callback: "https://discord.com/api/webhooks/123/abc",
			Subscriptions: []string{
				"DOFUS_EN",
			},
			Format: "discord",
		}).
		Expect(suite.T()).
		Status(http.StatusCreated).
		Assert(jsonpath.Chain().
			Present("$.id").
			Contains("$.subscriptions", "DOFUS_EN").
			End(),
		).
		End()

	id, err := testutilGetlastinsertedwebhookid()
	assert.Nil(suite.T(), err)

	hook, err := suite.db.GetSocialHook(TwitterWebhookType, id)
	assert.Nil(suite.T(), err)

	apitest.New().
		Handler(Router()).
		Put("/webhooks/twitter/" + id.String()).
		JSON(SocialWebhookPut{
			Subscriptions: []string{
				"DOFUSfr",
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
			Contains("$.subscriptions", "DOFUSfr").
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
		Put("/webhooks/twitter/" + id.String()).
		JSON(SocialWebhookPut{
			Subscriptions: []string{
				"DOFUS_fr", // invalid subscription
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

func TestTwitterTestSuite(t *testing.T) {
	suite.Run(t, new(TwitterTestSuite))
}
