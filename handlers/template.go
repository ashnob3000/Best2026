package handlers

import (
	"fmt"
	"html/template"
	"net/http"
)

func renderTemplate(w http.ResponseWriter, file string, data interface{}) error {

	funcs := template.FuncMap{
		"formatBytes": formatBytes,
		"percentage":  percentage,
	}

	tmpl, err := template.New(file).
		Funcs(funcs).
		ParseFiles(file)

	if err != nil {
		return err
	}

	return tmpl.Execute(w, data)
}

func formatBytes(bytes int64) string {

	if bytes < 1000 {
		return fmt.Sprintf("%d B", bytes)
	}

	if bytes < 1000000 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1000)
	}

	if bytes < 1000000000 {
		return fmt.Sprintf("%.1f MB", float64(bytes)/1000000)
	}

	return fmt.Sprintf("%.2f GB", float64(bytes)/1000000000)
}

func percentage(used int64, limit int64) float64 {

	if limit <= 0 {
		return 0
	}

	percent := float64(used) * 100 / float64(limit)

	if percent > 100 {
		return 100
	}

	if percent < 0 {
		return 0
	}

	return percent
}
