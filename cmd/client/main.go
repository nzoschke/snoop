package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"os"
	"sync"

	"golang.org/x/net/websocket"
)

func main() {
	err := connect()
	if err != nil {
		fmt.Printf("ERROR: %+v\n", err)
	}
}

func connect() error {
	origin := fmt.Sprintf("https://localhost:8000")
	endpoint := fmt.Sprintf("ws://localhost:8000")

	config, err := websocket.NewConfig(endpoint, origin)
	if err != nil {
		return err
	}

	config.TlsConfig = &tls.Config{
		InsecureSkipVerify: true,
	}

	ws, err := websocket.DialConfig(config)
	if err != nil {
		return err
	}
	defer ws.Close()

	go io.Copy(ws, os.Stdin)

	var wg sync.WaitGroup
	wg.Add(1)
	go copyAsync(os.Stdout, ws, &wg)
	wg.Wait()

	return nil
}

func copyAsync(dst io.Writer, src io.Reader, wg *sync.WaitGroup) {
	defer wg.Done()
	io.Copy(dst, src)
}
