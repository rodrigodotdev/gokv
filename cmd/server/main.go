package main

import (
	"context"
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rodrigodotdev/gokv/internal/container"
	"github.com/rodrigodotdev/gokv/internal/infraestructure/persistence"
)

var (
	port      = flag.String("port", "6379", "Port to listen on")
	enableAOF = flag.Bool("aof", false, "Enable append-only file persistence")
)

func main() {
	flag.Parse()

	cfg := persistence.AOFProviderConfig{
		Enabled:  *enableAOF,
		FilePath: "data.aof",
	}

	ctn, cleanup, err := container.InitializeContainer(cfg)
	if err != nil {
		log.Fatalf("Failed to create container: %v", err)
	}

	defer func() {
		if cleanup != nil {
			cleanup()
		}
		ctn.Close()
	}()

	if *enableAOF && ctn.Persistence != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := ctn.Persistence.Replay(ctx, ctn.Store); err != nil {
			log.Printf("Warning: failed to replay AOF: %v", err)
		} else {
			log.Println("AOF replay completed")
		}
	}

	ctn.Store.StartCleanup(1000)
	defer ctn.Store.StopCleanup()

	listener, err := net.Listen("tcp", ":"+*port)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", *port, err)
	}

	defer listener.Close()

	log.Printf("Server listening on port %s (AOF: %v)", *port, *enableAOF)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down...")
		listener.Close()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			break
		}

		go ctn.TCPHandler.HandleConnection(conn)
	}
}
