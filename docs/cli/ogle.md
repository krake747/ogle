## ogle

A TUI for monitoring Docker Compose projects.

```
ogle [flags]
```

### Options

```
      --config string         config file (default is $HOME/.ogle/config)
  -h, --help                  help for ogle
      --log-buffer-cap int    maximum log lines buffered per service (env: OGLE_LOG_BUFFER_CAP; default 1000)
      --pprof-addr string     pprof HTTP server address (e.g. localhost:6060)
  -f, --project-file string   path to docker compose file
      --theme string          theme name; built-ins: "default", "default_light", "catppuccino_frappe", "catppuccino_latte", "catppuccino_macchiato", "catppuccino_mocha", "solarized_dark", "solarized_light" (env: OGLE_THEME)
```

### SEE ALSO

* [ogle version](ogle_version.md)	 - Print version information

