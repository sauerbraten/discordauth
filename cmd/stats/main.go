package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"

	"github.com/sauerbraten/maitred/pkg/auth"
)

const gitRevision = "<filled in by CI service>"

func main() {
	r := chi.NewRouter()

	r.Use(
		middleware.RedirectSlashes,
		requestLogging,
	)

	r.Mount("/api", NewAPI(conf.db))

	r.HandleFunc("/gen/{name}", func(resp http.ResponseWriter, req *http.Request) {
		name := chi.URLParam(req, "name")

		priv, pub, err := auth.GenerateKeyPair()
		if err != nil {
			resp.Write([]byte(err.Error()))
			return
		}

		fmt.Fprintf(resp, "add the following line to your auth.cfg:\n\n authkey \"%s\" \"%s\" \"stats.p1x.pw\"\n\n", name, hex.EncodeToString(priv))
		fmt.Fprintf(resp, "then, copy and paste these commands and run them in-game:\n\n")
		fmt.Fprintf(resp, " 1. /autoauth 1; connect p1x.pw\n")
		fmt.Fprintf(resp, " 2. /servcmd register %s %s\n", name, pub.String())
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
