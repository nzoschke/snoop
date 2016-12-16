package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/fsouza/go-dockerclient"

	"golang.org/x/net/websocket"
)

// Echo the data received on the WebSocket.
func EchoServer(ws *websocket.Conn) {
	io.Copy(ws, ws)
}

func DockerRunServer(ws *websocket.Conn) {
	dc, err := docker.NewClient("unix:///var/run/docker.sock")
	if err != nil {
		panic(err)
	}

	container, err := dc.CreateContainer(docker.CreateContainerOptions{
		Config: &docker.Config{
			Image:        "alpine",
			Cmd:          []string{"sh"},
			AttachStdin:  true,
			AttachStdout: true,
			AttachStderr: true,
			Tty:          true,
		},
	})
	fmt.Printf("CREATE CONTAINER: %+v\nERROR: %+v\n", container, err)
	if err != nil {
		panic(err)
	}

	err = dc.StartContainer(container.ID, &docker.HostConfig{})
	fmt.Printf("START CONTAINER ERROR: %+v\n", err)
	if err != nil {
		panic(err)
	}

	eres, err := dc.CreateExec(docker.CreateExecOptions{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true,
		Cmd:          []string{"sh"},
		Container:    container.ID,
	})
	if err != nil {
		panic(err)
	}

	success := make(chan struct{})

	go func() {
		<-success
		dc.ResizeExecTTY(eres.ID, 200, 60)
		success <- struct{}{}
	}()

	err = dc.StartExec(eres.ID, docker.StartExecOptions{
		Detach:       false,
		Tty:          true,
		InputStream:  ioutil.NopCloser(ws),
		OutputStream: ws,
		ErrorStream:  ws,
		RawTerminal:  true,
		Success:      success,
	})
	if err != nil {
		panic(err)
	}
}

// This example demonstrates a trivial echo server.
func main() {
	err := serve()
	if err != nil {
		fmt.Printf("ERROR: %+v\n", err)
	}
}

func serve() error {
	http.Handle("/", websocket.Handler(DockerRunServer))
	err := http.ListenAndServe(":8000", nil)
	if err != nil {
		return err
	}

	return nil
}
