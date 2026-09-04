package handlers

import (
	"html/template"
	"net/http"
)

func renderTemplate(w http.ResponseWriter, file string, data interface{}) error {
	tmpl, err := template.ParseFiles(file)
	if err != nil {
		return err
	}

	return tmpl.Execute(w, data)
}
