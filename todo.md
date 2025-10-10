### File Binding

When a series is completed, we need to delete any existing file bindings

### Rating

When an anime is completed, the user should be prompted to set a rating
for the anime.

- None
- 1, 1.5, 2, 2.5, etc...

### Custom List Component

I'm not a huge fan of the complexity of the list component and I think it would
be more efficient to create my own, for my own specific use case. The display
options are very limited.

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
