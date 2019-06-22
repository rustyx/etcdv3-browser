package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/cors"
	"go.etcd.io/etcd/clientv3"
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
		DialTimeout:          time.Duration(7) * time.Second,
		DialKeepAliveTime:    time.Duration(30) * time.Second,
		DialKeepAliveTimeout: time.Duration(10) * time.Second,
	})
	if err != nil {
		log.Fatal(errors.Wrap(err, "etcd client"))
	}
	server := newServer(etcdClient, editable == 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/debug/health", healthCheck)
	mux.Handle("/metrics", promhttp.Handler())

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
