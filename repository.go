package main

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var repositoryMutex = sync.Mutex{}

type Repository struct {
	conn *pgxpool.Pool
	ctx  context.Context
}

func (r *Repository) Init(ctx context.Context) error {
	var err error
	r.ctx = ctx
	r.conn, err = pgxpool.New(ctx, PostgresUrl)
	return err
}

func (r *Repository) Deinit() {
	r.conn.Close()
	r.conn = nil
}

func (r *Repository) GetAllWeekdayTranslations() (map[string]map[string]string, error) {
	var err error
	var rows pgx.Rows
	rows, err = r.conn.Query(r.ctx, "select weekday, translation, language from weekday_translations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var translations map[string]map[string]string
	for rows.Next() {
		var weekday string
		var language string
		var translation string
		err = rows.Scan(&weekday, &translation, &language)
		if err != nil {
			return nil, err
		}

		if translations == nil {
			translations = make(map[string]map[string]string)
		}

		if translations[language] == nil {
			translations[language] = make(map[string]string)
		}

		translations[language][weekday] = translation
	}

	return translations, err
}

func (r *Repository) GetTranslationFor(language string, enWeekday string) (string, error) {
	var err error
	var translation string
	err = r.conn.QueryRow(r.ctx, "select translation from weekday_translations where language = $1 and weekday = $2", language, enWeekday).Scan(&translation)
	return translation, err
}

func (r *Repository) FireStampWebhook(callback string) error {
	var err error
	_, err = r.conn.Exec(r.ctx, "update webhooks set last_fired_at = $1 where callback = $2", time.Now(), callback)
	return err
}

func (r *Repository) GetAlmanaxFeeds(ids []uint64) ([]AlmanaxFeed, error) {
	var err error
	var feeds []AlmanaxFeed
	var rows pgx.Rows
	if len(ids) == 0 {
		rows, err = r.conn.Query(r.ctx, "select af.id, af.human_readable_id, af.language, f.created_at from almanax_feeds af inner join feeds f on f.id = af.id where f.deleted_at is null")
	} else {
		rows, err = r.conn.Query(r.ctx, "select af.id, af.human_readable_id, af.language, f.created_at from almanax_feeds af inner join feeds f on f.id = af.id where f.deleted_at is null and af.id = any($1)", ids)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var feed AlmanaxFeed
		err = rows.Scan(&feed.Id, &feed.HumanReadableId, &feed.Language, &feed.CreatedAt)
		if err != nil {
			return nil, err
		}
		feeds = append(feeds, feed)
	}

	return feeds, err
}

func GetAlmanaxFeeds(ids []uint64, repo Repository) ([]AlmanaxFeed, error) {
	return repo.GetAlmanaxFeeds(ids)
}

func GetTwitterFeeds(ids []uint64, repo Repository) ([]TwitterFeed, error) {
	return repo.GetTwitterFeeds(ids)
}

func (r *Repository) GetTwitterFeeds(ids []uint64) ([]TwitterFeed, error) {
	var err error
	var feeds []TwitterFeed
	var rows pgx.Rows
	if len(ids) == 0 {
		rows, err = r.conn.Query(r.ctx, "select f.id, tf.twitter_id, tf.is_official, f.created_at, tf.human_readable_id from twitter_feeds tf inner join feeds f on f.id = tf.id where f.deleted_at is null")
	} else {
		rows, err = r.conn.Query(r.ctx, "select f.id, tf.twitter_id, tf.is_official, f.created_at, tf.human_readable_id from twitter_feeds tf inner join feeds f on f.id = tf.id where f.deleted_at is null and tf.id = any($1)", ids)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var feed TwitterFeed
		err = rows.Scan(&feed.Id, &feed.TwitterId, &feed.IsOfficial, &feed.CreatedAt, &feed.HumanReadableId)
		if err != nil {
			return nil, err
		}
		feeds = append(feeds, feed)
	}

	return feeds, err
}

func (r *Repository) GetRssFeeds(ids []uint64) ([]RssFeed, error) {
	var err error
	var feeds []RssFeed
	var rows pgx.Rows
	if len(ids) == 0 {
		rows, err = r.conn.Query(r.ctx, "select f.id, rf.url, rf.api_readable_id, rf.is_official, f.created_at from rss_feeds rf inner join feeds f on f.id = rf.id where f.deleted_at is null")
	} else {
		rows, err = r.conn.Query(r.ctx, "select f.id, rf.url, rf.api_readable_id, rf.is_official, f.created_at from rss_feeds rf inner join feeds f on f.id = rf.id where f.deleted_at is null and rf.id = any($1)", ids)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var feed RssFeed
		err = rows.Scan(&feed.Id, &feed.Url, &feed.ApiReadableId, &feed.IsOfficial, &feed.CreatedAt)
		if err != nil {
			return nil, err
		}
		feeds = append(feeds, feed)
	}

	return feeds, err
}

func GetRssFeeds(ids []uint64, repo Repository) ([]RssFeed, error) {
	return repo.GetRssFeeds(ids)
}

func (r *Repository) HasAlmanaxWebhook(id uuid.UUID) (bool, error) {
	var err error
	var exists bool
	err = r.conn.QueryRow(r.ctx, "select exists(select 1 from almanax_webhooks inner join webhooks w on almanax_webhooks.id = w.id where w.id = $1 and deleted_at is null)", id).Scan(&exists)
	return exists, err
}

func (r *Repository) HasSocialWebhook(socialType string, id uuid.UUID) (bool, error) {
	var err error
	var exists bool
	switch socialType {
	case TwitterWebhookType:
		err = r.conn.QueryRow(r.ctx, "select exists(select 1 from twitter_webhooks inner join webhooks w on w.id = twitter_webhooks.id where twitter_webhooks.id = $1 and w.deleted_at is null)", id).Scan(&exists)
	case RSSWebhookType:
		err = r.conn.QueryRow(r.ctx, "select exists(select 1 from rss_webhooks inner join webhooks w on w.id = rss_webhooks.id where rss_webhooks.id = $1 and w.deleted_at is null)", id).Scan(&exists)
	default:
		err = errors.New("invalid social type")
	}
	return exists, err
}

func (r *Repository) HasGetSocialFeeds(socialType string, feedIdentifiers []string) (bool, []uint64, error) {
	var err error
	var ids []uint64
	for _, identifier := range feedIdentifiers {
		var exists bool
		switch socialType {
		case TwitterWebhookType:
			err = r.conn.QueryRow(r.ctx, "select exists(select 1 from twitter_feeds tf inner join feeds f on f.id = tf.id where tf.human_readable_id = $1 and f.deleted_at is null)", identifier).Scan(&exists)
			if err != nil {
				return exists, []uint64{}, err
			}
			if !exists {
				return exists, []uint64{}, nil
			}
			var id uint64
			err = r.conn.QueryRow(r.ctx, "select f.id from twitter_feeds tf inner join feeds f on f.id = tf.id where tf.human_readable_id = $1 and f.deleted_at is null", identifier).Scan(&id)
			if err != nil {
				return exists, []uint64{}, err
			}
			ids = append(ids, id)

		case RSSWebhookType:
			err = r.conn.QueryRow(r.ctx, "select exists(select 1 from rss_feeds tf inner join feeds f on f.id = tf.id where tf.api_readable_id = $1 and f.deleted_at is null)", identifier).Scan(&exists)
			if err != nil {
				return exists, []uint64{}, err
			}
			if !exists {
				return exists, []uint64{}, nil
			}
			var id uint64
			err = r.conn.QueryRow(r.ctx, "select f.id from rss_feeds tf inner join feeds f on f.id = tf.id where tf.api_readable_id = $1 and f.deleted_at is null", identifier).Scan(&id)
			if err != nil {
				return exists, []uint64{}, err
			}
			ids = append(ids, id)
		default:
			err = errors.New("invalid social type")
		}
	}

	return len(ids) > 0, ids, err
}

func (r *Repository) HasGetAlmanaxFeeds(feedIdentifiers []string) (bool, []uint64, error) {
	var err error
	var ids []uint64
	for _, identifier := range feedIdentifiers {
		var exists bool
		err = r.conn.QueryRow(r.ctx, "select exists(select 1 from almanax_feeds tf inner join feeds f on f.id = tf.id where tf.human_readable_id = $1 and f.deleted_at is null)", identifier).Scan(&exists)
		if err != nil {
			return exists, []uint64{}, err
		}
		if !exists {
			return exists, []uint64{}, nil
		}
		var id uint64
		err = r.conn.QueryRow(r.ctx, "select f.id from almanax_feeds tf inner join feeds f on f.id = tf.id where tf.human_readable_id = $1 and f.deleted_at is null", identifier).Scan(&id)
		if err != nil {
			return exists, []uint64{}, err
		}
		ids = append(ids, id)
	}

	return len(ids) > 0, ids, err
}

func (r *Repository) hasWebhookCallback(callback string, tableName string) (bool, error) {
	var err error
	var exists bool
	err = r.conn.QueryRow(r.ctx, "select exists(select 1"+" from "+tableName+" tw inner join webhooks w on tw.id = w.id where w.callback = $1 and w.deleted_at is null)", callback).Scan(&exists)
	return exists, err
}

func (r *Repository) HasTwitterWebhookCallback(callback string) (bool, error) {
	return r.hasWebhookCallback(callback, "twitter_webhooks")
}

func (r *Repository) HasRssWebhookCallback(callback string) (bool, error) {
	return r.hasWebhookCallback(callback, "rss_webhooks")
}

func (r *Repository) HasAlmanaxWebhookCallback(callback string) (bool, error) {
	return r.hasWebhookCallback(callback, "almanax_webhooks")
}

func (r *Repository) GetSocialHookSubscriptions(socialType string, id uuid.UUID) ([]IFeed, error) {
	var err error
	var subs []IFeed

	var subRows pgx.Rows
	subRows, err = r.conn.Query(r.ctx, "select subscriptions.feed_id from subscriptions inner join webhooks w on w.id = subscriptions.webhook_id where subscriptions.webhook_id = $1 and w.deleted_at is null", id)
	if err != nil {
		return subs, err
	}
	defer subRows.Close()

	for subRows.Next() {
		var subId uint64
		err = subRows.Scan(&subId)
		if err != nil {
			return subs, err
		}
		switch socialType {
		case TwitterWebhookType:
			var sub TwitterFeed
			err = r.conn.QueryRow(r.ctx, "select tf.id, tf.twitter_id, f.created_at, tf.is_official, tf.human_readable_id from twitter_feeds tf inner join feeds f on f.id = tf.id where tf.id = $1", subId).
				Scan(&sub.Id, &sub.TwitterId, &sub.CreatedAt, &sub.IsOfficial, &sub.HumanReadableId)
			if err != nil {
				return subs, err
			}
			subs = append(subs, &sub)
		case RSSWebhookType:
			var sub RssFeed
			err = r.conn.QueryRow(r.ctx, "select rf.id, rf.url, f.created_at, rf.is_official, rf.api_readable_id from rss_feeds rf inner join feeds f on f.id = rf.id where rf.id = $1", subId).
				Scan(&sub.Id, &sub.Url, &sub.CreatedAt, &sub.IsOfficial, &sub.ApiReadableId)
			if err != nil {
				return subs, err
			}
			subs = append(subs, &sub)
		default:
			err = errors.New("invalid social type")
		}
	}

	return subs, err
}

func (r *Repository) GetAlmanaxHookSubscriptions(id uuid.UUID) ([]IFeed, error) {
	var err error
	var res []IFeed

	var rows pgx.Rows
	rows, err = r.conn.Query(r.ctx, "select subscriptions.id, af.human_readable_id, f.created_at, af.language from subscriptions inner join feeds f on f.id = subscriptions.feed_id inner join almanax_feeds af on f.id = af.id where subscriptions.webhook_id = $1", id)
	if err != nil {
		return []IFeed{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var sub AlmanaxFeed
		err = rows.Scan(&sub.Id, &sub.HumanReadableId, &sub.CreatedAt, &sub.Language)
		if err != nil {
			return []IFeed{}, err
		}
		res = append(res, sub)
	}

	return res, nil
}

func (r *Repository) DeleteHook(id uuid.UUID) error {
	var err error
	_, err = r.conn.Exec(r.ctx, "update webhooks set deleted_at = $1 where id = $2", time.Now(), id)
	return err
}

func (r *Repository) FindSocialFeedId(socialType string, humanId string) (uint64, error) {
	var err error
	var id uint64
	switch socialType {
	case TwitterWebhookType:
		var exists bool
		err = r.conn.QueryRow(r.ctx, "select exists(select 1 from feeds inner join twitter_feeds tf on feeds.id = tf.id where human_readable_id = $1)", humanId).Scan(&exists)
		if err != nil {
			return id, err
		}
		if !exists {
			return id, errors.New("feed not found")
		}
		err = r.conn.QueryRow(r.ctx, "select tf.id from feeds inner join twitter_feeds tf on feeds.id = tf.id where human_readable_id = $1", humanId).Scan(&id)
	case RSSWebhookType:
		var exists bool
		err = r.conn.QueryRow(r.ctx, "select exists(select 1 from feeds inner join rss_feeds tf on feeds.id = tf.id where api_readable_id = $1)", humanId).Scan(&exists)
		if err != nil {
			return id, err
		}
		if !exists {
			return id, errors.New("feed not found")
		}
		err = r.conn.QueryRow(r.ctx, "select rf.id from feeds inner join rss_feeds rf on feeds.id = rf.id where api_readable_id = $1", humanId).Scan(&id)
	default:
		err = errors.New("invalid social type")
	}
	return id, err
}

func (r *Repository) CreateSocialHook(socialType string, createHook SocialHookCreate) (uuid.UUID, error) {
	var err error
	if err != nil {
		return uuid.Nil, err
	}
	var id uuid.UUID
	err = r.conn.QueryRow(r.ctx, "insert into webhooks (format, callback, type) values ($1, $2, $3) returning id", createHook.Format, createHook.Callback, socialType).Scan(&id)
	if err != nil {
		return uuid.Nil, err
	}

	switch socialType {
	case TwitterWebhookType:
		_, err = r.conn.Exec(r.ctx, "insert into twitter_webhooks (id, whitelist, blacklist, preview_length) values ($1, $2, $3, $4)", id, createHook.Whitelist, createHook.Blacklist, createHook.PreviewLength)
		if err != nil {
			return uuid.Nil, err
		}
	case RSSWebhookType:
		_, err = r.conn.Exec(r.ctx, "insert into rss_webhooks (id, whitelist, blacklist, preview_length) values ($1, $2, $3, $4)", id, createHook.Whitelist, createHook.Blacklist, createHook.PreviewLength)
		if err != nil {
			return uuid.Nil, err
		}
	default:
		err = errors.New("invalid social type")
	}

	var allFound bool
	var feedIds []uint64
	allFound, feedIds, err = r.HasGetSocialFeeds(socialType, createHook.Subscriptions)
	if err != nil {
		return uuid.Nil, err
	}

	if !allFound {
		return uuid.Nil, errors.New("some feeds not found")
	}

	for _, feed := range feedIds {
		_, err = r.conn.Exec(r.ctx, "insert into subscriptions (webhook_id, feed_id) values ($1, $2)", id, feed)
		if err != nil {
			return uuid.Nil, err
		}
	}

	return id, err
}

func (r *Repository) UpdateAlmanaxHook(hook AlmanaxHookPut, id uuid.UUID) error {
	var err error
	if hook.BonusBlacklist != nil || hook.BonusWhitelist != nil {
		_, err = r.conn.Exec(r.ctx, "update almanax_webhooks set whitelist = null, blacklist = null where id = $1", id)
		if err != nil {
			return err
		}

		if hook.BonusBlacklist != nil {
			_, err = r.conn.Exec(r.ctx, "update almanax_webhooks set blacklist = $1 where id = $2", hook.BonusBlacklist, id)
			if err != nil {
				return err
			}
		}

		if hook.BonusWhitelist != nil {
			_, err = r.conn.Exec(r.ctx, "update almanax_webhooks set whitelist = $1 where id = $2", hook.BonusWhitelist, id)
			if err != nil {
				return err
			}
		}
	}

	if hook.DailySettings != nil {
		if hook.DailySettings.Timezone != nil {
			_, err = r.conn.Exec(r.ctx, "update almanax_webhooks set daily_timezone = $1 where id = $2", hook.DailySettings.Timezone, id)
			if err != nil {
				return err
			}
		}

		if hook.DailySettings.MidnightOffset != nil {
			_, err = r.conn.Exec(r.ctx, "update almanax_webhooks set daily_midnight_offset = $1 where id = $2", hook.DailySettings.MidnightOffset, id)
			if err != nil {
				return err
			}
		}
	}

	if hook.WantsIsoDate != nil {
		_, err = r.conn.Exec(r.ctx, "update almanax_webhooks set wants_iso_date = $1 where id = $2", hook.WantsIsoDate, id)
		if err != nil {
			return err
		}
	}

	if hook.Mentions != nil {
		var mentionIds []uuid.UUID
		err = r.conn.QueryRow(r.ctx, "select array_agg(am.id) from discord_mentions inner join almanax_mentions am on discord_mentions.id = am.discord_mention_id where am.almanax_webhook_id = $1", id).Scan(&mentionIds)
		if err != nil {
			return err
		}

		_, err = r.conn.Exec(r.ctx, "delete from almanax_mentions where almanax_webhook_id = $1", id)
		if err != nil {
			return err
		}

		_, err = r.conn.Exec(r.ctx, "delete from discord_mentions where id = any($1)", mentionIds)
		if err != nil {
			return err
		}

		for bonusId, mentions := range *hook.Mentions {
			for _, mention := range mentions {
				var mentionId uuid.UUID
				err = r.conn.QueryRow(r.ctx, "insert into discord_mentions (discord_id, is_role) values ($1, $2) returning id", mention.DiscordId, mention.IsRole).Scan(&mentionId)
				if err != nil {
					return err
				}
				_, err = r.conn.Exec(r.ctx, "insert into almanax_mentions (almanax_webhook_id, almanax_bonus_id, discord_mention_id, ping_days_before) values ($1, $2, $3, $4)",
					id, bonusId, mentionId)
				if err != nil {
					return err
				}
			}
		}
	}

	if hook.Subscriptions != nil {
		var hasFound bool
		var feedIds []uint64
		hasFound, feedIds, err = r.HasGetAlmanaxFeeds(hook.Subscriptions)
		if err != nil {
			return err
		}

		if !hasFound {
			return errors.New("some feeds not found")
		}

		_, err = r.conn.Exec(r.ctx, "delete from subscriptions where webhook_id = $1", id)
		if err != nil {
			return err
		}

		for _, feed := range feedIds {
			_, err = r.conn.Exec(r.ctx, "insert into subscriptions (webhook_id, feed_id) values ($1, $2)", id, feed)
			if err != nil {
				return err
			}
		}
	}

	err = r.setUpdatedHookTimestamp(id)

	return err
}

func (r *Repository) setUpdatedHookTimestamp(id uuid.UUID) error {
	_, err := r.conn.Exec(r.ctx, "update webhooks set updated_at = $1 where id = $2", time.Now(), id)
	return err
}

func (r *Repository) UpdateSocialHook(socialType string, hook ISocialHookUpdate) error {
	var err error
	var tableName string

	if hook.GetSubscriptions() != nil {
		_, err = r.conn.Exec(r.ctx, "delete from subscriptions where webhook_id = $1", hook.GetId())
		if err != nil {
			return err
		}

		var hasFeed bool
		var feedIds []uint64
		hasFeed, feedIds, err = r.HasGetSocialFeeds(socialType, hook.GetSubscriptions())
		if err != nil {
			return err
		}

		if !hasFeed {
			return errors.New("feed not found")
		}

		for _, feedId := range feedIds {
			_, err = r.conn.Exec(r.ctx, "insert into subscriptions (webhook_id, feed_id) values ($1, $2)", hook.GetId(), feedId)
			if err != nil {
				return err
			}
		}
	}

	switch socialType {
	case TwitterWebhookType:
		tableName = "twitter_webhooks"
	case RSSWebhookType:
		tableName = "rss_webhooks"
	default:
		err = errors.New("invalid social type")
	}

	if hook.GetBlacklist() != nil {
		_, err = r.conn.Exec(r.ctx, "update "+tableName+" set blacklist = $1 where id = $2", hook.GetBlacklist(), hook.GetId())
	}

	if hook.GetWhitelist() != nil {
		_, err = r.conn.Exec(r.ctx, "update "+tableName+" set whitelist = $1 where id = $2", hook.GetWhitelist(), hook.GetId())
	}

	if hook.GetPreviewLength() != nil {
		_, err = r.conn.Exec(r.ctx, "update "+tableName+" set preview_length = $1 where id = $2", hook.GetPreviewLength(), hook.GetId())
	}

	err = r.setUpdatedHookTimestamp(hook.GetId())

	return err
}

func (r *Repository) GetSocialHook(socialType string, id uuid.UUID) (ISocialHook, error) {
	var err error
	switch socialType {
	case TwitterWebhookType:
		var webhook TwitterWebhook
		err = r.conn.QueryRow(r.ctx, "select w.id, w.last_fired_at, w.callback, w.created_at, w.updated_at, tw.preview_length, w.format, tw.whitelist, tw.blacklist from twitter_webhooks tw inner join webhooks w on w.id = tw.id where tw.id = $1 and w.deleted_at is null", id).
			Scan(&webhook.Id, &webhook.LastFiredAt, &webhook.Callback, &webhook.CreatedAt, &webhook.UpdatedAt, &webhook.PreviewLength, &webhook.Format, &webhook.Whitelist, &webhook.Blacklist)
		return webhook, err
	case RSSWebhookType:
		var webhook RssWebhook
		err = r.conn.QueryRow(r.ctx, "select w.id, w.last_fired_at, w.callback, w.created_at, w.updated_at, rw.preview_length, w.format, rw.whitelist, rw.blacklist from rss_webhooks rw inner join webhooks w on w.id = rw.id where rw.id = $1 and w.deleted_at is null", id).
			Scan(&webhook.Id, &webhook.LastFiredAt, &webhook.Callback, &webhook.CreatedAt, &webhook.UpdatedAt, &webhook.PreviewLength, &webhook.Format, &webhook.Whitelist, &webhook.Blacklist)
		return webhook, err
	default:
		return nil, errors.New("unknown social type")
	}
}

func (r *Repository) DeleteHooksByCallback(callback string) error {
	_, err := r.conn.Exec(r.ctx, "update webhooks set deleted_at = $1 where callback = $2", time.Now(), callback)
	return err
}

func (r *Repository) GetAlmanaxDiscordMentions(id uuid.UUID) (map[string][]MentionDTO, error) {
	var err error
	var res = make(map[string][]MentionDTO)

	var rows pgx.Rows
	rows, err = r.conn.Query(r.ctx, "select almanax_mentions.almanax_bonus_id, dm.discord_id, dm.is_role, almanax_mentions.ping_days_before from almanax_mentions inner join discord_mentions dm on dm.id = almanax_mentions.discord_mention_id where almanax_webhook_id = $1", id)
	if err != nil {
		return res, err
	}
	defer rows.Close()

	for rows.Next() {
		var bonusId string

		var mention MentionDTO
		err = rows.Scan(&bonusId, &mention.DiscordId, &mention.IsRole, &mention.PingDaysBefore)
		if err != nil {
			return res, err
		}

		res[bonusId] = append(res[bonusId], mention)
	}

	return res, nil
}

func (r *Repository) CreateAlmanaxHook(createHook CreateAlmanaxHook) (uuid.UUID, error) {
	var err error
	var id uuid.UUID

	err = r.conn.QueryRow(r.ctx, "insert into webhooks (format, callback, type) values ($1, $2, $3) returning id", createHook.Format, createHook.Callback, "almanax").Scan(&id)
	if err != nil {
		return uuid.UUID{}, err
	}

	_, err = r.conn.Exec(r.ctx, "insert into almanax_webhooks (id, wants_iso_date, daily_midnight_offset, daily_timezone, blacklist, whitelist) values ($1, $2, $3, $4, $5, $6)",
		id, createHook.WantsIsoDate, createHook.DailySettings.MidnightOffset, createHook.DailySettings.Timezone, createHook.BonusBlacklist, createHook.BonusWhitelist)
	if err != nil {
		return uuid.UUID{}, err
	}

	var hasAllFeeds bool
	var feedIds []uint64
	hasAllFeeds, feedIds, err = r.HasGetAlmanaxFeeds(createHook.Subscriptions)
	if err != nil {
		return uuid.UUID{}, err
	}

	if !hasAllFeeds {
		return uuid.UUID{}, errors.New("some feeds not found")
	}

	for _, feed := range feedIds {
		_, err = r.conn.Exec(r.ctx, "insert into subscriptions (webhook_id, feed_id) values ($1, $2)", id, feed)
		if err != nil {
			return uuid.UUID{}, err
		}
	}

	if createHook.Mentions != nil {
		for bonus, mentions := range *createHook.Mentions {
			for _, mention := range mentions {
				var discordId uuid.UUID
				err = r.conn.QueryRow(r.ctx, "insert into discord_mentions (discord_id, is_role) values ($1, $2) returning id", mention.DiscordId, mention.IsRole).Scan(&discordId)
				if err != nil {
					return id, err
				}
				_, err = r.conn.Exec(r.ctx, "insert into almanax_mentions (almanax_webhook_id, almanax_bonus_id, discord_mention_id, ping_days_before) values ($1, $2, $3, $4)", id, bonus, discordId, mention.PingDaysBefore)
				if err != nil {
					return id, err
				}
			}
		}
	}

	return id, nil
}

func (r *Repository) GetAlmanaxHook(id uuid.UUID) (AlmanaxWebhook, error) {
	var err error

	var webhook AlmanaxWebhook
	err = r.conn.QueryRow(r.ctx, "select w.id, w.last_fired_at, w.callback, w.created_at, w.updated_at, w.format, aw.daily_timezone, aw.daily_midnight_offset, aw.wants_iso_date, aw.whitelist, aw.blacklist from almanax_webhooks aw inner join webhooks w on w.id = aw.id where w.id = $1 and w.deleted_at is null", id).
		Scan(&webhook.Id, &webhook.LastFiredAt, &webhook.Callback, &webhook.CreatedAt, &webhook.UpdatedAt, &webhook.Format,
			&webhook.DailySettings.Timezone, &webhook.DailySettings.MidnightOffset, &webhook.WantsIsoDate, &webhook.BonusWhitelist, &webhook.BonusBlacklist)

	mentions, err := r.GetAlmanaxDiscordMentions(id)
	if err != nil {
		return AlmanaxWebhook{}, err
	}

	if len(mentions) > 0 {
		webhook.Mentions = &mentions
	}

	return webhook, err
}

func (r *Repository) GetTwitterSubsForFeed(feed IFeed) ([]HasIdBlackWhiteList[string], error) {
	var err error
	var webhooks []HasIdBlackWhiteList[string]
	var subRows pgx.Rows
	subRows, err = r.conn.Query(r.ctx, "select s.webhook_id from subscriptions s inner join feeds f on f.id = s.feed_id inner join twitter_feeds rf on f.id = rf.id inner join webhooks w on s.webhook_id = w.id where rf.id = $1 and f.deleted_at is null and w.deleted_at is null", feed.GetId())
	if err != nil {
		if err == pgx.ErrNoRows {
			return webhooks, nil
		}
		return webhooks, err
	}
	defer subRows.Close()

	for subRows.Next() {
		var webhookId uuid.UUID
		err = subRows.Scan(&webhookId)
		if err != nil {
			log.Println("iterating empty")
			return webhooks, err
		}

		var webhook ISocialHook
		webhook, err = r.GetSocialHook(TwitterWebhookType, webhookId)
		if err != nil {
			log.Println("err in get social hook", webhookId)
			return nil, err
		}

		webhookFull := TwitterWebhook{
			Id:            webhookId,
			Callback:      webhook.GetCallback(),
			CreatedAt:     webhook.GetCreatedAt(),
			UpdatedAt:     webhook.GetUpdatedAt(),
			Format:        webhook.GetFormat(),
			Blacklist:     webhook.GetBlacklist(),
			Whitelist:     webhook.GetWhitelist(),
			PreviewLength: webhook.GetPreviewLength(),
			LastFiredAt:   webhook.GetLastFiredAt(),
		}

		webhooks = append(webhooks, webhookFull)
	}

	return webhooks, nil
}

func (r *Repository) GetRSSSubsForFeed(feed IFeed) ([]HasIdBlackWhiteList[string], error) {
	var err error
	var webhooks []HasIdBlackWhiteList[string]
	var subRows pgx.Rows
	subRows, err = r.conn.Query(r.ctx, "select s.webhook_id from subscriptions s inner join feeds f on f.id = s.feed_id inner join rss_feeds rf on f.id = rf.id inner join webhooks w on s.webhook_id = w.id where rf.id = $1 and f.deleted_at is null and w.deleted_at is null", feed.GetId())
	if err != nil {
		if err == pgx.ErrNoRows {
			return webhooks, nil
		}
		return webhooks, err
	}
	defer subRows.Close()

	for subRows.Next() {
		var webhookId uuid.UUID
		err = subRows.Scan(&webhookId)
		if err != nil {
			return webhooks, err
		}

		var webhook ISocialHook
		webhook, err = r.GetSocialHook(RSSWebhookType, webhookId)
		if err != nil {
			return nil, err
		}

		webhookFull := RssWebhook{
			Id:            webhookId,
			Callback:      webhook.GetCallback(),
			CreatedAt:     webhook.GetCreatedAt(),
			UpdatedAt:     webhook.GetUpdatedAt(),
			Format:        webhook.GetFormat(),
			Blacklist:     webhook.GetBlacklist(),
			Whitelist:     webhook.GetWhitelist(),
			PreviewLength: webhook.GetPreviewLength(),
			LastFiredAt:   webhook.GetLastFiredAt(),
		}

		webhooks = append(webhooks, webhookFull)
	}

	return webhooks, nil
}

func (r *Repository) GetAlmanaxSubsForFeed(feed IFeed) ([]AlmanaxWebhook, error) {
	var err error
	var webhooks []AlmanaxWebhook
	var subRows pgx.Rows
	subRows, err = r.conn.Query(r.ctx, "select s.webhook_id from subscriptions s inner join feeds f on f.id = s.feed_id inner join almanax_feeds rf on f.id = rf.id inner join webhooks w on s.webhook_id = w.id where rf.id = $1 and f.deleted_at is null and w.deleted_at is null", feed.GetId())
	if err != nil {
		if err.Error() == "no rows in result set" || err == pgx.ErrNoRows {
			return webhooks, nil
		}
		return webhooks, err
	}
	defer subRows.Close()

	for subRows.Next() {
		var webhookId uuid.UUID
		err = subRows.Scan(&webhookId)
		if err != nil {
			return webhooks, err
		}

		var webhook AlmanaxWebhook
		webhook, err = r.GetAlmanaxHook(webhookId)
		if err != nil {
			return nil, err
		}

		webhooks = append(webhooks, webhook)
	}

	return webhooks, nil
}
