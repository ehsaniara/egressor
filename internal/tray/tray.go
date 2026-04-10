//go:build darwin && !notray

package tray

import (
	"github.com/energye/systray"
)

// Callbacks holds the functions the tray uses to control the proxy.
type Callbacks struct {
	OnStart       func()            // called once when tray is ready — start the proxy here
	OnPauseToggle func(paused bool) // called when user toggles pause
	OnQuit        func()            // called when user clicks Quit
}

// Register sets up the systray without starting its own event loop.
// This must be used when another framework (like Wails) already owns
// the main thread and NSApplication run loop on macOS.
func Register(cb Callbacks) {
	systray.Register(func() {
		systray.SetTemplateIcon(iconData, iconData)
		systray.SetTitle("")
		systray.SetTooltip("Egressor — egress monitor")

		// On macOS, left-click doesn't show the menu by default.
		// We must handle it explicitly.
		systray.SetOnClick(func(menu systray.IMenu) {
			_ = menu.ShowMenu()
		})

		mStatus := systray.AddMenuItem("Status: Running", "")
		mStatus.Disable()

		systray.AddSeparator()

		mPause := systray.AddMenuItem("Pause", "Pause policy enforcement")

		systray.AddSeparator()

		mQuit := systray.AddMenuItem("Quit", "Stop Egressor")

		paused := false
		mPause.Click(func() {
			paused = !paused
			if paused {
				mPause.SetTitle("Resume")
				mPause.SetTooltip("Resume policy enforcement")
				mStatus.SetTitle("Status: Paused")
			} else {
				mPause.SetTitle("Pause")
				mPause.SetTooltip("Pause policy enforcement")
				mStatus.SetTitle("Status: Running")
			}
			if cb.OnPauseToggle != nil {
				cb.OnPauseToggle(paused)
			}
		})

		mQuit.Click(func() {
			systray.Quit()
		})

		if cb.OnStart != nil {
			go cb.OnStart()
		}
	}, func() {
		if cb.OnQuit != nil {
			cb.OnQuit()
		}
	})
}

// Available returns true when the systray is compiled in.
func Available() bool {
	return true
}
