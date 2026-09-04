package main

import (
	"log"
	"net/http"
	"os"
	"panel/handlers"
)

func main() {
	http.HandleFunc("/", handlers.DashboardHandler)
	http.HandleFunc("/login", handlers.LoginHandler)
	http.HandleFunc("/logout", handlers.LogoutHandler)
	http.HandleFunc("/api/config/create", handlers.CreateConfigHandler)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Panel running on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
