# Slack-Mod

Inject __JS__ and __CSS__ into the Official Slack Desktop Client. Works as a loader over the standard client.

## Build

### Windows

```sh
go build -ldflags "-s -w -H=windowsgui" -o slack-mod.exe
```

### macOS

```sh
go build -ldflags "-s -w" -o slack-mod
```

### Linux

```sh
go build -ldflags "-s -w" -o slack-mod
```

The macOS and Linux binaries run cleanly in the background when launched from the terminal, so no extra console window is created. If you prefer to start them detached from the shell session, use `nohup` or append `&` when launching.
