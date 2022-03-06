package internal

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"golang.org/x/net/context"
)

func RunHttpServer(mcChan chan<- []byte, quit <-chan bool) <-chan bool {
	isDone := make(chan bool)
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		p, err := ioutil.ReadAll(r.Body)
		if err != nil {
			fmt.Fprintf(rw, "err")
			log.Println("http", err)
			return
		}
		if len(p) > 128 {
			fmt.Fprintf(rw, "max limit")
			return
		}
		if len(p) == 0 {
			return
		}
		mcChan <- p
	})
	server := &http.Server{Addr: ":80", Handler: mux}
	go (func() {
		if err := server.ListenAndServe(); err != nil {
			log.Println(err)
		}
		isDone <- true
	})()
	go (func() {
		<-quit
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		server.Shutdown(ctx)
		isDone <- true
	})()
	return isDone
}
