package main

import (
	"log"
	"net/http"
	"os"

	"panel/cloudflare"
	"panel/database"
	"panel/handlers"
	"panel/xray"
)

func main() {

	if err := database.Init(); err != nil {
		log.Fatal("Database initialization failed:", err)
	}

	xrayManager := xray.NewManager()

	if err := xrayManager.Start(); err != nil {
		log.Fatal(err)
	}

	trafficCollector := xray.NewTrafficCollector(xrayManager)
	trafficCollector.Start()

	tunnelManager := cloudflare.NewTunnelManager()

	go tunnelManager.StartVLESS()
	go tunnelManager.StartTrojan()

	clientHandler := handlers.NewClientHandler(
		tunnelManager,
		xrayManager,
		trafficCollector,
	)

	http.HandleFunc("/", handlers.RequireAuth(handlers.DashboardHandler))
	http.HandleFunc("/login", handlers.LoginHandler)
	http.HandleFunc("/logout", handlers.RequireAuth(handlers.LogoutHandler))

	// Client management
	http.HandleFunc("/clients/create", handlers.RequireAuth(clientHandler.Create))
	http.HandleFunc("/clients/delete", handlers.RequireAuth(clientHandler.Delete))
	http.HandleFunc("/clients/config", handlers.RequireAuth(clientHandler.Config))
	http.HandleFunc("/clients/quota", handlers.RequireAuth(clientHandler.EditQuota))
	http.HandleFunc("/clients/reset", handlers.RequireAuth(clientHandler.ResetTraffic))

	// Client controls
	http.HandleFunc("/clients/name", handlers.RequireAuth(clientHandler.EditName))
	http.HandleFunc("/clients/uuid", handlers.RequireAuth(clientHandler.ChangeUUID))
	http.HandleFunc("/clients/password", handlers.RequireAuth(clientHandler.ChangePassword))
	http.HandleFunc("/clients/status", handlers.RequireAuth(clientHandler.SetEnabled))

	// Live client status
	http.HandleFunc(
		"/api/clients/status",
		handlers.RequireAuth(handlers.ClientsStatusHandler),
	)

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
