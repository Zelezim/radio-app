-- 003_faithfm_full_schedule.sql
--
-- Replaces the 3 placeholder rows from 001 / 002 with the real Faith FM
-- anchor shows (Breakfast Show / Bible Hour Live / Voice of Prophecy),
-- preserving any votes and ratings already cast against them, and then
-- inserts the remaining 19 shows from the Faith FM weekday rotation.
--
-- Safe to re-run: a UNIQUE constraint on programs.title makes the
-- inserts idempotent.

-- 1) Make titles unique so re-running can't create duplicates.
do $$ begin
  if not exists (
    select 1 from pg_constraint where conname = 'programs_title_unique'
  ) then
    alter table programs add constraint programs_title_unique unique (title);
  end if;
end $$;

-- 2) Rebrand the existing placeholder rows. The IN-clause covers BOTH
--    states: post-001 (Triple J names) and post-002 (interim Faith FM
--    placeholders). Whichever row matches gets updated in place — its
--    UUID stays the same, so existing votes/ratings carry over.

update programs
   set title       = 'Breakfast Show',
       description = 'Positively different news, interviews and Bible studies with Skafy & friends.',
       schedule    = 'Daily 17:00',
       image_url   = 'https://images.unsplash.com/photo-1478737270239-2f02b77fc618?w=600'
 where title in ('Morning Drive', 'Morning Devotion');

update programs
   set title       = 'The Bible Hour Live',
       description = 'Live Bible study with Benjamin Ng and Maritza Brunt.',
       schedule    = 'Daily 21:00',
       image_url   = 'https://images.unsplash.com/photo-1504052434569-70ad5836ab65?w=600'
 where title in ('Lunchtime Lounge', 'Family Hour');

update programs
   set title       = 'Voice of Prophecy',
       description = 'Christ-centred Bible teaching from the classic broadcast started by H.M.S. Richards.',
       schedule    = 'Daily 15:00',
       image_url   = 'https://images.unsplash.com/photo-1438032005730-c779502df39b?w=600'
 where title in ('After Dark', 'Sunday Service');

-- 3) Insert the remaining Faith FM rotation.
insert into programs (title, description, schedule, image_url) values
  ('A Light In The Dark',
   'End-time prophecy and current events with Peter Watts and John Weeks.',
   'Daily 11:00',
   'https://images.unsplash.com/photo-1502920917128-1aa500764cbd?w=600'),

  ('HeartWise',
   'Faith-and-health insights with Dr. James Marcum.',
   'Daily 12:14',
   'https://images.unsplash.com/photo-1545239351-1141bd82e8a6?w=600'),

  ('The Desire of Ages',
   'Devotional readings from the classic Christian text, presented by Nancy Hamilton.',
   'Daily 13:00',
   'https://images.unsplash.com/photo-1531425300797-d5dc8b021c84?w=600'),

  ('Lead Your Life',
   'Leadership through love with Vikram Panchal and Graeme Christian.',
   'Daily 13:28',
   'https://images.unsplash.com/photo-1521737604893-d14cc237f11d?w=600'),

  ('It Is Written Australia',
   'Christ-centred Bible teaching from Gary Kent and Shawn Boonstra.',
   'Daily 14:17 · 01:30 · 06:11',
   'https://images.unsplash.com/photo-1497633762265-9d179a990aa6?w=600'),

  ('Prophecy With Purpose',
   'Biblical prophecy seminars with Christopher Petersen.',
   'Daily 15:27',
   'https://images.unsplash.com/photo-1490127252417-7c393f993ee4?w=600'),

  ('10 Minute Marriage Tips to Keep You Hitched',
   'Quick, practical marriage advice from Phil Yates and Kyle Morrison.',
   'Daily 16:24',
   'https://images.unsplash.com/photo-1519741497674-611481863552?w=600'),

  ('Tassie Encounters Moments',
   'Stories of faith from Tasmania with David Leo and Jason Cook.',
   'Daily 20:00',
   'https://images.unsplash.com/photo-1444703686981-a3abbc4d4fe3?w=600'),

  ('Classic Radio Sermons Featuring Joe Crews',
   'Timeless sermons from Amazing Facts founder Joe Crews.',
   'Daily 20:27',
   'https://images.unsplash.com/photo-1485579149621-3123dd979885?w=600'),

  ('The Faith Experiment',
   'Q&A on the gospel of the Kingdom with Robbie Berghan.',
   'Daily 22:00',
   'https://images.unsplash.com/photo-1481627834876-b7833e8f5570?w=600'),

  ('The Perfect Storm',
   'Cultural commentary from a Christian perspective with Justin Lawman and Rick Hergenhan.',
   'Daily 23:00',
   'https://images.unsplash.com/photo-1500382017468-9049fed747ef?w=600'),

  ('End Time Events with Dr Jillda and Pr Ben',
   'Signs of the Second Coming explored by Dr Jillda and Pr Ben.',
   'Daily 23:35',
   'https://images.unsplash.com/photo-1495020689067-958852a7765e?w=600'),

  ('SA Bible Study',
   'Growing in a relationship with God — Nick Creta and friends.',
   'Daily 00:30',
   'https://images.unsplash.com/photo-1455156218388-5e61b526818b?w=600'),

  ('Jason Sliger Radio Sermons',
   'Sermons on Christian living from Jason Sliger.',
   'Daily 02:24',
   'https://images.unsplash.com/photo-1507692049790-de58290a4334?w=600'),

  ('Men''s Matters',
   'Living well as a man of faith with Peter Keioskie.',
   'Daily 03:30',
   'https://images.unsplash.com/photo-1556761175-5973dc0f32e7?w=600'),

  ('Drivetime — Big Q&A',
   'Big questions, biblical answers with Garry and friends.',
   'Daily 04:30',
   'https://images.unsplash.com/photo-1503376780353-7e6692767b70?w=600'),

  ('Cover to Cover: Jesus in All the Bible',
   'Tracing Jesus through every book of Scripture with Doug Batchelor.',
   'Daily 05:30',
   'https://images.unsplash.com/photo-1497369702784-c66f5359add6?w=600'),

  ('Encounters With Jesus',
   'Reflections on Gospel encounters with Benjamin Ng.',
   'Daily 07:00',
   'https://images.unsplash.com/photo-1511895426328-dc8714191300?w=600'),

  ('Philosophy''s Achilles'' Heel',
   'Christian apologetics with Chris Holland and Phil Pennington.',
   'Daily 07:39',
   'https://images.unsplash.com/photo-1481627834876-b7833e8f5570?w=600')

on conflict (title) do nothing;
