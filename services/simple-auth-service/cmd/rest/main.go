package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/TSM-061/Raggy/shared/env"
	"github.com/TSM-061/Raggy/simple-auth-service/internal/config"
)

func main() {
	helper := env.NewHelper(os.LookupEnv)
	config := config.LoadConfig(helper)

	mux := http.NewServeMux()

	mux.HandleFunc("/api/auth/hello", handleHello)

	log.Printf("Simple Auth Service listening on :%d", config.Port)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", config.Port), mux))
}

func handleHello(w http.ResponseWriter, r *http.Request) {
	// Get the "name" from the URL query parameters
	name := r.URL.Query().Get("name")
	if name == "" {
		name = "World"
	}

	// Set the content type to plain text
	w.Header().Set("Content-Type", "text/plain")

	// Write the response
	fmt.Fprintf(w, "Hello, %s! The Auth Service is alive.", name)
}
