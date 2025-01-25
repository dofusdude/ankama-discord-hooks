package main

import (
	"time"

	"github.com/dofusdude/dodugo"
	"github.com/google/uuid"
	"github.com/mmcdole/gofeed"
)

const (
	TwitterWebhookType string = "twitter"
	RSSWebhookType            = "rss"
	AlmanaxWebhookType        = "almanax"
)

type RssState struct {
	LastHash uint64
}

type RssSend struct {
	Item     gofeed.Item
	Webhooks []IHook
	Feed     IFeed
}

type TwitterState struct {
	LastCheck time.Time
}

type TwitterSend struct {
	Tweet    Tweet
	Webhooks []IHook
}

type PreparedHook struct {
	Callback string
	Body     string
}

type SendCallbackReturn struct {
	Callback string
	Ok       bool
}

type AlmanaxSend struct {
	Feed            AlmanaxFeed
	BuildInfo       AlmanaxHookBuildInfo
	Webhooks        []IHook
	OnlyPreMentions []bool
	IntervalType    []string
	TickTime        time.Time
}

type ApiUserTweetResult struct {
	Data     []ApiUserTweet       `json:"data"`
	Includes ApiUserTweetIncludes `json:"includes"`
	Meta     ApiUserTweetMeta     `json:"meta"`
}

type ApiUserTweetMeta struct {
	ResultCount int `json:"result_count"`
}

type ApiUser struct {
	Id              string `json:"id"`
	Name            string `json:"name"`
	Username        string `json:"username"`
	ProfileImageURL string `json:"profile_image_url"`
}

type ApiUserTweetMedia struct {
	Type     string `json:"type"`
	Url      string `json:"url"`
	MediaKey string `json:"media_key"`
}

type ApiUserTweetIncludes struct {
	Media []ApiUserTweetMedia `json:"media"`
	Users []ApiUser           `json:"users"`
}

type TweetAttachments struct {
	MediaKeys []string `json:"media_keys"`
}

type ApiUserTweet struct {
	AuthorId    string           `json:"author_id"`
	Id          string           `json:"id"`
	Text        string           `json:"text"`
	CreatedAt   time.Time        `json:"created_at"`
	Attachments TweetAttachments `json:"attachments"`
	InReplyTo   *string          `json:"in_reply_to_user_id"`
}

type Tweet struct {
	Author      ApiUser   `json:"author"`
	Text        string    `json:"text"`
	CreatedAt   time.Time `json:"created_at"`
	Attachments []string  `json:"attachments"`
	IsNew       bool      `json:"is_new"`
}

type DiscordImage struct {
	Url string `json:"url"`
}

type DiscordEmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

type DiscordEmbed struct {
	Title       *string             `json:"title"`
	Color       int                 `json:"color"`
	Image       *DiscordImage       `json:"image"`
	Thumbnail   *DiscordImage       `json:"thumbnail"`
	Fields      []DiscordEmbedField `json:"fields"`
	Url         *string             `json:"url"`
	Description *string             `json:"description"`
}

type DiscordWebhook struct {
	Content     *string        `json:"content"`
	Embeds      []DiscordEmbed `json:"embeds"`
	Username    string         `json:"username"`
	AvatarUrl   string         `json:"avatar_url"`
	Attachments []string       `json:"attachments"`
}

type SocialWebhookPut struct {
	Whitelist     []string `json:"whitelist"`
	Blacklist     []string `json:"blacklist"`
	Subscriptions []string `json:"subscriptions"`
	PreviewLength *int     `json:"preview_length"`
}

type WebhookJobs struct {
	Jobs []WebhookJob `json:"jobs"`
}

type WebhookJob struct {
	Url  string `json:"url"`
	Body string `json:"json_body"`
}

type HookMeta struct {
	Subscriptions []string `json:"subscriptions"`
}

type DailySettings struct {
	Timezone       string `json:"timezone"`
	MidnightOffset int    `json:"midnight_offset"`
}

type AlmanaxHookDTO struct {
	Id             uuid.UUID                `json:"id"`
	DailySettings  DailySettings            `json:"daily_settings"`
	LastFiredAt    *time.Time               `json:"last_fired_at"`
	Subscriptions  []Subscription           `json:"subscriptions"`
	BonusWhitelist []string                 `json:"bonus_whitelist"`
	BonusBlacklist []string                 `json:"bonus_blacklist"`
	Format         string                   `json:"format"`
	WantsIsoDate   bool                     `json:"iso_date"`
	Intervals      []string                 `json:"intervals"`
	WeeklyWeekday  *string                  `json:"weekly_weekday"`
	Mentions       *map[string][]MentionDTO `json:"mentions"`
	CreatedAt      time.Time                `json:"created_at"`
	UpdatedAt      time.Time                `json:"updated_at"`
}

type AlmanaxHookPost struct {
	BonusWhitelist []string                 `json:"bonus_whitelist"`
	BonusBlacklist []string                 `json:"bonus_blacklist"`
	DailySettings  *WebhookDailySettings    `json:"daily_settings"`
	Callback       string                   `json:"callback"`
	Subscriptions  []string                 `json:"subscriptions"`
	WantsIsoDate   *bool                    `json:"iso_date"`
	Format         string                   `json:"format"`
	Mentions       *map[string][]MentionDTO `json:"mentions"`
	Intervals      []string                 `json:"intervals"`
	WeeklyWeekday  *string                  `json:"weekly_weekday"`
}

type AlmanaxHookBuildInfo struct {
	almData      map[string]dodugo.Almanax
	translations map[string]map[string]string
}

type IFeed interface {
	GetId() uint64
	GetTwitterId() uint64
	GetRSSUrl() string
	GetAlmanaxFeed() string
	GetType() string
	GetFeedName() string
}

type IHook interface {
	GetId() uuid.UUID
	GetCallback() string
	GetPreviewLength() int
	IsWantIsoDate() bool
	GetTimezone() string
	GetMentions() *map[string][]MentionDTO
}

type HasIdBlackWhiteList[T any] interface {
	IHook
	GetBlacklist() []T
	GetWhitelist() []T
	GetType() string
}

type TwitterFeed struct {
	Id              uint64
	TwitterId       uint64
	IsOfficial      bool
	HumanReadableId string
	CreatedAt       time.Time
}

func (f TwitterFeed) GetId() uint64 {
	return f.Id
}

func (f TwitterFeed) GetTwitterId() uint64 {
	return f.TwitterId
}

func (f TwitterFeed) GetRSSUrl() string {
	return ""
}

func (f TwitterFeed) GetAlmanaxFeed() string {
	return ""
}

func (f TwitterFeed) GetType() string {
	return TwitterWebhookType
}

func (f TwitterFeed) GetFeedName() string {
	return f.HumanReadableId
}

type RssFeed struct {
	Id            uint64
	Url           string
	ApiReadableId string
	IsOfficial    bool
	CreatedAt     time.Time
}

func (f RssFeed) GetId() uint64 {
	return f.Id
}

func (f RssFeed) GetTwitterId() uint64 {
	return 0
}

func (f RssFeed) GetRSSUrl() string {
	return f.Url
}

func (f RssFeed) GetAlmanaxFeed() string {
	return ""
}

func (f RssFeed) GetType() string {
	return RSSWebhookType
}

func (f RssFeed) GetFeedName() string {
	return f.ApiReadableId
}

type AlmanaxFeed struct {
	Id              uint64
	HumanReadableId string
	Language        string
	CreatedAt       time.Time
}

func (f AlmanaxFeed) GetId() uint64 {
	return f.Id
}

func (f AlmanaxFeed) GetTwitterId() uint64 {
	return 0
}

func (f AlmanaxFeed) GetRSSUrl() string {
	return ""
}

func (f AlmanaxFeed) GetFeedName() string {
	return f.HumanReadableId
}

func (f AlmanaxFeed) GetAlmanaxFeed() string {
	return f.HumanReadableId
}

func (f AlmanaxFeed) GetType() string {
	return AlmanaxWebhookType
}

type Subscription struct {
	Id string `json:"id"`
}

type WebhookDailySettings struct {
	Timezone       *string `json:"timezone"`
	MidnightOffset *int    `json:"midnight_offset"`
}

type AlmanaxWebhook struct {
	Id             uuid.UUID
	DailySettings  WebhookDailySettings
	BonusBlacklist []string
	BonusWhitelist []string
	Subscriptions  []Subscription
	Callback       string
	Format         string
	WantsIsoDate   bool
	Mentions       *map[string][]MentionDTO
	Intervals      []string
	WeeklyWeekday  *string
	LastFiredAt    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (a AlmanaxWebhook) GetMentions() *map[string][]MentionDTO {
	return a.Mentions
}

func (a AlmanaxWebhook) GetTimezone() string {
	if a.DailySettings.Timezone == nil {
		return ""
	}
	return *a.DailySettings.Timezone
}

func (a AlmanaxWebhook) GetId() uuid.UUID {
	return a.Id
}

func (a AlmanaxWebhook) GetCallback() string {
	return a.Callback
}

func (a AlmanaxWebhook) GetPreviewLength() int {
	return 0
}

func (a AlmanaxWebhook) IsWantIsoDate() bool {
	return a.WantsIsoDate
}

type TwitterWebhook struct {
	Id            uuid.UUID  `json:"id"`
	Callback      string     `json:"-"`
	Whitelist     []string   `json:"bonus_whitelist"`
	Blacklist     []string   `json:"bonus_blacklist"`
	Format        string     `json:"format"`
	LastFiredAt   *time.Time `json:"last_fired_at"`
	PreviewLength int        `json:"preview_length"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func (s TwitterWebhook) GetLastFiredAt() *time.Time {
	return s.LastFiredAt
}

func (s TwitterWebhook) GetCreatedAt() time.Time {
	return s.CreatedAt
}

func (s TwitterWebhook) GetUpdatedAt() time.Time {
	return s.UpdatedAt
}

func (s TwitterWebhook) GetFormat() string {
	return s.Format
}

func (s TwitterWebhook) GetMentions() *map[string][]MentionDTO {
	return nil
}

func (s TwitterWebhook) GetTimezone() string {
	return ServerTz
}

func (s TwitterWebhook) GetBlacklist() []string {
	return s.Blacklist
}

func (s TwitterWebhook) GetWhitelist() []string {
	return s.Whitelist
}

func (s TwitterWebhook) GetId() uuid.UUID {
	return s.Id
}

func (s TwitterWebhook) GetType() string {
	return TwitterWebhookType
}

func (s TwitterWebhook) GetCallback() string {
	return s.Callback
}

func (s TwitterWebhook) GetPreviewLength() int {
	return s.PreviewLength
}

func (s TwitterWebhook) IsWantIsoDate() bool {
	return false
}

type RssWebhook struct {
	Id            uuid.UUID  `json:"id"`
	Callback      string     `json:"-"`
	Whitelist     []string   `json:"bonus_whitelist"`
	Blacklist     []string   `json:"bonus_blacklist"`
	Format        string     `json:"format"`
	LastFiredAt   *time.Time `json:"last_fired_at"`
	PreviewLength int        `json:"preview_length"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func (s RssWebhook) GetLastFiredAt() *time.Time {
	return s.LastFiredAt
}

func (s RssWebhook) GetCreatedAt() time.Time {
	return s.CreatedAt
}

func (s RssWebhook) GetUpdatedAt() time.Time {
	return s.UpdatedAt
}

func (s RssWebhook) GetFormat() string {
	return s.Format
}

func (s RssWebhook) GetMentions() *map[string][]MentionDTO {
	return nil
}

func (s RssWebhook) GetTimezone() string {
	return ServerTz
}

func (s RssWebhook) GetBlacklist() []string {
	return s.Blacklist
}

func (s RssWebhook) GetWhitelist() []string {
	return s.Whitelist
}

func (s RssWebhook) GetId() uuid.UUID {
	return s.Id
}

func (s RssWebhook) GetType() string {
	return RSSWebhookType
}

func (s RssWebhook) GetCallback() string {
	return s.Callback
}

func (s RssWebhook) GetPreviewLength() int {
	return s.PreviewLength
}

func (s RssWebhook) IsWantIsoDate() bool {
	return false
}

type ISocialHook interface {
	GetId() uuid.UUID
	GetLastFiredAt() *time.Time
	GetCallback() string
	GetCreatedAt() time.Time
	GetUpdatedAt() time.Time
	GetFormat() string
	GetPreviewLength() int
	GetBlacklist() []string
	GetWhitelist() []string
}

type ISocialHookUpdate interface {
	GetId() uuid.UUID
	GetBlacklist() []string
	GetWhitelist() []string
	GetSubscriptions() []string
	GetPreviewLength() *int
}

type SocialHookCreate struct {
	Whitelist     []string `json:"whitelist"`
	Blacklist     []string `json:"blacklist"`
	Subscriptions []string `json:"subscriptions"`
	PreviewLength *int     `json:"preview_length"`
	Callback      string   `json:"callback"`
	Format        string   `json:"format"`
}

type MentionDTO struct {
	DiscordId      uint64 `json:"discord_id"`
	IsRole         bool   `json:"is_role"`
	PingDaysBefore *int   `json:"ping_days_before"`
}

type AlmanaxHookPut struct {
	BonusWhitelist []string                 `json:"bonus_whitelist"`
	BonusBlacklist []string                 `json:"bonus_blacklist"`
	DailySettings  *WebhookDailySettings    `json:"daily_settings"`
	Subscriptions  []string                 `json:"subscriptions"`
	WantsIsoDate   *bool                    `json:"iso_date"`
	Mentions       *map[string][]MentionDTO `json:"mentions"`
	Intervals      []string                 `json:"intervals"`
	WeeklyWeekday  *string                  `json:"weekly_weekday"`
}

type CreateAlmanaxHook struct {
	BonusWhitelist []string
	BonusBlacklist []string
	DailySettings  WebhookDailySettings
	Callback       string
	Subscriptions  []string
	WantsIsoDate   bool
	Format         string
	Mentions       *map[string][]MentionDTO
	Intervals      []string
	WeeklyWeekday  *string
}

type SocialWebhookDTO struct {
	Id            uuid.UUID  `json:"id"`
	Whitelist     []string   `json:"whitelist"`
	Blacklist     []string   `json:"blacklist"`
	Subscriptions []string   `json:"subscriptions"`
	Format        string     `json:"format"`
	PreviewLength int        `json:"preview_length"`
	CreatedAt     time.Time  `json:"created_at"`
	LastFiredAt   *time.Time `json:"last_fired_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type SocialWebhookPutDb struct {
	Id            uuid.UUID
	Whitelist     []string `json:"whitelist"`
	Blacklist     []string `json:"blacklist"`
	Subscriptions []string `json:"subscriptions"`
	PreviewLength *int     `json:"preview_length"`
}

func (hook SocialWebhookPutDb) GetId() uuid.UUID {
	return hook.Id
}

func (hook SocialWebhookPutDb) GetBlacklist() []string {
	return hook.Blacklist
}

func (hook SocialWebhookPutDb) GetWhitelist() []string {
	return hook.Whitelist
}

func (hook SocialWebhookPutDb) GetSubscriptions() []string {
	return hook.Subscriptions
}

func (hook SocialWebhookPutDb) GetPreviewLength() *int {
	return hook.PreviewLength
}
