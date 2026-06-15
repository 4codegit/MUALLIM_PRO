package main

import (
        "os"
        "runtime"
)

func main() {
        // Force software rendering to avoid OpenGL driver crashes
        // Works across Windows, Linux, and older macOS
        if os.Getenv("FYNE_RENDER") == "" {
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
