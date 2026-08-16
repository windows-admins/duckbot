# DuckBot3
Build Status: [![Build Status](https://dev.azure.com/duckbot3/duckbot3/_apis/build/status/windows-admins.duckbot?branchName=main)](https://dev.azure.com/duckbot3/duckbot3/_build/latest?definitionId=1&branchName=main)

## Custom counters

Server administrators and Discord user `281125480072085515` can customize the
singular and plural counter names for their server:

```text
@DuckBot counter Rubber Duck | Rubber Ducks
```

Use `@DuckBot counter` to show the current names or
`@DuckBot counter reset` to restore `Loch Ness Goose` / `Loch Ness Geese`.

## Public leaderboards

Leaderboard API access is private by default. A server administrator or Mainduck can
change the persisted visibility setting:

```text
@DuckBot leaderboard public
@DuckBot leaderboard private
@DuckBot leaderboard status
```

Private and unconfigured guilds return HTTP 404 from leaderboard API routes.
Public leaderboard responses include the Discord guild ID and current guild name:

```json
{
  "guild": {
    "id": "618712310185197588",
    "name": "Windows Admins"
  },
  "items": []
}
```

## Deployment image

Production uses `rowdychildren/duckbot:latest`. Main-branch builds publish both a
versioned image and the `latest` tag. The legacy `rowdychildren/duckbot3` image must
not be used; it has not been updated since 2021 and can point the API at stale data.
