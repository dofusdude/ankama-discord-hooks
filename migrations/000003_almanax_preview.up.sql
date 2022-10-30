create type almanax_interval as enum ('daily', 'weekly', 'monthly');
alter table almanax_webhooks add intervals almanax_interval[] default array['daily']::almanax_interval[];
alter table almanax_webhooks add weekly_weekday varchar(255) default null;
