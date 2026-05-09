# ogle

A terminal UI for observing and operating Docker Compose projects — no setup required.

## Language

### Compose file concepts

**Compose File**:
The YAML file on disk that defines a Docker Compose project (e.g., `compose.yaml`, `docker-compose.yml`).
_Avoid_: Project file, compose config

**Project**:
The parsed, named in-memory representation of a Compose File, including its name and list of declared Services.
_Avoid_: Compose project, workspace

**Service**:
A named unit declared in the Compose File that maps to a running (or stopped) container. Spans both the declaration and its runtime instance.
_Avoid_: Container (reserve "container" for implementation-level discussion, e.g., container ID or Docker API calls)

**Service State**:
The current Docker container state for a Service: `running`, `exited`, `paused`, `restarting`, `dead`, or `not created` (declared in the Compose File but no container exists yet).
_Avoid_: Status, health (health check result is separate from container state)

**Orphan**:
A running container that has no corresponding Service in the current Project. Typically the result of removing a Service from the Compose File while its container is still running.
_Avoid_: Orphan container (redundant — all Orphans are containers by definition)

**Orphan Toggle**:
The user action that shows or hides Orphans in the service list on the Dashboard.
_Avoid_: Show orphans, orphan visibility

### User interaction

**Dashboard**:
The main screen displayed after a Project is loaded. Shows all Services, their states, and the Selected Service's Log Stream.
_Avoid_: Monitor view, main view, service view

**Selected Service**:
The Service whose Log Stream is currently displayed in the Dashboard.
_Avoid_: Active service, focused service

**Log Stream**:
The live, tailing output of a Selected Service's logs, streamed in real time from Docker.
_Avoid_: Logs (too generic), log tail

**Log Buffer**:
The bounded in-memory store of log lines for the Selected Service. Capped at a configurable maximum; when exceeded, the oldest lines are discarded to maintain performance.
_Avoid_: Log history, log cache

**State Polling**:
The background process that periodically queries Docker for the current Service State of each Service. Runs on a configurable interval.
_Avoid_: Polling (unqualified), refresh, state sync

**Service Filter**:
The interactive mode that narrows the Service list to entries whose name matches a user-supplied substring. Activated with `/` on the Dashboard.
_Avoid_: Log filter (distinct feature), search

**Log Filter**:
The interactive mode that narrows the Log Stream to lines matching a user-supplied substring. Planned feature; not yet implemented.
_Avoid_: Log search, service filter (distinct feature)

**Service Action**:
A user-initiated operation applied to a Service: stop, start, restart, or rebuild. Executed asynchronously without blocking the UI.
_Avoid_: Command (overloaded in the Bubble Tea runtime context), operation

**Settings**:
An in-session overlay that lets the user adjust configuration values (e.g., poll interval, log buffer cap) without leaving the TUI or editing the Config File. Changes take effect for the current session; persistence to the Config File is a separate action.
_Avoid_: Config, preferences — also distinct from instant keybinding toggles (e.g., Orphan Toggle) which take effect immediately with no overlay

### Startup

**File Discovery**:
The startup process of scanning the working directory for Compose Files, validating each candidate, and selecting one to load as a Project.
_Avoid_: Scanning, auto-detection, file search

**Explicit File**:
The startup mode where the user provides the Compose File path directly (via `-f`), bypassing File Discovery entirely. Validation failures in this mode are hard exits before the TUI opens.
_Avoid_: Manual file, specified file

**Watching**:
The startup state where File Discovery found no valid Compose Files and ogle is monitoring the working directory for one to appear.
_Avoid_: Waiting, idle — also distinct from **Disconnected**, which waits for a *specific* file to return

**Watcher Error**:
A recoverable failure state where ogle cannot monitor the working directory (e.g., permissions problem, directory missing). The user is shown an error message and must explicitly retry.
_Avoid_: Watcher failure, monitor error

**Project Selector**:
The screen shown during startup when File Discovery finds two or more valid Compose Files. Lets the user choose which Project to load.
_Avoid_: File picker, file selector

**Live Reload**:
The automatic re-parse of the Compose File and silent update of the Dashboard when the Compose File changes on disk, without interrupting the session.
_Avoid_: Hot reload, refresh, file sync

**Parse Error**:
The condition where the Compose File exists on disk but cannot be successfully parsed into a Project. At startup, shown inline on the current screen; at runtime (during Live Reload), shown as a persistent banner over the Dashboard with the last-known state preserved.
_Avoid_: Invalid file, broken compose, YAML error

**Disconnected**:
The state ogle enters when the monitored Compose File disappears at runtime. ogle waits for the same file to reappear before resuming the Dashboard.
_Avoid_: Offline, paused, suspended

## Relationships

- A **Project** declares one or more **Services**.
- **File Discovery** finds the **Compose File** and parses it into a **Project**. If no valid Compose File is found, ogle enters the **Watching** state.
- When 2+ valid Compose Files are found, the **Project Selector** lets the user choose which to load.
- The **Dashboard** displays all Services and the **Selected Service**'s **Log Stream**.
- **State Polling** periodically updates each Service's **Service State**.
- A user triggers a **Service Action** on a Service from the Dashboard; actions run asynchronously and do not block the UI.
- When the Compose File changes on disk, **Live Reload** updates the Project without leaving the Dashboard.
- When the Compose File disappears at runtime, the Dashboard enters the **Disconnected** state and waits for that specific file to reappear.
- An **Orphan** appears alongside Services in the Dashboard but is not part of the Project. The **Orphan Toggle** controls whether Orphans are shown.

## Example dialogue

> **Dev:** "If the user has two compose files, which Project do they load?"
> **Domain expert:** "The **Project Selector** appears. They pick one and the **Dashboard** opens."

> **Dev:** "What happens if they edit the compose file while the Dashboard is open?"
> **Domain expert:** "**Live Reload** — the Dashboard re-parses it silently. If the file disappears entirely, the Dashboard goes **Disconnected** and waits for that specific file to come back."

> **Dev:** "Can you stop a service from the Dashboard?"
> **Domain expert:** "Yes — trigger a **Service Action** (stop). It runs in the background; the UI stays responsive. **State Polling** picks up the state change in the next poll cycle."

> **Dev:** "There's a container running that isn't in the compose file anymore — what is it?"
> **Domain expert:** "An **Orphan**. It shows up in the Dashboard alongside the Services but it's not part of the **Project**."

## Flagged ambiguities

- **"Zero configuration"** was initially proposed in the aim statement. Resolved: ogle supports optional configuration via config files and environment variables, so the accurate claim is *"no setup required"* — it works out of the box but can be configured.
- **"Container" vs "Service"** — ogle uses **Service** as the user-facing term that spans both the Compose File declaration and its runtime container. "Container" is reserved for implementation-level precision (e.g., when targeting a specific container ID for log streaming or Docker API calls).
