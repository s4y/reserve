package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/s4y/reserve"
)

func main() {
	httpAddr := flag.String("http", "127.0.0.1:8080", "Listening address")
	flag.Parse()
	fmt.Printf("http://%s/\n", *httpAddr)

	ln, err := net.Listen("tcp", *httpAddr)
	if err != nil {
		log.Fatal(err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	server := reserve.CreateServer(cwd)
	log.Fatal(http.Serve(ln, server))
}
