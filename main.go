// eDonish Auto — Modern desktop application (Go + Fyne UI)
// Automated grade management for edonish.tj
package main

import (
	"os"
	"runtime"

	"github.com/4codegit/edonish-auto/internal/ui"
)

func main() {
	// On Windows, force software rendering to avoid OpenGL driver crashes.
	// Many Windows systems (especially VMs, older hardware, or machines
	// with missing GPU drivers) don't have proper OpenGL support, which
	// causes Fyne to crash on startup. Software rendering is slower but
	// universally compatible.
	//
	// On Linux/macOS, OpenGL is typically available via Mesa/native drivers,
	// so we keep hardware rendering unless the user explicitly sets
	// FYNE_RENDER=software in their environment.
	if runtime.GOOS == "windows" && os.Getenv("FYNE_RENDER") == "" {
		os.Setenv("FYNE_RENDER", "software")
	}

	app := ui.NewApp()
	app.Run()
}
