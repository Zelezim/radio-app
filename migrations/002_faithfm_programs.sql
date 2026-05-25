-- 002_faithfm_programs.sql
--
-- Rebrands the three demo programs from the Triple J / generic-rock seed
-- to Faith-FM-themed placeholders. Safe to re-run: each statement only
-- touches rows that match the previous title, and existing votes/ratings
-- are preserved (we UPDATE in place rather than DELETE + INSERT).
--
-- Run this in the Supabase SQL Editor after switching the project to
-- Faith FM. Replace these placeholders with the real Faith FM schedule
-- as soon as you have it.

update programs
   set title       = 'Morning Devotion',
       description = 'Start your day with Scripture readings and uplifting worship music.',
       schedule    = 'Mon-Fri 06:00-09:00',
       image_url   = 'https://images.unsplash.com/photo-1507692049790-de58290a4334?w=600'
 where title = 'Morning Drive';

update programs
   set title       = 'Family Hour',
       description = 'Stories, music and chat for the whole family — kid-friendly and fun.',
       schedule    = 'Mon-Fri 12:00-13:00',
       image_url   = 'https://images.unsplash.com/photo-1511895426328-dc8714191300?w=600'
 where title = 'Lunchtime Lounge';

update programs
   set title       = 'Sunday Service',
       description = 'Live worship and sermons from Australian churches.',
       schedule    = 'Sun 09:00-11:00',
       image_url   = 'https://images.unsplash.com/photo-1438032005730-c779502df39b?w=600'
 where title = 'After Dark';
