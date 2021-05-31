package main

import (
	"flag"

	"github.com/DENKweit/distlock/cmd"
)

func main() {
	var port int
	flag.IntVar(&port, "port", 9876, "set port")
	flag.Parse()
	cmd.Start(port)
}
