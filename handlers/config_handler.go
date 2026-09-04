package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"panel/config"
)

func CreateConfigHandler(w http.ResponseWriter, r *http.Request) {
	if !IsAuthenticated(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	r.ParseForm()
	protocol := r.FormValue("protocol")
	sni := os.Getenv("DOMAIN")
	if sni == "" { sni = "example.com" }

	var cfg map[string]interface{}
	if protocol == "trojan" {
		cfg = config.GenerateTrojanConfig(sni, sni)
	} else {
		cfg = config.GenerateVlessConfig(sni, sni)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cfg)
}
