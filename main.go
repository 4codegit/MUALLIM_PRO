// eDonish Auto — Modern desktop application (Go + Fyne UI)
// Automated grade management for edonish.tj
package main

import (
	"os"
	"runtime"

	"github.com/4codegit/edonish-auto/internal/ui"
)

// main - точка входа в приложение
func main() {
	// На Windows принудительно используем software rendering
	// для избежания крашей OpenGL драйверов
	if runtime.GOOS == "windows" && os.Getenv("FYNE_RENDER") == "" {
		os.Setenv("FYNE_RENDER", "software")
	}

	// Создаём и запускаем приложение
	app := ui.NewApp()
	app.Run()
}
