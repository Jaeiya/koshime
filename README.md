# Koshime
This is a `2.0` version of [Wakitsu], my first application designed to solve a specific problem; updating my [Kitsu] anime library. It was written in `TypeScript` and it got the job done, but it was a true CLI app, not a TUI (Terminal User Interface).

Because it was a proof-of-concept, it [didn't have proper fansub filtering] (regex is an abomination for performance) and required execution for every single operation. Cleaning your watched dir or manual progression of an anime required you to enter separate commands. 

This has all been fixed in **Koshime**. It is a proper TUI that doesn't require you to memorize commands or wade through very specific argument syntax. Everything is at your fingertips from the beginning, and every operation has been tuned for performance.

Unlike [Wakitsu], **Koshime** has a fansub parser that is not only capable of discovering just about every possible combination of naming-convention, but also extracts and normalizes all meta-data.

> It's better in every way

## Installation
Download the latest binary from the releases page and stick it either in the directory where your anime is, or in a directory referenced in your environment `PATH`. 

## How to Use
This is a terminal application, so open up your favorite terminal, navigate to your anime directory, and type `koshime`; it's really that simple. You'll be walked through the setup process. 

Because Koshime is instanced to where you execute it, if you navigate in your terminal between `/dir/my_anime_series` and `/dir/my_anime_movies` then you'll have to set it up for both those directories. 

While it's possible to keep two distinct instances working without issue, it's not recommended. If at any point you decide to reload one of the instances, it will load your entire [Kitsu] library, not just *series* or *movies*. Neither **Koshime** nor **Kitsu** distinguishes between the type of content, only that content exists.

As long as you execute **Koshime** from where your anime is downloaded, it doesn't matter what type it is, it will always be picked up.

## Features

- Add anime to watch list
- Manage your watch list
- Search Kitsu or Local library for anime 
- Watch anime directly from prompt with auto-update option
- Manage your `(watched)` directory
- Bind anime rss feed to qBittorrent client for auto-download
    - This uses nyaa (and mirrors) for feed links
- Manage your access token with [Kitsu]
    - This is necessary because Kitsu only keeps a login session active for 30 days
    - Yes, I could have it auto-renew, but what if you don't watch anime for a month?
        - Koshime **does not** run in the background

[didn't have proper fansub filtering]:https://github.com/Jaeiya/wakitsu/blob/f103342c41c59c8c15d946cc3f9ba299d0956b3c/src/lib/utils.ts#L119
[Kitsu]:https://kitsu.app
[Wakitsu]:https://github.com/Jaeiya/Wakitsu
