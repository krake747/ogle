// Package layout defines shared layout policy constants for the ogle TUI.
// These are cross-cutting constants used by both the app chrome and phase
// components to agree on how much terminal space is consumed by chrome.
package layout

// FrameHeight is the number of terminal lines consumed by the app-level
// chrome (topbar + helpbar + status bar) that phase components must subtract
// from raw terminal height to determine their usable body area.
const FrameHeight = 3
