# Flows

Documents the state machines, screen states, and transition logic for the TUI.

---

## Overview

### With `-f <path>` (Explicit File)

```
-f given
├── path is a directory          → hard exit: "path is a directory, expected a file"
├── file does not exist          → hard exit: "file not found: <path>"
├── file fails Validate()        → hard exit: "invalid compose file: <error>"
└── valid                        → dashboard
```

Hard exits happen in `cmd/root.go` before the TUI is initialised.

### Without `-f` (File Discovery)

```
no -f
└── ScanAll(CWD) + Validate() each candidate
    ├── 0 valid files            → Watching screen (cold start)
    ├── 1 valid file             → dashboard
    └── 2+ valid files           → Project Selector → dashboard
```

Validity requires both conditions: file exists on disk **and** parses as valid compose YAML.

### Runtime: file disappears (Disconnected)

```
dashboard → watched file deleted or moved
└── Watching screen ("disconnected — waiting for <filename>")
    └── watches for the SAME filename to reappear
        └── file reappears + valid   → dashboard
```

### Watching screen: file appears (cold start)

```
fsnotify event (create/write in CWD)
└── re-run ScanAll() + Validate()
    ├── 0 valid  → stay on Watching screen
    ├── 1 valid  → dashboard
    └── 2+ valid → Project Selector → dashboard
```

---

## Watcher Lifetime

The watcher is created at app startup and runs for the entire process lifetime — including while the dashboard is active. `dashboard.Init()` starts `watcher.Next()` and the root orchestrator re-subscribes after every `FileAvailabilityChanged` by returning another `watcher.Next()` Cmd from `Update`. The active sub-model (startup flow or project) receives the message via the root orchestrator's dispatch logic.

---

## Root Orchestrator (`internal/ui/flows/dashboard`)

```
startup    — startup flow is the active sub-model
project    — project sub-model is active (post ProjectLoaded)
```

### Init (two Cmds in parallel)

```
dashboard.Init()
├── current.Init()    → kicks off scan (or immediate parse for -f case)
└── watcher.Next()    → begins perpetual watcher subscription
```

If `-f` was given (already validated in `cmd/root.go`), the initial scan is skipped and the startup flow goes directly to `Parsing` with the provided path.

### FileAvailabilityChanged dispatch

```
FileAvailabilityChanged received
├── startup active   → re-subscribe watcher, forward to startup flow
└── project active   → re-subscribe watcher, forward to project sub-model
```

---

## Startup Flow (`internal/ui/flows/startup`)

```
Scanning   — Init: ScanAll + Validate Cmds in flight; no screen rendered
Watching   — Watching view active
Selecting  — fileselect view active (Project Selector)
Parsing    — Parse Cmd in flight; invisible; current view held (no UI change)
```

### From Scanning

```
ScanAll + Validate results
├── 0 valid files   → Watching
├── 1 valid file    → Parsing
└── 2+ valid files  → Selecting
```

### From Watching (FileAvailabilityChanged)

```
FileAvailabilityChanged{Files}
├── 0 valid after Validate   → stay in Watching
├── 1 valid after Validate   → Parsing
└── 2+ valid after Validate  → Selecting
```

### From Selecting (user confirms file)

```
FileSelected{Path}
└── Parsing
```

### From Parsing

```
Parse result
├── success                → emit ProjectLoaded{Project} → app transitions to appDashboard
├── failure (parse error)  → Watching (notice on sub-model)
│                          or Selecting (error on sub-model)
└── failure (read error)   → return to display state; watcher will correct
```

---

## Watching View (`internal/ui/views/watching`)

Rendered by the startup flow in the `Watching` state. Also used when the dashboard transitions to the Disconnected state (file disappeared at runtime).

```
watchingIdle    — monitoring CWD; no valid files present
watchingNotice  — a file appeared but failed Validate (exists, invalid YAML)
                  transient inline message: "compose.yaml found but could not be parsed"
                  cleared automatically on the next FileAvailabilityChanged
watchingError   — watcher failed to initialise (permissions, missing CWD, etc.)
                  shows error message + retry keybinding; recoverable
```

### Cold start vs. Disconnected

The watching view accepts a mode that controls the message shown:

| Mode           | Heading                                   |
|----------------|-------------------------------------------|
| `cold`         | Watching for a compose file…              |
| `disconnected` | Disconnected — waiting for `<filename>`   |

In disconnected mode, `FileAvailabilityChanged` is only acted on if the specific filename that was being monitored is present in `Files`.

---

## Fileselect View (`internal/ui/views/fileselect`)

Rendered by the startup flow in the `Selecting` state (Project Selector).

```
fileselectBrowsing  — list of valid files, cursor navigating
fileselectError     — Parse failed for the confirmed selection
                      (file was valid at list time, broken by the time Parse ran)
                      inline notice beneath the list; list remains active
```

On a new `FileAvailabilityChanged` the list is refreshed. If the previously errored file is no longer present, the error notice is cleared.

---

## Dashboard (`internal/ui/flows/dashboard/project`)

Entered after the startup flow emits `ProjectLoaded{Project}`.

```
dashboardLoaded     — monitoring services; project is current
dashboardReloading  — compose file changed, Parse Cmd in flight
                      invisible (no UI change); state exists for correctness
dashboardParseError — live reload failed; compose file has syntax errors
                      persistent notice banner over the last-known service list
                      cleared automatically when the next Parse succeeds
```

### FileAvailabilityChanged on dashboard

```
FileAvailabilityChanged{Files}
├── project file NOT in Files   → app transitions to appStartup (disconnected Watching)
└── project file in Files       → dashboardReloading → Parse Cmd
    ├── Parse success            → dashboardLoaded (service list updated silently)
    └── Parse failure            → dashboardParseError (banner; last-known state preserved)
```

---

## Runtime: file disappears (full trace)

```
dashboardLoaded
└── FileAvailabilityChanged{Files} where project file ∉ Files
    └── app → appStartup
        └── startup flow → Watching (disconnected mode)
            └── watches for the SAME filename to reappear
                └── file reappears + valid → Parsing → appDashboard
```

---

## Message Types (`internal/msgs`)

| Message                          | Emitted by          | Consumed by                                      |
|----------------------------------|---------------------|--------------------------------------------------|
| `FileAvailabilityChanged{Files}` | `watcher`           | root orchestrator → startup flow or project      |
| `FileSelected{Path}`             | fileselect view     | startup flow                                     |
| `ProjectLoaded{Project}`         | startup flow        | root orchestrator                                |
| `WatcherError{Err}`              | root orchestrator   | startup flow → watching view (error state)       |
| `RetryWatcher{}`                 | watching view       | root orchestrator (triggers watcher re-creation) |
| `DaemonConnected{}`              | docker service      | Dashboard (transitions to ConnectStateConnected) |
| `DaemonUnavailable{Err}`         | docker service      | Dashboard (starts retry countdown)               |
