package main

import (
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

		fmt.Fprintf(resp, "+++ THIS IS ALPHA ZONE! ACCOUNTS ARE LOST AT EACH SERVER RESTART! +++")
		fmt.Fprintf(resp, "\n")
		fmt.Fprintf(resp, "\n")
		fmt.Fprintf(resp, "CREATE AN ACCOUNT IN 3 SIMPLE STEPS\n")
		fmt.Fprintf(resp, "each step is just a command you have to execute in sauer\n")
		fmt.Fprintf(resp, "\n")
		fmt.Fprintf(resp, "1. copy and paste the following line and execute it in in the chat prompt:\n")
		fmt.Fprintf(resp, "\n")
		fmt.Fprintf(resp, "/authkey \"%s\" \"%s\" \"stats.p1x.pw\"; saveauthkeys; autoauth 1\n", name, priv.String())
		fmt.Fprintf(resp, "\n")
		fmt.Fprintf(resp, "2. then:\n")
		fmt.Fprintf(resp, "\n")
		fmt.Fprintf(resp, "/connect p1x.pw\n")
		fmt.Fprintf(resp, "\n")
		fmt.Fprintf(resp, "3. when you are connected, complete the registration:\n")
		fmt.Fprintf(resp, "\n")
		fmt.Fprintf(resp, "/servcmd register %s %s\n", name, pub.String())
		fmt.Fprintf(resp, "\n")
		fmt.Fprintf(resp, "\n")
		fmt.Fprintf(resp, "+++ THIS IS ALPHA ZONE! ACCOUNTS ARE LOST AT EACH SERVER RESTART! +++")
	})

	r.Mount("/", http.FileServer(http.Dir("./public")))

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
