package main

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

func indexController(w http.ResponseWriter, r *http.Request) {

	type indexContextData struct {
		Events []Event
		Today  time.Time
	}

	theEvents, err := getAllEvents()
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	contextData := indexContextData{
		Events: theEvents,
		Today:  time.Now(),
	}

	tmpl["index"].Execute(w, contextData)
}

func createController(w http.ResponseWriter, r *http.Request) {

	tmpl["create"].Execute(w, nil)
}

func accessEvent(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/events/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid event ID", http.StatusBadRequest)
		return
	}

	contextEvent, exists := getEventByID(id)
	if !exists {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	tmpl["access"].Execute(w, contextEvent)
}
