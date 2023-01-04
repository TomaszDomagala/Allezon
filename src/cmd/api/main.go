package main

import (
	"log"

	"github.com/TomaszDomagala/Allezon/src/cmd/api/server"
)

func main() {
	producer, err := newProducer()
	if err != nil {
		log.Fatalf("Error while creating producer, %s", err)
	}
	srv := server.New(producer)

	if err := srv.Run(); err != nil {
		log.Fatalf("Error while running a server, %s", err)
	}
}
