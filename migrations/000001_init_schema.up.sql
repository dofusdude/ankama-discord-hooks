create type hooktopic as enum ('twitter', 'rss', 'almanax');

create table webhooks (
    id uuid default gen_random_uuid() not null
        primary key,
    format                text,
    last_fired_at         timestamp with time zone,
    callback              text,
    type                 hooktopic,

    created_at            timestamp with time zone default now(),
    updated_at            timestamp with time zone default now(),
    deleted_at            timestamp with time zone
);
alter table webhooks owner to postgres;
create index idx_webhooks_callback on webhooks (callback); /* not unique to allow multiple webhooks for the same channel */


create table feeds (
    id bigserial not null primary key,
    created_at            timestamp with time zone default now(),
    updated_at            timestamp with time zone default now(),
    deleted_at            timestamp with time zone
);
alter table feeds owner to postgres;
create index idx_feeds_deleted_at on feeds (deleted_at);

create table subscriptions (
    id bigserial not null primary key,
    webhook_id            uuid not null references webhooks (id),
    feed_id              bigserial not null references feeds (id),
    created_at            timestamp with time zone default now()
);
alter table subscriptions owner to postgres;


create table rss_webhooks
(
    id uuid not null primary key constraint fk_webhook_rss_webhook
        references webhooks,
    whitelist text[],
    blacklist text[],
    preview_length bigint
);
alter table rss_webhooks owner to postgres;



create table twitter_webhooks
(
    id uuid not null primary key constraint fk_webhook_twitter_webhook
        references webhooks,
    whitelist text[],
    blacklist text[],
    preview_length bigint
);
alter table twitter_webhooks owner to postgres;


create table almanax_webhooks
(
    id uuid not null primary key constraint fk_webhook_almanax_webhook
        references webhooks,
    daily_timezone        text,
    daily_midnight_offset bigint,
    wants_iso_date        boolean,
    whitelist text[],
    blacklist text[]
);
alter table almanax_webhooks owner to postgres;

create table almanax_feeds
(
    id bigserial not null primary key constraint fk_almanax_feeds_webhook_feed
        references feeds,
    human_readable_id text,
    language   varchar(2)
);
alter table almanax_feeds owner to postgres;
create unique index idx_almanax_feeds_api_human_readable_id on almanax_feeds (human_readable_id);


create table discord_mentions
(
    id uuid default gen_random_uuid() not null
        primary key,

    discord_id bigint,
    is_role boolean,
    created_at timestamp with time zone default now()
);
alter table discord_mentions owner to postgres;
create index idx_discord_mentions_discord_id on discord_mentions (discord_id);

create table almanax_mentions (
    id uuid default gen_random_uuid() not null
        primary key,

    almanax_webhook_id uuid not null
        constraint fk_almanax_mentions_almanax_webhook
            references almanax_webhooks,
    almanax_bonus_id  text not null,
    discord_mention_id uuid not null
        constraint fk_almanax_mentions_discord_mention
            references discord_mentions,

    created_at timestamp with time zone default now()
);
alter table almanax_mentions owner to postgres;
create index idx_almanax_mentions_almanax_bonus_id on almanax_mentions (almanax_bonus_id);

create table twitter_feeds
(
    id bigserial not null primary key constraint fk_twitter_feeds_feed
        references feeds,
    twitter_id          bigint,
    is_official boolean default false, /* pre populated from webhook service */
    human_readable_id  text not null
);
alter table twitter_feeds owner to postgres;
create unique index idx_twitter_feeds_api_readable_id on twitter_feeds (human_readable_id);


create table rss_feeds
(
    id bigserial not null primary key constraint fk_rss_feeds_feed
        references feeds,
    url          text,
    api_readable_id  text not null,
    is_official boolean default false
);
alter table rss_feeds owner to postgres;
create unique index idx_rss_feeds_api_readable_id on rss_feeds (api_readable_id);

create table weekday_translations (
    id bigserial not null primary key,
    language varchar(2) not null,
    weekday  varchar(255) not null,
    translation varchar(255) not null
);

alter table weekday_translations owner to postgres;
create index idx_weekday_translations_language on weekday_translations (language);
create index idx_weekday_translations_weekday on weekday_translations (weekday);

/* insert translations */
insert into weekday_translations (language, weekday, translation) values ('en', 'Monday', 'Monday');
insert into weekday_translations (language, weekday, translation) values ('en', 'Tuesday', 'Tuesday');
insert into weekday_translations (language, weekday, translation) values ('en', 'Wednesday', 'Wednesday');
insert into weekday_translations (language, weekday, translation) values ('en', 'Thursday', 'Thursday');
insert into weekday_translations (language, weekday, translation) values ('en', 'Friday', 'Friday');
insert into weekday_translations (language, weekday, translation) values ('en', 'Saturday', 'Saturday');
insert into weekday_translations (language, weekday, translation) values ('en', 'Sunday', 'Sunday');

insert into weekday_translations (language, weekday, translation) values ('fr', 'Monday', 'Lundi');
insert into weekday_translations (language, weekday, translation) values ('fr', 'Tuesday', 'Mardi');
insert into weekday_translations (language, weekday, translation) values ('fr', 'Wednesday', 'Mercredi');
insert into weekday_translations (language, weekday, translation) values ('fr', 'Thursday', 'Jeudi');
insert into weekday_translations (language, weekday, translation) values ('fr', 'Friday', 'Vendredi');
insert into weekday_translations (language, weekday, translation) values ('fr', 'Saturday', 'Samedi');
insert into weekday_translations (language, weekday, translation) values ('fr', 'Sunday', 'Dimanche');

insert into weekday_translations (language, weekday, translation) values ('de', 'Monday', 'Montag');
insert into weekday_translations (language, weekday, translation) values ('de', 'Tuesday', 'Dienstag');
insert into weekday_translations (language, weekday, translation) values ('de', 'Wednesday', 'Mittwoch');
insert into weekday_translations (language, weekday, translation) values ('de', 'Thursday', 'Donnerstag');
insert into weekday_translations (language, weekday, translation) values ('de', 'Friday', 'Freitag');
insert into weekday_translations (language, weekday, translation) values ('de', 'Saturday', 'Samstag');
insert into weekday_translations (language, weekday, translation) values ('de', 'Sunday', 'Sonntag');

insert into weekday_translations (language, weekday, translation) values ('es', 'Monday', 'Lunes');
insert into weekday_translations (language, weekday, translation) values ('es', 'Tuesday', 'Martes');
insert into weekday_translations (language, weekday, translation) values ('es', 'Wednesday', 'Miércoles');
insert into weekday_translations (language, weekday, translation) values ('es', 'Thursday', 'Jueves');
insert into weekday_translations (language, weekday, translation) values ('es', 'Friday', 'Viernes');
insert into weekday_translations (language, weekday, translation) values ('es', 'Saturday', 'Sábado');
insert into weekday_translations (language, weekday, translation) values ('es', 'Sunday', 'Domingo');

insert into weekday_translations (language, weekday, translation) values ('it', 'Monday', 'Lunedì');
insert into weekday_translations (language, weekday, translation) values ('it', 'Tuesday', 'Martedì');
insert into weekday_translations (language, weekday, translation) values ('it', 'Wednesday', 'Mercoledì');
insert into weekday_translations (language, weekday, translation) values ('it', 'Thursday', 'Giovedì');
insert into weekday_translations (language, weekday, translation) values ('it', 'Friday', 'Venerdì');
insert into weekday_translations (language, weekday, translation) values ('it', 'Saturday', 'Sabato');
insert into weekday_translations (language, weekday, translation) values ('it', 'Sunday', 'Domenica');

/* prepare ids */
/* rss */
insert into feeds (created_at) values (now());
insert into feeds (created_at) values (now());
insert into feeds (created_at) values (now());
insert into feeds (created_at) values (now());

insert into feeds (created_at) values (now());
insert into feeds (created_at) values (now());
insert into feeds (created_at) values (now());
insert into feeds (created_at) values (now());

insert into feeds (created_at) values (now());
insert into feeds (created_at) values (now());
insert into feeds (created_at) values (now());
insert into feeds (created_at) values (now());

/* twitter */
insert into feeds (created_at) values (now());
insert into feeds (created_at) values (now());
insert into feeds (created_at) values (now());
insert into feeds (created_at) values (now());

insert into feeds (created_at) values (now());
insert into feeds (created_at) values (now());
insert into feeds (created_at) values (now());
insert into feeds (created_at) values (now());

insert into feeds (created_at) values (now());
insert into feeds (created_at) values (now());
insert into feeds (created_at) values (now());
insert into feeds (created_at) values (now());

insert into feeds (created_at) values (now());

/* almanax */
insert into feeds (created_at) values (now());
insert into feeds (created_at) values (now());
insert into feeds (created_at) values (now());
insert into feeds (created_at) values (now());
insert into feeds (created_at) values (now());

/* RSS */
insert into rss_feeds (id, url, api_readable_id, is_official) values (1 ,'https://www.dofus.com/fr/rss/news.xml', 'dofus2-fr-official-news', true);
insert into rss_feeds (id, url, api_readable_id, is_official) values (2, 'https://www.dofus.com/en/rss/news.xml', 'dofus2-en-official-news', true);
insert into rss_feeds (id, url, api_readable_id, is_official) values (3, 'https://www.dofus.com/es/rss/news.xml', 'dofus2-es-official-news', true);
insert into rss_feeds (id, url, api_readable_id, is_official) values (4, 'https://www.dofus.com/pt/rss/news.xml', 'dofus2-pt-official-news', true);

insert into rss_feeds (id, url, api_readable_id, is_official) values (5, 'https://www.dofus.com/fr/rss/changelog.xml', 'dofus2-fr-official-changelog', true);
insert into rss_feeds (id, url, api_readable_id, is_official) values (6, 'https://www.dofus.com/en/rss/changelog.xml', 'dofus2-en-official-changelog', true);
insert into rss_feeds (id, url, api_readable_id, is_official) values (7, 'https://www.dofus.com/es/rss/changelog.xml', 'dofus2-es-official-changelog', true);
insert into rss_feeds (id, url, api_readable_id, is_official) values (8, 'https://www.dofus.com/pt/rss/changelog.xml', 'dofus2-pt-official-changelog', true);

insert into rss_feeds (id, url, api_readable_id, is_official) values (9, 'https://www.dofus.com/fr/rss/devblog.xml', 'dofus2-fr-official-devblog', true);
insert into rss_feeds (id, url, api_readable_id, is_official) values (10, 'https://www.dofus.com/en/rss/devblog.xml', 'dofus2-en-official-devblog', true);
insert into rss_feeds (id, url, api_readable_id, is_official) values (11, 'https://www.dofus.com/es/rss/devblog.xml', 'dofus2-es-official-devblog', true);
insert into rss_feeds (id, url, api_readable_id, is_official) values (12, 'https://www.dofus.com/pt/rss/devblog.xml', 'dofus2-pt-official-devblog', true);

/* Twitter */
insert into twitter_feeds (id, twitter_id, human_readable_id, is_official) values (13, 83587596, 'DOFUS_EN', true);
insert into twitter_feeds (id, twitter_id, human_readable_id, is_official) values (14, 3334787519, 'ES_DOFUS', true);
insert into twitter_feeds (id, twitter_id, human_readable_id, is_official) values (15,72272795, 'DOFUSfr', true);
insert into twitter_feeds (id, twitter_id, human_readable_id, is_official) values (16,3201179218, 'DOFUS_PTBR', true);
insert into twitter_feeds (id, twitter_id, human_readable_id, is_official) values (17, 3947817100, 'DOFUSTouch_EN', true);
insert into twitter_feeds (id, twitter_id, human_readable_id, is_official) values (18, 39510851, 'AnkamaGames', true);
insert into twitter_feeds (id, twitter_id, human_readable_id, is_official) values (19, 3947714549, 'DOFUSTouch', true);
insert into twitter_feeds (id, twitter_id, human_readable_id, is_official) values (20, 1278714204808241152, 'DOFUSTouch_ES', true);
insert into twitter_feeds (id, twitter_id, human_readable_id, is_official) values (21, 1094244760888459264, 'KTA_En', true);
insert into twitter_feeds (id, twitter_id, human_readable_id, is_official) values (22, 985193436725829637, 'KTA_fr', true);
insert into twitter_feeds (id, twitter_id, human_readable_id, is_official) values (23, 1700816222, 'DPLNofficiel', true);
insert into twitter_feeds (id, twitter_id, human_readable_id, is_official) values (24, 2548826726, 'dofusbook_modo', true);
insert into twitter_feeds (id, twitter_id, human_readable_id, is_official) values (25, 153029506, 'JOL_Dofus', true);

/* Almanax */
insert into almanax_feeds (id, human_readable_id, language) values (26, 'dofus2_en', 'en');
insert into almanax_feeds (id, human_readable_id, language) values (27, 'dofus2_fr', 'fr');
insert into almanax_feeds (id, human_readable_id, language) values (28, 'dofus2_es', 'es');
insert into almanax_feeds (id, human_readable_id, language) values (29, 'dofus2_it', 'it');
insert into almanax_feeds (id, human_readable_id, language) values (30, 'dofus2_de', 'de');