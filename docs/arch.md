# Architecture

## Package Structure

```text
internal/
├── app/                              # root flow orchestrator (owns watcher lifecycle)
│   └── app.go
├── domain/                           # canonical domain types (Project, ServiceDef, ServiceState, etc.)
│   └── domain.go
├── msgs/                             # all inter-component tea.Msg types (no logic, types only)
│   └── msgs.go
├── profiling/                        # profiling utilities
├── services/
│   ├── docker/                       # Docker daemon connectivity
│   │   ├── service.go                # Docker interface + Service (connect, ps, stop, start, restart, rebuild)
│   │   ├── actions.go                # service action implementations (docker compose CLI)
│   │   ├── ps.go                     # state polling (docker compose ps --format json)
│   │   ├── connection/               # connection state machine (Machine: Connecting/Connected/Unavailable)
│   │   └── logs/                     # LogStreamer: Docker log streaming
│   │       ├── streamer.go           # Streamer interface
│   │       ├── service.go            # LogStreamer implementation
│   │       └── null.go               # NullLogStreamer (no-op adapter for unselected services)
│   ├── parser/                       # Compose File parsing
│   │   └── service.go                # Parser interface, Validate(), Parse()
│   ├── scanner/                      # Compose File discovery
│   │   └── service.go                # Scanner interface, ScanAll(), KnownFilenames()
│   └── watcher/                      # directory monitoring via fsnotify
│       ├── service.go                # Watcher interface + Service implementation
│       └── null.go                   # NullWatcher (never emits events)
├── tools/
│   └── docgen/                       # CLI documentation generation
└── ui/
    ├── colorutil/                    # colour utilities (Brighten)
    ├── components/
    │   ├── accordion/                # Service Inspector — detail header (name, image, ports, state, age)
    │   │   └── value/                # scroll-animated value display
    │   ├── carousel/                 # service card grid with pagination, focus, hover
    │   │   ├── carousel.go
    │   │   ├── keymap.go
    │   │   └── card/                 # single service card (state colouring, actions)
    │   ├── fileselect/               # Project Selector (file picker)
    │   ├── helpbar/                  # key binding help bar
    │   ├── logpane/                  # log stream display (scroll, wrap)
    │   ├── servicehost/              # log stream lifecycle manager
    │   ├── servicepanel/             # service detail panel wrapper
    │   ├── settings/                 # Settings overlay
    │   ├── statusbar/                # status bar (info/error messages, auto-clear)
    │   ├── topbar/                   # top bar (brand, context, daemon status)
    │   └── watching/                 # Watching / Disconnected waiting screen
    ├── flows/
    │   ├── dashboard/                # Dashboard flow (post-ProjectLoaded)
    │   │   ├── dashboard.go          # Model: message dispatch, service actions, file monitoring
    │   │   └── keymap.go             # dashboard-specific key bindings
    │   └── startup/                  # Startup flow (simple model, no state pattern)
    │       ├── startup.go            # Model: receives FileSelected, emits ProjectLoaded
    │       └── keymap.go             # startup-specific key bindings
    ├── hoverlist/                    # reusable hover-highlight list infrastructure
    ├── layout/                       # layout constants (FrameHeight)
    └── theme/                        # Theme type, built-in themes, user theme loading
        ├── theme.go                  # Theme struct, Load(), override logic
        ├── builtin.go                # Default theme (dark)
        ├── default_light.go          # Default theme (light)
        ├── solarized_dark.go
        ├── solarized_light.go
        ├── catppuccino_mocha.go
        ├── catppuccino_macchiato.go
        ├── catppuccino_frappe.go
        └── catppuccino_latte.go
```

## Dependency Graph

```text
cmd → app
app → ui/flows/startup, ui/flows/dashboard, msgs, services/watcher, services/docker, config
ui/flows/startup → services/parser, services/scanner, ui/components/fileselect, msgs, ui/theme
ui/flows/dashboard → services/parser, services/docker/logs, ui/components/{accordion,carousel,servicepanel,settings}, msgs, ui/theme, config
ui/components/* → msgs, ui/theme, ui/colorutil
services/docker → domain, msgs
services/docker/logs → msgs, domain
services/parser → domain
services/scanner → (stdlib only)
services/watcher → msgs, services/scanner
domain → (stdlib only)
msgs → domain, ui/theme
```

No circular imports are possible with this layout. All arrows point infrastructure → domain, never the reverse.
