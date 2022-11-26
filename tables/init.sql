create table if not exists youtube_videos (
    id bigserial primary key,
    video_id text not null unique,
    channel_id text not null,
    channel_title text not null,
    title text not null,
    published_at timestamptz not null
);
