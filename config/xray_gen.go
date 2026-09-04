package config

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
)

func GenerateVlessConfig(sni, wsHost string) map[string]interface{} {
	id := uuid.New().String()
	return map[string]interface{}{
		"protocol": "vless",
		"settings": map[string]interface{}{
			"clients": []map[string]string{{"id": id}},
		},
		"streamSettings": map[string]interface{}{
			"network":  "ws",
			"security": "tls",
			"wsSettings": map[string]string{"path": "/vless", "host": wsHost},
			"tlsSettings": map[string]string{"serverName": sni},
		},
	}
}

func GenerateTrojanConfig(sni, wsHost string) map[string]interface{} {
	password := uuid.New().String()
	return map[string]interface{}{
		"protocol": "trojan",
		"settings": map[string]interface{}{
			"clients": []map[string]string{{"password": password}},
		},
		"streamSettings": map[string]interface{}{
			"network":  "ws",
			"security": "tls",
			"wsSettings": map[string]string{"path": "/trojan", "host": wsHost},
			"tlsSettings": map[string]string{"serverName": sni},
		},
	}
}

func ToJSON(cfg map[string]interface{}) (string, error) {
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal error: %v", err)
	}
	return string(b), nil
}
