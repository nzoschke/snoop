package main // "borrowed" from https://groups.google.com/forum/#!topic/golang-nuts/KBx9pDlvFOc
import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
)

func websocketProxy(target string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("PROXY SERVER\n")

		dialer, err := net.Dial("tcp", "localhost:8000")
		if err != nil {
			http.Error(w, "Error contacting backend server.", 500)
			log.Printf("Error dialing websocket backend %s: %v", target, err)
			return
		}
		defer dialer.Close()

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

		// write raw HTTP request to target.
		// NOTE: this includes proxied connection: upgrade headers with
		//       our newly injected auth header
		err = r.Write(dialer)
		if err != nil {
			log.Printf("Error copying request to target: %v", err)
			return
		}

		// closure to block on both channels until they
		// are done copying, ignoring any errors
		errc := make(chan error, 2)
		cp := func(dst io.Writer, src io.Reader) {
			_, err := io.Copy(dst, src)
			errc <- err
		}

		//hookup both sides of connection
		go cp(dialer, conn)
		go cp(conn, dialer)
		<-errc
	})
}

// This example demonstrates a trivial echo server.
func main() {
	http.Handle("/", websocketProxy("localhost:8000"))
	err := http.ListenAndServe(":9000", nil)
	if err != nil {
		panic(err)
	}

	// proxy := websocketProxy("localhost:8000")
	// proxy.ServeHTTP(w, r)
}
