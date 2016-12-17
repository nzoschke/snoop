package main // "borrowed" from https://groups.google.com/forum/#!topic/golang-nuts/KBx9pDlvFOc
import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"

	"golang.org/x/net/websocket"
)

func websocketProxy(target string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("PROXY SERVER\n")

		backend, err := net.Dial("tcp", "localhost:8000")
		if err != nil {
			http.Error(w, "Error contacting backend server.", 500)
			log.Printf("Error dialing websocket backend %s: %v", target, err)
			return
		}
		defer backend.Close()

		snooper, err := net.Dial("tcp", "localhost:9001")
		if err != nil {
			http.Error(w, "Error contacting backend server.", 500)
			log.Printf("Error dialing websocket backend %s: %v", target, err)
			return
		}
		defer snooper.Close()

		hj, ok := w.(http.Hijacker)
		if !ok {
			log.Printf("Not a hijacker: %s", target)
			http.Error(w, "Not a hijacker?", 500)
			return
		}

		conn, _, err := hj.Hijack()
		if err != nil {
			log.Printf("Hijack error: %v", err)
			return
		}
		defer conn.Close()

		backends := io.MultiWriter(backend, snooper)
		clients := io.MultiWriter(conn, snooper)

		// write raw HTTP request to target.
		// NOTE: this includes proxied connection: upgrade headers with
		//       our newly injected auth header
		err = r.Write(backends)
		if err != nil {
			log.Printf("Error copying request to target: %v", err)
			return
		}

		// closure to block on both channels until they
		// are done copying, ignoring any errors
		errc := make(chan error, 2)
		cp := func(dst io.Writer, src io.Reader) {
			_, err := io.Copy(dst, src)
			fmt.Printf("COPY ERROR: %+v\n", err)
			errc <- err
		}

		// hookup both sides of connection
		go cp(backends, conn)
		go cp(clients, backend)
		<-errc
	})
}

func SnoopServer(ws *websocket.Conn) {
	fmt.Printf("SNOOP SERVER\n")
	buf := make([]byte, 32*1024)
	for {
		nr, err := ws.Read(buf)
		fmt.Printf("BUF: %+v\nN: %+v\nERR: %+v\n", string(buf), nr, err)
	}
}

func main() {
	snoop := &http.Server{
		Addr:    ":9001",
		Handler: websocket.Handler(SnoopServer),
	}
	go snoop.ListenAndServe()

	proxy := &http.Server{
		Addr:    ":9000",
		Handler: websocketProxy("localhost:8000"),
	}
	proxy.ListenAndServe()
}
