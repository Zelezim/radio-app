-- RadioApp initial schema
-- Run this in the Supabase SQL editor (or via `supabase db push`).

-- Enable pgcrypto for gen_random_uuid()
create extension if not exists pgcrypto;

-- Profile extension on top of auth.users
create table if not exists profiles (
  id uuid references auth.users primary key,
  full_name text,
  phone text,
  created_at timestamp default now()
);

-- Radio programs
create table if not exists programs (
  id uuid primary key default gen_random_uuid(),
  title text not null,
  description text,
  schedule text,
  image_url text,
  created_at timestamp default now()
);

-- Votes on content (programs or songs)
create table if not exists votes (
  id uuid primary key default gen_random_uuid(),
  user_id uuid references profiles(id),
  content_id uuid not null,
  content_type text not null, -- 'program' or 'music'
  value int not null,         -- 1 (up) or -1 (down)
  created_at timestamp default now(),
  unique(user_id, content_id, content_type)
);

-- Ratings of programs (1..5 stars)
create table if not exists ratings (
  id uuid primary key default gen_random_uuid(),
  user_id uuid references profiles(id),
  program_id uuid references programs(id),
  score int check (score between 1 and 5),
  created_at timestamp default now(),
  unique(user_id, program_id)
);

-- CRM: contact requests / inbound forms
create table if not exists contact_requests (
  id uuid primary key default gen_random_uuid(),
  name text not null,
  email text not null,
  phone text,
  address text,
  message text,
  request_type text,                 -- 'visit', 'cd_dvd', 'book', 'other'
  status text default 'new',         -- 'new', 'in_progress', 'done'
  created_at timestamp default now()
);

-- Aggregated views for the frontend ---------------------------------------

-- Vote tallies per content item
create or replace view vote_totals as
select content_id,
       content_type,
       coalesce(sum(case when value =  1 then 1 else 0 end), 0)::int as up_votes,
       coalesce(sum(case when value = -1 then 1 else 0 end), 0)::int as down_votes,
       coalesce(sum(value), 0)::int as score
from votes
group by content_id, content_type;

-- Average rating per program
create or replace view program_rating_avg as
select program_id,
       round(avg(score)::numeric, 2) as avg_score,
       count(*)::int as rating_count
from ratings
group by program_id;

-- Row level security (RLS) ------------------------------------------------
-- The Go backend uses the service role key, which bypasses RLS. We still
-- enable RLS so that any client connecting with the anon key cannot read
-- or write directly.

alter table profiles          enable row level security;
alter table votes             enable row level security;
alter table ratings           enable row level security;
alter table contact_requests  enable row level security;

-- Programs are public read-only.
alter table programs enable row level security;
drop policy if exists "programs are readable by everyone" on programs;
create policy "programs are readable by everyone"
  on programs for select using (true);

-- Seed a few programs so the UI has something to render on first boot.
insert into programs (title, description, schedule, image_url)
values
  ('Morning Drive', 'Wake up with the best mix of indie and rock.',
   'Mon-Fri 06:00-09:00',
   'https://images.unsplash.com/photo-1485579149621-3123dd979885?w=600'),
  ('Lunchtime Lounge', 'Chillout beats while you eat.',
   'Mon-Fri 12:00-13:00',
   'https://images.unsplash.com/photo-1514525253161-7a46d19cd819?w=600'),
  ('After Dark', 'Late night electronic sessions.',
   'Daily 22:00-00:00',
   'https://images.unsplash.com/photo-1470225620780-dba8ba36b745?w=600')
on conflict do nothing;
