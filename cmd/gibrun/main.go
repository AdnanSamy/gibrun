package main

import (
	"flag"
	"fmt"
	"gibrun/internal/web"
	"log"
	"net"
	"net/http"
)

func main() {
	path := flag.String("data", "", "data file (default: OS configuration directory/gibrun/data.json)")
	port := flag.String("listen", "127.0.0.1:0", "listen address (explicit IP:port)")
	flag.Parse()
	overridden := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "data" {
			overridden = true
		}
	})
	resolved, err := dataPath(*path, overridden)
	if err != nil {
		log.Fatal(err)
	}
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
	s, err := web.New(resolved, l.Addr().String())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(s.URL())
	log.Fatal(http.Serve(l, s))
}
