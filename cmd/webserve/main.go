// webserve serves only the built web directory, with WASM MIME support.
package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
)

func main() {
	address := flag.String("listen", "127.0.0.1:8765", "listen address")
	dir := flag.String("dir", "web", "built web directory")
	flag.Parse()
	if _, e := os.Stat(*dir + "/main.wasm"); e != nil {
		fmt.Fprintln(os.Stderr, "Run sh tools/build-web.sh first")
		os.Exit(1)
	}
	fmt.Println("Open http://" + *address)
	if e := http.ListenAndServe(*address, http.FileServer(http.Dir(*dir))); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
