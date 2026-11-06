package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"solarclient/internal/web"
)

func main() {
	dir, err := os.UserConfigDir()
	if err != nil {
		log.Fatal(err)
	}
	path := flag.String("data", filepath.Join(dir, "solarclient", "data.json"), "data file")
	port := flag.String("listen", "127.0.0.1:0", "listen address (explicit IP:port)")
	flag.Parse()
	addr, err := net.ResolveTCPAddr("tcp", *port)
	if err != nil || addr.IP == nil || addr.IP.IsUnspecified() {
		log.Fatal("listen address must specify an IP address, not a wildcard")
	}
	if !addr.IP.IsLoopback() {
		log.Print("WARNING: HTTP exposes your session and credentials to the network; use only on a trusted network")
	}
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()
	s, err := web.New(*path, l.Addr().String())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(s.URL())
	log.Fatal(http.Serve(l, s))
}
