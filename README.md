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
@DuckBot leaderboard emoji :high_heel:
@DuckBot leaderboard emoji reset
```

Private and unconfigured guilds return HTTP 404 from leaderboard API routes.
The first-ranked member uses a configurable emoji that defaults to `:high_heel:`.

Managers, server administrators, and Mainduck can delete a non-user leaderboard item
with a persisted five-minute confirmation:

```text
@DuckBot leaderboard delete ITEM
@DuckBot leaderboard confirm-delete
@DuckBot leaderboard cancel-delete
```

## Production deployment

DuckBot must run with exactly one App Service worker because every connected process
receives Discord gateway events. The ARM template fixes the App Service Plan capacity
at one worker. Main builds publish both a versioned container and a convenience
`latest` tag. Deployments must use the generated `deploy.parameters.json`, which pins
the exact versioned image produced by that build.
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

## Guild manager role

A server administrator or Mainduck can designate one role whose members may manage
DuckBot's server-specific settings:

```text
@DuckBot manager @Role
@DuckBot manager status
@DuckBot manager reset
```

The configured manager role can change counter names and leaderboard visibility. It
cannot designate or reset the manager role; that remains restricted to server
administrators and Mainduck.
