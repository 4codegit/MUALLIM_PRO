package main

import (
	"os"
	"runtime"
)

func main() {
	// For Windows, force software rendering to avoid driver OpenGL crashes
	if runtime.GOOS == "windows" && os.Getenv("FYNE_RENDER") == "" {
		os.Setenv("FYNE_RENDER", "software")
	}
	if runtime.GOOS == "windows" && os.Getenv("GALLIUM_DRIVER") == "" {
		os.Setenv("GALLIUM_DRIVER", "llvmpipe")
	}
	if runtime.GOOS == "windows" && os.Getenv("LIBGL_ALWAYS_SOFTWARE") == "" {
		os.Setenv("LIBGL_ALWAYS_SOFTWARE", "1")
	}

	controller := NewAppController()
	controller.Run()
}
