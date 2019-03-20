package main

import (
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

const gitRevision = "<filled in by CI service>"

func main() {
	r := chi.NewRouter()

	r.Use(
		middleware.RedirectSlashes,
		requestLogging,
	)

	r.Mount("/", NewAPI(conf.db))

	r.HandleFunc("/help", func(resp http.ResponseWriter, req *http.Request) {
		url := func(s string) string { return "http://" + req.Host + s }
		writeln := func(s string) { resp.Write([]byte(s + "\n")) }

		writeln("endpoints:")
		writeln("- /games ?user ?mode ?map")
		writeln("- /game/{id}")
		writeln("- /stats ?game ?user ?mode ?map")
		writeln("")
		writeln("examples:")
		writeln(url("/games"))
		writeln(url("/games?user=pix"))
		writeln(url("/games?mode=ectf&map=forge"))
		writeln(url("/game/1"))
		writeln(url("/stats"))
		writeln(url("/stats?user=pix"))
		writeln(url("/stats?mode=insta"))
		writeln(url("/stats?mode=ectf&map=forge"))
	})

	log.Println("server listening on", conf.webInterfaceAddress)
	err := http.ListenAndServe(conf.webInterfaceAddress, r)
	if err != nil {
		log.Println(err)
	}
}

func requestLogging(h http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		remoteAddr := req.Header.Get("X-Real-IP")
		if remoteAddr == "" {
			remoteAddr = req.RemoteAddr
		}
		log.Println(strings.Split(remoteAddr, ":")[0], "requested", req.URL.String())

		h.ServeHTTP(resp, req)
	})
}
