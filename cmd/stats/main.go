package main

import (
	"fmt"
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

	r.Mount("/api", NewAPI(conf.db))

	r.HandleFunc("/gen/{name}", func(resp http.ResponseWriter, req *http.Request) {
		name := chi.URLParam(req, "name")
		resp.Header().Set("Content-Type", "text/html")
		resp.Write([]byte(fmt.Sprintf(`<doctype html>
<html style="font-family: monospace; font-size: 11pt;">
<head><title>%s | %s</title></head>
<body>
<h1>Create your account in 2 easy steps</h1>

<p>Each step is just a command you can to copy and paste into Sauerbraten's game console:</p>

<ol>
<li>/connect p1x.pw</li>
<li>/authkey "%s" (genauthkey (rndstr 32)) "stats.p1x.pw"; saveauthkeys; autoauth 1; /servcmd register "%s" (getpubkey "stats.p1x.pw")</li>
</ol>
</body>
</html>`, name, req.Host, name, name)))
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
