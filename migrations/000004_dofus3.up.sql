update almanax_feeds
set
    human_readable_id = replace (human_readable_id, 'dofus2', 'dofus3');

update rss_feeds
set
    api_readable_id = replace (api_readable_id, 'dofus2', 'dofus3');

update twitter_feeds
set
    human_readable_id = replace (human_readable_id, 'dofus2', 'dofus3');
