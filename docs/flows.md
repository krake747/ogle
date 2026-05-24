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

The watcher is created at app startup and runs for the entire process lifetime — including while the dashboard is active. `app.Init()` starts `watcher.Next()` and the app re-subscribes after every `FileAvailabilityChanged` by returning another `watcher.Next()` Cmd from `Update`. The active sub-model (startup flow or dashboard) receives the message via `app.Update`'s dispatch logic.

---

## Root Orchestrator (`internal/app/app.go`)

The app manages three phases:

```
appStartup    — startup flow is the active sub-model
appDashboard  — dashboard flow is active (post-ProjectLoaded)
appWatching   — watching flow is active (disconnected, waiting for file to reappear)
```

### Init (two Cmds in parallel)

```
app.Init()
├── watcher.Next()                → begins perpetual watcher subscription
└── startup.Init() (or direct)    → kicks off scan (or immediate parse for -f case)
```

If `-f` was given (already validated in `cmd/root.go`), the initial scan is skipped and the startup flow receives the path directly.

### Message dispatch

```
app.Update(msg)
├── msgs.ProjectLoaded           → transition startup → dashboard
├── msgs.FileAvailabilityChanged → re-subscribe watcher, dispatch to startup or dashboard
├── tea.WindowSizeMsg            → forward to active sub-model
├── theme.Changed                → update pointer, forward to active sub-model
├── msgs.SettingsApplied         → update config, forward to dashboard
└── other msgs                   → forward to active sub-model
```

---

## Startup Flow (`internal/ui/flows/startup`)

A simple model (82 lines, no State pattern). Key behaviour:

- On `msgs.FileSelected`: parse the selected file, emit `msgs.ProjectLoaded`
- On `tea.WindowSizeMsg`: forward to fileSelect sub-model
- All other messages: forward to fileSelect sub-model

The startup flow does not own scan/validate logic — those happen via `scanner.ScanAll()` and `parser.Validate()` in the watching/fileselect components before a `FileSelected` msg reaches this flow.

---

## Watching View (`internal/ui/components/watching`)

Rendered by the app's `appWatching` phase. Also used when the dashboard transitions to the Disconnected state (file disappeared at runtime).

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

## Fileselect View (`internal/ui/components/fileselect`)

Rendered by the startup flow in the `Selecting` state (Project Selector).

```
fileselectBrowsing  — list of valid files, cursor navigating
fileselectError     — Parse failed for the confirmed selection
                      (file was valid at list time, broken by the time Parse ran)
                      inline notice beneath the list; list remains active
```

On a new `FileAvailabilityChanged` the list is refreshed. If the previously errored file is no longer present, the error notice is cleared.

---

## Dashboard (`internal/ui/flows/dashboard`)

Entered after `app` receives `ProjectLoaded{Project}`.

The dashboard is a flat model (no sub-states). It:

- Dispatches `StatePollTick` to the service panel and emits a `docker.Ps()` Cmd
- Routes `ServiceStop/Start/Restart/Rebuild/ActionCompleted` to `handleServiceAction`
- Handles `FileAvailabilityChanged` — if the project file is still present, re-parses and updates; if absent, sends a msg that triggers `app` to transition to `appWatching`
- Forwards all messages to its sub-models (accordion, carousel, panel, settings)
- Toggles settings overlay via `SettingsVisibilityChanged`

---

## Message Types (`internal/msgs`)

| Message                          | Emitted by                      | Consumed by                                 |
|----------------------------------|---------------------------------|---------------------------------------------|
| `FileAvailabilityChanged{Files}` | `watcher`                       | `app` (dispatches to startup/dashboard)     |
| `FileSelected{Path}`             | fileselect                      | startup                                     |
| `ProjectLoaded{Project}`         | startup / watching              | `app` (triggers appDashboard)               |
| `WatcherError{Err}`              | `watcher`                       | `app` → watching view                       |
| `RetryWatcher{}`                 | watching view                   | `app` (triggers watcher re-creation)        |
| `DaemonConnected{}`              | `svcdocker.Connect`             | `topbar`, `servicepanel`, `servicehost`     |
| `DaemonUnavailable{Err}`         | `svcdocker.Connect`             | `topbar` (starts retry countdown)           |
| `StatePollTick`                  | `servicepanel` (timer)          | `dashboard` (triggers `docker.Ps`)          |
| `ServicesPolled{Runtimes}`       | `docker.Ps`                     | `dashboard`, `carousel`, `accordion`        |
| `ServiceStop`                    | `carousel/card` (user action)   | `dashboard` → `handleServiceAction`         |
| `ServiceStart`                   | `carousel/card` (user action)   | `dashboard` → `handleServiceAction`         |
| `ServiceRestart`                 | `carousel/card` (user action)   | `dashboard` → `handleServiceAction`         |
| `ServiceRebuild`                 | `carousel/card` (user action)   | `dashboard` → `handleServiceAction`         |
| `ServiceActionCompleted`         | `svcdocker`                     | `dashboard`, `carousel/card`                |
| `LogLinesAvailable{Lines}`       | `LogStreamer`                   | `logpane` (via `servicehost`)               |
| `LogStreamError{Err}`            | `LogStreamer`                   | `servicehost` (re-subscribes streamer)      |
| `LogStreamContainerNotFound`     | `LogStreamer`                   | `servicehost` (re-subscribes streamer)      |
| `ServiceSelected{ServiceName}`   | `carousel` (hover/focus)        | `dashboard`, `accordion`, `servicehost`     |
| `SettingsApplied{Theme,LBCap}`   | `settings`                      | `dashboard`                                 |
| `SettingsVisibilityChanged`      | `settings`                      | `dashboard`                                 |
| `ToggleLogWrap`                  | `dashboard` (keybinding)        | `logpane`                                   |
| `BindingsMsg{Keymap}`            | various flows                   | `helpbar`                                   |
| `theme.Changed`                  | external (theme switcher)       | all components with theme pointer           |

---

## Runtime: file disappears (full trace)

```
dashboard (appDashboard)
└── FileAvailabilityChanged{Files} where project file ∉ Files
    └── app → appWatching
        └── watching view (disconnected mode)
            └── watches for the SAME filename to reappear
                └── file reappears + valid → Parsing → appDashboard
```
