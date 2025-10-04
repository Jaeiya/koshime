# Changelog (10/4/2025)

## Feat

- add RSS feed lookup ([1679e25](https://github.com/Jaeiya/koshime/commit/1679e25)) `7-5-2025`
- add file binding support ([3399656](https://github.com/Jaeiya/koshime/commit/3399656)) `9-25-2025`

**watch_anime**

- make `watch` warnings brighter ([21d6136](https://github.com/Jaeiya/koshime/commit/21d6136)) `7-3-2025`
- make sure list item titles are not empty ([621812a](https://github.com/Jaeiya/koshime/commit/621812a)) `7-22-2025`

**view**

- place more active menu items at top ([1347ff5](https://github.com/Jaeiya/koshime/commit/1347ff5)) `7-3-2025`
- add default error tea msg for views ([76dc128](https://github.com/Jaeiya/koshime/commit/76dc128)) `7-4-2025`

**fansub_parser**

- add support for VARYG fansub group ([64f4067](https://github.com/Jaeiya/koshime/commit/64f4067)) `7-3-2025`
- export batch error ([09a2c57](https://github.com/Jaeiya/koshime/commit/09a2c57)) `7-5-2025`
- strip all possible episode versions ([47f383a](https://github.com/Jaeiya/koshime/commit/47f383a)) `8-27-2025`
- get explicit episode versions ([7f4e61d](https://github.com/Jaeiya/koshime/commit/7f4e61d)) `8-27-2025`

**view_template**

- update with `init()` ([01b3e84](https://github.com/Jaeiya/koshime/commit/01b3e84)) `7-4-2025`

**ui_components**

- support disabling list filtering ([5f73c32](https://github.com/Jaeiya/koshime/commit/5f73c32)) `7-5-2025`

**add_rss**

- display file info along with title and episode ([4abf05d](https://github.com/Jaeiya/koshime/commit/4abf05d)) `7-6-2025`

**ui_menu**

- add option to enable/disable menu rotation ([8b31293](https://github.com/Jaeiya/koshime/commit/8b31293)) `7-6-2025`
- support menu descriptions ([dc891e1](https://github.com/Jaeiya/koshime/commit/dc891e1)) `7-6-2025`
- reduce left margin on description ([0feabfe](https://github.com/Jaeiya/koshime/commit/0feabfe)) `7-6-2025`

**watch_list**

- add menu descriptions ([95fc0da](https://github.com/Jaeiya/koshime/commit/95fc0da)) `7-6-2025`

**ui_helpers**

- reduce left margin on menu display ([3e08053](https://github.com/Jaeiya/koshime/commit/3e08053)) `7-6-2025`

**menu**

- use menu model for viewing/updating menu ([c14366f](https://github.com/Jaeiya/koshime/commit/c14366f)) `7-6-2025`

**fansub_filter**

- accept 50% or more confidence as valid find ([e2bb862](https://github.com/Jaeiya/koshime/commit/e2bb862)) `7-22-2025`
- make normalization more aggressive ([6be36c2](https://github.com/Jaeiya/koshime/commit/6be36c2)) `7-22-2025`
- fallback to previous confidence thresholds ([8e620f7](https://github.com/Jaeiya/koshime/commit/8e620f7)) `7-22-2025`

---

## Fix

**fansub_parser**

- don't use invalid ext for filename ([b3e6bad](https://github.com/Jaeiya/koshime/commit/b3e6bad)) `7-3-2025`

**token**

- forgot to implement token reset (whoops) ([7484e9b](https://github.com/Jaeiya/koshime/commit/7484e9b)) `7-6-2025`

**rss_scraper**

- forgot error handling ([7c508c4](https://github.com/Jaeiya/koshime/commit/7c508c4)) `7-6-2025`

**fansub_filter**

- do not add blank entry for jpn title ([bb6b42b](https://github.com/Jaeiya/koshime/commit/bb6b42b)) `7-22-2025`
- assume 50 or greater confidence = found ([b6d75ac](https://github.com/Jaeiya/koshime/commit/b6d75ac)) `7-22-2025`
- compensate for double whitespace ([21cae21](https://github.com/Jaeiya/koshime/commit/21cae21)) `7-22-2025`
- test for duplicate entries during binding lookup ([ebcc0b3](https://github.com/Jaeiya/koshime/commit/ebcc0b3)) `9-25-2025`

**kitsu_urls**

- update top level domain to `io` ([ee1a0db](https://github.com/Jaeiya/koshime/commit/ee1a0db)) `8-29-2025`

**watch_anime**

- always update total episodes ([dd0919d](https://github.com/Jaeiya/koshime/commit/dd0919d)) `10-4-2025`

---

## Docs

**main**

- leave commented test code for future use ([49cdb64](https://github.com/Jaeiya/koshime/commit/49cdb64)) `7-22-2025`
- add filename info to debug info ([63834fb](https://github.com/Jaeiya/koshime/commit/63834fb)) `7-22-2025`

**fansub_parser**

- add reasoning for 7 versions ([4884925](https://github.com/Jaeiya/koshime/commit/4884925)) `8-27-2025`

---

## Refactor

**kitsu_api**

- clarify actual username value as profile slug ([eef6e91](https://github.com/Jaeiya/koshime/commit/eef6e91)) `7-6-2025`

**ui_helpers**

- move `DisplayMenuItems` into the menu model ([5dc9376](https://github.com/Jaeiya/koshime/commit/5dc9376)) `7-6-2025`

---

## Perf

**fansub_filter**

- use more efficient loop over fields ([f3954b3](https://github.com/Jaeiya/koshime/commit/f3954b3)) `7-22-2025`

---

## Chore

**add_anime**

- remove deprecated code ([ab25509](https://github.com/Jaeiya/koshime/commit/ab25509)) `7-5-2025`

---
