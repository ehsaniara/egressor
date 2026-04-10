//go:build !darwin || notray

package tray

// Callbacks holds the functions the tray uses to control the proxy.
type Callbacks struct {
	OnStart       func()
	OnPauseToggle func(paused bool)
	OnQuit        func()
}

// Register is a no-op on unsupported platforms.
func Register(_ Callbacks) {}

// Available returns false when the systray is not compiled in.
func Available() bool {
	return false
}
