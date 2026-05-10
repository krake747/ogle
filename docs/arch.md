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
├── services/
│   ├── scanner/
│   │   └── service.go          # ScanAll(), KnownFilenames() — file discovery (stdlib only)
│   ├── parser/
│   │   └── service.go          # Validate(), Parse() — Compose File parsing; depends on domain
│   └── watcher/
│       ├── service.go          # *Service concrete type (fsnotify wrapper)
│       └── null.go             # NullWatcher: Watcher interface satisfied; never delivers events
│                               # see ADR-0006 (Accepted), ADR-0009 (Proposed)
├── domain/
│   └── domain.go               # canonical domain types: Project, ServiceDef
├── msgs/
│   └── msgs.go                 # all inter-component tea.Msg types (no logic, types only)
└── tools/
    └── docgen/                 # CLI documentation generation tooling
```

## Dependency Graph

```
cmd → ui/flows/dashboard
ui/flows/dashboard → ui/flows/startup, ui/flows/dashboard/project, msgs, services/watcher
ui/flows/startup → ui/flows/startup/states, msgs
ui/flows/startup/states → ui/views/watching, ui/views/fileselect, msgs, services/parser, services/scanner
ui/flows/dashboard/project → ui/flows/dashboard/project/states, msgs, services/parser
ui/views/* → msgs, ui/components
services/watcher → msgs, services/scanner
services/parser → domain
services/scanner → (stdlib only)
domain → (stdlib only)
msgs → domain
```

No circular imports are possible with this layout.
