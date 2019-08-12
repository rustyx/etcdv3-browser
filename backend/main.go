package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/cors"
)

var (
	httpPort       = envInt("HTTP_PORT", 8081, "listen port")
	allowedOrigins = env("CORS", "http://localhost:8080,http://localhost:8081", "CORS allowed origins")
	etcdEndpoints  = env("ETCD", "etcd:2379", "comma-separated list of etcd endpoints")
	editable       = envInt("EDITABLE", 0, "enable update functionality")
)

func main() {
	log.Printf("etcdv3-browser starting on port %d, etcd endpoint: %s\n", httpPort, etcdEndpoints)

	etcdClient, err := clientv3.New(clientv3.Config{
		Endpoints:            strings.Split(etcdEndpoints, ","),
		DialTimeout:          7 * time.Second,
		DialKeepAliveTime:    30 * time.Second,
		DialKeepAliveTimeout: 10 * time.Second,
	})
	if err != nil {
		log.Fatal(errors.Wrap(err, "etcd client"))
	}
	server := newServer(etcdClient, editable == 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/debug/health", healthCheck)
	mux.Handle("/metrics", promhttp.Handler())

	mux.HandleFunc("/test", handleTestPage)
	baseURI := "/api/kv/"
	mux.HandleFunc(baseURI, func(w http.ResponseWriter, r *http.Request) {
		server.handleRequest(w, r, baseURI)
	})
	mux.HandleFunc("/api/kvws", server.handleWebsocket)

	mux.Handle("/", http.FileServer(http.Dir("dist"))) // serves the frontend in a production image

	cors := cors.New(cors.Options{
		AllowedOrigins: strings.Split(allowedOrigins, ","),
		AllowedMethods: []string{"GET", "POST", "DELETE", "PUT", "OPTIONS"},
		AllowedHeaders: []string{"*"},
		// Debug:          true,
	})

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", httpPort), cors.Handler(mux)))
}

var templates = template.Must(template.ParseGlob("templates/*.gohtml"))

func handleTestPage(w http.ResponseWriter, r *http.Request) {
	model := struct {
		Method     string
		Proto      string
		RemoteAddr string
		Headers    []string
		Cookies    []*http.Cookie
	}{
		r.Method,
		r.Proto,
		r.RemoteAddr,
		[]string{},
		r.Cookies(),
	}
	var keys []string
	for k := range r.Header {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		for _, v := range r.Header[k] {
			model.Headers = append(model.Headers, fmt.Sprintf("%v: %v", k, v))
		}
	}

	if err := templates.ExecuteTemplate(w, "test.gohtml", &model); err != nil {
		log.Print("ExecuteTemplate: ", err)
	}
}
