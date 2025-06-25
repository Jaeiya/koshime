### Add Anime

Should allow users to interactively add a movie or anime series to their
library. Wakitsu was limited to just adding anime, but Koshime should also
ask if the user wants to look for an RSS feed for the anime series.

1. Once an anime has been selected, start the RSS query process.
1. Allow the user to specify an optional fansub & resolution.
1. If they entered an invalid resolution, give them a verbose description of what a resolution should look like.
1. Display a specific amount of top results from the search.
1. Once the user selects a result, the full RSS information should be displayed.

**Caveats**
If the anime is a movie, it should avoid the RSS feed prompts.

### Update Anime

Should allow users to manually update their anime progress. Sometimes there
will be edge cases where the program may not be able to pick something up
without a little help. (after this port, that should be very rare)

- List available anime from library
- Choice to update with name and episode number
  - Can also choose to just update the episode progress
- Choice to confirm the changes about to be made

### Drop

1. Lookup an anime saved in cache
1. Confirmation to drop anime

### RSS Lookup

1. Choose optional Fansub and Resolution.
1. Provide input for manual entry
1. Display basic results
1. Allow selection for full details (or on hover)

### Maintenance

**Cache**
Should allow users to view or rebuild their cache info. This is useful if the
user needs to make manual changes to their kitsu library and have those changes
reflected in Koshime.

- Viewing the cache should display all stored information about the anime in a users library
- Rebuilding the cache should replace the users currently stored anime library with the one from Kitsu. Incremental updates don't make sense on such a small amount of data.

**Directory**
Should allow users to manage their watch directory. The watch directory will get
very full and should be cleaned out regularly.

- Choice to view directory stats (size, oldest, latest, largest, etc...)
- Choice to adjust the interval of when files are deleted (keep most recent)
- Choice to delete entire contents
- Choice to keep the most recent episode of each anime
