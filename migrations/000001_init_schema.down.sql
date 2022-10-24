drop index
    idx_almanax_feeds_api_human_readable_id,
    idx_twitter_feeds_api_readable_id,
    idx_rss_feeds_api_readable_id,
    idx_webhooks_callback,
    idx_feeds_deleted_at,
    idx_weekday_translations_language,
    idx_weekday_translations_weekday,
    idx_almanax_mentions_almanax_bonus_id,
    idx_discord_mentions_discord_id;

drop table
    weekday_translations,
    rss_feeds,
    twitter_feeds,
    almanax_mentions,
    discord_mentions,
    almanax_feeds,
    almanax_webhooks,
    twitter_webhooks,
    rss_webhooks,
    subscriptions,
    feeds,
    webhooks;

drop type hooktopic;