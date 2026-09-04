package main

import (
	"log"
	"net/http"
	"os"

	"panel/database"
	"panel/handlers"
)

func main() {

	// Initialize database
	if err := database.Init(); err != nil {
		log.Fatal("Database initialization failed:", err)
	}

	// Client handlers
	clientHandler := handlers.NewClientHandler()

	http.HandleFunc("/", handlers.DashboardHandler)
	http.HandleFunc("/login", handlers.LoginHandler)
	http.HandleFunc("/logout", handlers.LogoutHandler)

	// Old API
	http.HandleFunc("/api/config/create", handlers.CreateConfigHandler)

	// Client management
	http.HandleFunc("/clients/create", clientHandler.Create)
	http.HandleFunc("/clients/delete", clientHandler.Delete)
	http.HandleFunc("/clients/config", clientHandler.Config)

	// Static files
	http.Handle(
		"/static/",
		http.StripPrefix(
			"/static/",
			http.FileServer(http.Dir("static")),
		),
	)

	port := os.Getenv("PORT")

	if port == "" {
		port = "8080"
	}

	log.Printf("Panel running on port %s", port)

	log.Fatal(
		http.ListenAndServe(":"+port, nil),
	)
}
