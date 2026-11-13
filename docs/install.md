# Install

You need to be a Mattermost system admin. No SSH needed.

## 1. Allow plugin uploads

System Console → Plugins → Plugin Management → set **Enable Plugin Uploads** to `true`.

On a self-hosted server that's this part of `config.json`:

```json
"PluginSettings": { "Enable": true, "EnableUploads": true }
```

## 2. Upload the file

Click **Upload Plugin** and pick the `.tar.gz` you downloaded from shiftplugins.com. Or from the server:

```sh
mmctl plugin add ./com.shiftplugins.planning-1.5.0.tar.gz
```

With the official Docker setup:

```sh
docker cp com.shiftplugins.planning-1.5.0.tar.gz mattermost:/tmp/
docker exec mattermost mmctl --local plugin add /tmp/com.shiftplugins.planning-1.5.0.tar.gz
```

## 3. Enable it

Click **Enable** next to the plugin, then open its settings page. Or:

```sh
mmctl plugin enable com.shiftplugins.planning
```

Plugin IDs: `com.shiftplugins.planning`, `com.shiftplugins.leave` (Leave requests), `com.shiftplugins.timeclock` (Time clock).

## First steps

- **Planning:** in a channel, type `/planning`, turn it on for the channel and add shifts. Members get a pinned "Open planning" button.
- **Leave requests:** set the managers in the plugin settings, then anyone types `/leave` (or `/conges`, `/ferien`, `/ferie` depending on their language).
- **Time clock:** type `/pointage` in the channel your team should clock in from. The bot pins the Clock in / Clock out buttons.

## Updating

Upload the new `.tar.gz` the same way. Mattermost replaces the old version and keeps all data.

## Uninstalling

Disable and remove the plugin in Plugin Management. Data stays in the database until you remove it, so reinstalling brings everything back.
