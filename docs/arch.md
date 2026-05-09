# Architecture

## Package Structure

```
internal/
├── ui/
│   ├── flows/
│   │   ├── dashboard/          # root flow orchestrator; owns watcher lifecycle
│   │   │   ├── dashboard.go    # Model: watcher init, startup → project transition
│   │   │   └── project/        # project sub-flow (post-load state machine)
│   │   │       ├── project.go  # project.Model; delegates to active State
│   │   │       └── states/     # State interface + concrete states (State pattern)
│   │   │           ├── state.go
│   │   │           ├── idle.go # Idle — placeholder until Dashboard is implemented
│   │   │           └── msgs.go
│   │   └── startup/            # startup flow: scanning → watching / selecting → parsing
│   │       ├── startup.go
│   │       └── states/         # State interface + concrete states (State pattern)
│   │           ├── scanning.go
│   │           ├── watching.go
│   │           ├── selecting.go
│   │           ├── parsing.go
│   │           └── msgs.go     # scanDoneMsg, parseDoneMsg, ScanCmd, ParseCmd
│   ├── views/
│   │   ├── fileselect/         # file picker view (Project Selector)
│   │   └── watching/           # Watching and Disconnected waiting screen
│   └── components/             # shared UI primitives (spinner, key help bar, etc.)
├── compose/
│   └── parser.go               # ScanAll(), Validate(), Parse() — file discovery and parsing
├── watcher/
│   └── watcher.go              # *Watcher concrete type (fsnotify wrapper)
│                               # NullWatcher + Watcher interface: see ADR-0006 (Proposed)
│                               # middleware sub-package: see ADR-0009 (Proposed)
├── msgs/
│   └── msgs.go                 # all inter-component tea.Msg types (no logic, types only)
├── docker/                     # future: Docker daemon client
└── tools/
    └── docgen/                 # CLI documentation generation tooling
```

## Dependency Graph

```
cmd → ui/flows/dashboard
ui/flows/dashboard → ui/flows/startup, ui/flows/dashboard/project, msgs, watcher
ui/flows/startup → ui/flows/startup/states, msgs
ui/flows/startup/states → ui/views/watching, ui/views/fileselect, msgs, compose
ui/flows/dashboard/project → ui/flows/dashboard/project/states, msgs, compose
ui/views/* → msgs, ui/components
watcher → msgs, compose
compose → (stdlib + yaml)
msgs → compose
```

No circular imports are possible with this layout.
