package main

import (
	"flag"

	"github.com/docker/go-plugins-helpers/sdk"
)

var (
	memlogSocket string
)

func main() {
	flMemlogSocket := flag.String("memlog-socket", "/tmp/memlogd.sock", "socket to pass fd's to memlogd")
	flag.Parse()
	memlogSocket = *flMemlogSocket

	h := sdk.NewHandler(`{"Implements": ["LoggingDriver"]}`)
	handlers(&h, newDriver())

	if err := h.ServeUnix("linuxkit-logwrite", 0); err != nil {
		panic(err)
	}
}
