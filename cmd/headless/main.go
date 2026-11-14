//go:build !js

// Headless drives the same application port without initializing graphics/audio.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"wwhsrt/internal/adapters/filestore"
	"wwhsrt/internal/application"
	"wwhsrt/pkg/gameapi"
)

func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
func run() error {
	n := flag.Int("advance", 0, "months to resolve (0–360)")
	name := flag.String("rename", "", "company name")
	path := flag.String("save", "", "optional development snapshot path")
	commands := flag.String("commands", "", "JSON command array to apply through the game port")
	flag.Parse()
	if *n < 0 || *n > 360 {
		return fmt.Errorf("advance must be 0–360")
	}
	var store application.Store
	if *path != "" {
		unlock, e := filestore.Lock(*path)
		if e != nil {
			return e
		}
		defer unlock()
		store = filestore.Store{Path: *path}
	}
	g, e := application.Open(store, slog.New(slog.NewJSONHandler(os.Stderr, nil)))
	if e != nil {
		return e
	}
	if *name != "" {
		if _, e = g.Apply(gameapi.Command{Kind: gameapi.RenameCompany, Revision: g.Snapshot().Revision, Name: *name}); e != nil {
			return e
		}
	}
	if *commands != "" {
		b, e := os.ReadFile(*commands)
		if e != nil {
			return e
		}
		var actions []gameapi.Command
		if e = json.Unmarshal(b, &actions); e != nil {
			return e
		}
		for i, c := range actions {
			c.Revision = g.Snapshot().Revision
			if _, e = g.Apply(c); e != nil {
				return fmt.Errorf("command %d: %w", i, e)
			}
		}
	}
	for i := 0; i < *n; i++ {
		if _, e = g.Apply(gameapi.Command{Kind: gameapi.AdvanceMonth, Revision: g.Snapshot().Revision}); e != nil {
			return e
		}
	}
	return json.NewEncoder(os.Stdout).Encode(g.Snapshot())
}
