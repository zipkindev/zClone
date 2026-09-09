---
title: "GUI"
description: "Web based Graphical User Interface"
versionIntroduced: "v1.49"
---

# zclone gui

The `zclone gui` command starts the official Web GUI that comes
bundled with zclone.

With this command, zclone can serve a web-based GUI (graphical user
interface) that is accessible from a normal web browser.

Run it in a terminal and zclone will initialize and then start the
GUI.

```console
zclone gui
```

The terminal window needs to stay open to continue to run the GUI. It will log
separate loopback URLs for the RC API and local control page:

```console
2026/04/14 11:36:04 NOTICE: Serving remote control on http://127.0.0.1:50803/
2026/04/14 11:36:04 NOTICE: Serving GUI on http://127.0.0.1:50802/
```

You can also add debugging flags when running the GUI, such as `-v`,
which will show more logging output from the rc server.

## Using the GUI

The bundled local control page submits RC API commands. Enter the RC username
and password locally in the page, choose an RC command such as `core/version`,
and run it. Credentials are used only for that browser request: they are not
written to the URL, persistent browser storage, or Zclone logs.

The bundled page is intentionally local and minimal. It does not check for
updates, collect telemetry, fetch external assets, or contact a central web
service.

## How it works

When you run `zclone gui` this is what happens

- Zclone starts the remote control API ("rc").
- Zclone starts a second server to serve the Web GUI.
- Authentication requires `--pass` unless `--no-auth` is explicitly set.
- If a username is not specified, Zclone uses `gui`.
- Unless `--no-open-browser` is passed, a browser window will open.
- The URL contains no credentials.

## Security

It's important to think first about what zclone has access to and what
you might be sharing.

A few good measures:

- Don't use `--no-auth` except on a trusted local network.
- Do not expose to the local network (eg with `--api-addr :5572 --addr
  :8080`) unless you trust all devices on your local network. Prefer
  `127.0.0.1` or `localhost` (the default).
- Use a strong password and non-obvious usernames like "admin" or
  "zclone" if you are using `--user` and `--pass`.
- If you expose the GUI beyond loopback, configure a trusted reverse proxy,
  TLS, and an explicit `--rc-allow-origin`. Do not expose the RC API directly.

## Options

```console
      --addr stringArray       IPaddress:Port for the GUI server (default auto-chosen localhost port)
      --api-addr stringArray   IPaddress:Port for the RC API server (default auto-chosen localhost port)
      --enable-metrics         Enable OpenMetrics/Prometheus compatible endpoint at /metrics
  -h, --help                   help for gui
      --no-auth                Don't require auth for the RC API
      --no-open-browser        Skip opening the browser automatically
      --pass string            Password for RC authentication
      --user string            User name for RC authentication
```

The GUI bundle ships as a compressed zip embedded in the Zclone binary and is
served from the zip at runtime.
