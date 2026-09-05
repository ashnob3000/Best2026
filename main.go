package main

import (
	"log"
	"net/http"
	"os"
	"os/exec"

	"panel/cloudflare"
	"panel/database"
	"panel/handlers"
)

func main() {

	// Initialize database
	if err := database.Init(); err != nil {
		log.Fatal("Database initialization failed:", err)
	}
	// Start Xray
xrayCmd := exec.Command(
	"xray",
	"run",
	"-c",
	"config/xray.json",
)

xrayCmd.Stdout = os.Stdout
xrayCmd.Stderr = os.Stderr

if err := xrayCmd.Start(); err != nil {
	log.Fatal("Failed to start Xray:", err)
}

log.Println("Xray started")

	// Cloudflared tunnels
	tunnelManager := cloudflare.NewTunnelManager()

	go tunnelManager.StartVLESS()
	go tunnelManager.StartTrojan()

	// Client handlers
	clientHandler := handlers.NewClientHandler(tunnelManager)

	http.HandleFunc("/", handlers.DashboardHandler)
	http.HandleFunc("/login", handlers.LoginHandler)
	http.HandleFunc("/logout", handlers.LogoutHandler)

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
