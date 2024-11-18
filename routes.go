package main

import (
	"github.com/go-chi/chi/v5"
)

func createRoutes() chi.Router {
	// We're using chi as the router. You'll want to read
	// the documentation https://github.com/go-chi/chi
	// so that you can capture parameters like /events/5
	// or /api/events/4 -- where you want to get the
	// event id (5 and 4, respectively).

	r := chi.NewRouter()
	r.Get("/", indexController)
	addStaticFileServer(r, "/static/", "staticfiles")

	r.Get("/events/new", createEventController)
	r.Post("/events/new", createEventController)

	r.Get("/events/{id}", accessEventController)
	r.Post("/events/{id}", accessEventController)
	//r.Post("/events/{id}/rsvp", rsvpController)

	r.Get("/events/{id}/donate", donateController)

	r.Get("/about", aboutController)

	r.Get("/api/events", apiController)
	r.Get("/api/events/{id}", apiController)

	return r
}
