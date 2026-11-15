package main

import (
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"log"
	"os"
	"wwhsrt/pkg/app"
)

// version is set by release builds: go build -ldflags "-X main.version=0.1.0".
var version = "dev"

func main() {
	// Packages check the installed binary with --version, which needs no display or save directory.
	if len(os.Args) == 2 && (os.Args[1] == "--version" || os.Args[1] == "-version") {
		fmt.Println("wwhsrt", version)
		return
	}
	port, cleanup, e := app.OpenHost()
	if e != nil {
		app.StartupError(e)
		return
	}
	defer cleanup()
	g, e := app.New(port)
	if e != nil {
		app.StartupError(e)
		return
	}
	ebiten.SetWindowSizeLimits(320, 480, -1, -1)
	ebiten.SetWindowTitle("World-wide High-speed Rail Tycoon")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	g.ConfigureWindow()
	defer g.FlushWindowPreferences()
	// Linux desktops match a window to its .desktop file by WM_CLASS, so use the application ID.
	options := &ebiten.RunGameOptions{X11ClassName: "io.github.adsouza.wwhsrt", X11InstanceName: "wwhsrt"}
	if e = ebiten.RunGameWithOptions(g, options); e != nil {
		log.Print(e)
	}
}
