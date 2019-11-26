package main

import (
	"fmt"
	flag "github.com/spf13/pflag"
	"os"
)

func main() {
	flag.Usage = func() {
		_, _ = fmt.Fprint(os.Stderr, "Usage of infocenter:\n"+
			"    infocenter [options]\n"+
			"Options:\n")
		flag.PrintDefaults()
		_, _ = fmt.Fprintln(os.Stderr,
			"\nInfocenter server application that uses server-sent events")
	}
	port := flag.Uint16P("port", "p", 8080, "port to listen on")
	flag.ParseAll(func(_ *flag.Flag, _ string) error { return nil })
	fmt.Printf("Listen on port %d\n", *port)
}
