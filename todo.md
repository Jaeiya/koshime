### Minor Issues

1.) When escaping from the maintenance menu, the main menu resets the menu position 
2.) When searching the library, the display view toggle doesn't work as expected

### Auto Renew Token

Check if token has some `n` expiration time and execute renew process
automatically. It should be somewhere within **1 week** left. 

If token has just been auto-renewed, update the profile with that info.

### Profile Updates

After each anime, technically the users profile has been updated, but we
don't need to do this for every anime watched. It should be on an
interval. If it has been `n` time since last update, then we update
the profile info.

### Manual RSS Copy

Allow copying the RSS feed to the clipboard from a manual search

### Rating

When an anime is completed, the user should be prompted to set a rating
for the anime.

- None
- 1, 1.5, 2, 2.5, etc...

### Custom List Component

I'm not a huge fan of the complexity of the list component and I think it would
be more efficient to create my own, for my own specific use case. The display
options are very limited.


### Update Anime

Should allow users to manually update their anime progress. Sometimes there
will be edge cases where the program may not be able to pick something up
without a little help. (after this port, that should be very rare)

- List available anime from library
- Choice to update with name and episode number
  - Can also choose to just update the episode progress
- Choice to confirm the changes about to be made
