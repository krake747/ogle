# Architecture

## Package Structure

```
internal/
├── app/
│   └── app.go              # root Bubble Tea model + state machine; delegates to active screen
├── compose/
│   └── parser.go           # ScanAll(), Validate(), Parse() — file discovery and parsing
├── watcher/
│   ├── watcher.go          # Watcher interface + fsWatcher concrete type (fsnotify wrapper)
│   ├── null.go             # NullWatcher — Null Object; used when watcher.New fails
│   └── middleware/
│       └── logging.go      # LoggingWatcher decorator
├── msgs/
│   └── msgs.go             # all inter-component tea.Msg types (no logic, types only)
├── docker/                 # future: Docker daemon client
├── ui/
│   ├── flows/
│   │   └── startup/        # startup flow: orchestrates watching → picker → dashboard
│   │       └── states/     # State interface + concrete state objects (State pattern)
│   ├── views/
│   │   ├── fileselect/     # file picker view (Project Selector)
│   │   ├── watching/       # Watching and Disconnected waiting screen
│   │   └── dashboard/      # main monitoring view
│   └── components/         # shared UI primitives: spinner, key help bar, status indicators
└── tools/
    └── docgen/             # CLI documentation generation tooling
```

## Dependency Graph

```
cmd → app
app → ui/flows/startup, msgs, watcher, watcher/middleware
ui/flows/startup → ui/flows/startup/states, msgs
ui/flows/startup/states → ui/views/watching, ui/views/fileselect, msgs, compose
ui/views/* → msgs, ui/components
watcher → msgs, compose
watcher/middleware → watcher, msgs
compose → (stdlib + yaml)
msgs → compose
```

No circular imports are possible with this layout.
