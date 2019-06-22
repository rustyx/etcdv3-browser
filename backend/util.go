package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
)

func env(key string, defaultValue string, comment string) string {
	val, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	return val
}

func envInt(key string, defaultValue int, comment string) int {
	val, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		log.Fatal(key, ": ", err)
	}
	return i
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "OK")
}
