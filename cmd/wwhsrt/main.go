package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"log"
	"wwhsrt/pkg/app"
)

func main() {
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
	if e = ebiten.RunGame(g); e != nil {
		log.Print(e)
	}
}
