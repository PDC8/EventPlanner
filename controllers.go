package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/mail"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func isValidURL(u string) bool {
	parsedURL, err := url.Parse(u)
	if err != nil {
		return false
	}
	// Check if the URL has a scheme and host
	return parsedURL.Scheme != "" && parsedURL.Host != ""
}

func isValidImageURL(filename string) bool {
	extensions := []string{".png", ".jpg", ".jpeg", ".gif", ".gifv"}
	for _, ext := range extensions {
		if strings.HasSuffix(filename, ext) {
			return isValidURL(filename)
		}
	}
	return false
}

func isFutureDate(dateStr string) (bool, time.Time) {
	// Parse the input string to a time.Time object (assuming the format is "2006-01-02" for date)
	date, err := time.Parse("2006-01-02T15:04", dateStr)
	if err != nil {
		return false, date
	}

	// Get the current time
	now := time.Now()
	return date.After(now), date

	// fiveHoursLater := now.Add(5 * time.Hour)
	// fmt.Println("Given time:", date)
	// fmt.Println("Current time:", now)
	// fmt.Println("Five hours later:", fiveHoursLater)

	// // // Check if the parsed date is at least 5 hours in the future
	// return fiveHoursLater.Before(date), date
}

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

func createEventController(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		// Parse form data from the POST request
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form submission", http.StatusBadRequest)
			return
		}

		// Error message

		data := EventError{
			ErrorMessage: "",
		}

		// Extract form values
		title := r.FormValue("title")
		if len(title) < 6 || len(title) > 49 {
			data.ErrorMessage += "Bad Title!"
		}
		location := r.FormValue("location")
		if len(location) < 6 || len(location) > 49 {
			data.ErrorMessage += " Bad Location!"
		}
		image := r.FormValue("image")
		if !isValidImageURL(image) {
			data.ErrorMessage += " Bad URL!"
		}
		dateStr := r.FormValue("date")

		// Parse the date
		// date, err := time.Parse("2006-01-02", dateStr)
		// if err != nil {
		// 	http.Error(w, "Invalid date format. Please use YYYY-MM-DD.", http.StatusBadRequest)
		// 	return
		// }
		//date, err := time.Parse("2006-01-02T15:04", dateStr)
		err, date := isFutureDate(dateStr)
		if !err {
			// http.Error(w, "Invalid date-time format. Please use YYYY-MM-DDTHH:MM.", http.StatusBadRequest)
			// return
			data.ErrorMessage += " Bad Date!"
			//data.ErrorMessage += dateStr
		}

		// Create new event
		newEvent := Event{
			Title:    title,
			Location: location,
			Image:    image,
			Date:     date,
		}
		if data.ErrorMessage == "" {
			// Add the event to the list of all events
			addEvent(newEvent)

			// Redirect or render success page
			http.Redirect(w, r, "/events/"+strconv.Itoa(getMaxEventID()), http.StatusSeeOther)
		} else {
			tmpl["create"].Execute(w, data)
		}

	} else {
		// Render the form if the request is a GET request
		tmpl["create"].Execute(w, nil)
	}
}

func accessEventController(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		//temp := r.URL. Path
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form submission", http.StatusBadRequest)
			return
		}

		idStr := r.FormValue("eventID")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Invalid event ID", http.StatusBadRequest)
			return
		}

		email := r.FormValue("email")
		_, err = mail.ParseAddress(email)
		if err != nil {
			http.Error(w, "Invalid email format. Please enter a valid email address.", http.StatusBadRequest)
			return
		}

		contextEvent, exists := getEventByID(id)
		if !exists {
			http.Error(w, "Event not found", http.StatusNotFound)
			return
		}

		contextEvent.RSVPMessage = ""
		contextEvent.RSVPClass = ""
		if !strings.HasSuffix(email, "@yale.edu") {
			contextEvent.RSVPMessage = "Bad email. Yalies only" //`<div class="error">Bad email. Yalies only</div>`
			contextEvent.RSVPClass = "error"
			//tmpl["access"].Execute(w, contextEvent)
		}

		if contextEvent.RSVPMessage == "" {
			for _, event := range contextEvent.Attending {
				if event == email {
					//http.Error(w, "Email is already RSVP-ed", http.StatusBadRequest)
					contextEvent.RSVPMessage = "Email is already RSVP-ed"
					break
					//tmpl["access"].Execute(w, contextEvent)
				}
			}
		}

		//addAttendee(id, email)
		if contextEvent.RSVPMessage == "" {
			err = addAttendee(contextEvent.ID, email)
			if err != nil {
				http.Error(w, "Event not found", http.StatusNotFound)
				return
			}

			// Compute the SHA-1 hash
			hasher := sha256.New()
			hasher.Write([]byte(email))
			hash := hasher.Sum(nil)

			// Convert the hash to a hexadecimal string
			hashHex := hex.EncodeToString(hash)

			// Print the first 7 characters of the hash
			contextEvent.SHA256Hash = hashHex[:7]

			contextEvent.RSVPMessage = "Thank You for your RSVP!"

			contextEvent.Attending = append(contextEvent.Attending, email)
		}

		tmpl["access"].Execute(w, contextEvent)

		//http.Redirect(w, r, r.URL.Path, http.StatusFound)
		// data := map[string]interface{}{
		// 	"Event":       contextEvent,
		// 	"RSVPMessage": rsvpMessage,
		// }
		// tmpl["access"].Execute(w, data)
		// fmt. Fprint(w, '«script>location.href = "http://localhost:8080/events/{id}";</script>*)
	} else {
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
}

// func rsvpController(w http.ResponseWriter, r *http.Request) {
// 	idStr := strings.TrimPrefix(r.URL.Path, "/events/")
// 	if strings.HasSuffix(idStr, "/rsvp") {
// 		idStr = strings.TrimSuffix(idStr, "/rsvp")
// 	}
// 	id, err := strconv.Atoi(idStr)
// 	if err != nil {
// 		http.Error(w, "Invalid event ID", http.StatusBadRequest)
// 		return
// 	}

// 	// Get the email from the form data
// 	email := r.FormValue("email")
// 	if email == "" {
// 		http.Error(w, "Email is required", http.StatusBadRequest)
// 		return
// 	}

// 	// Add the attendee to the event
// 	err = addAttendee(id, email)
// 	if err != nil {
// 		http.Error(w, "Event not found", http.StatusNotFound)
// 		return
// 	}

// 	// Retrieve the updated event data to show the latest attendee list
// 	contextEvent, exists := getEventByID(id)
// 	if !exists {
// 		http.Error(w, "Event not found", http.StatusNotFound)
// 		return
// 	}

// 	// Render the event page with updated data
// 	tmpl["access"].Execute(w, contextEvent)
// }

func aboutController(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		tmpl["about"].Execute(w, nil)
	}
}

func donateController(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		tmpl["donate"].Execute(w, nil)
	}
}

func apiController(w http.ResponseWriter, r *http.Request) {
	// Capture the event ID from the URL path
	idStr := strings.TrimPrefix(r.URL.Path, "/api/events")
	if idStr != "" {
		// Convert the event ID to an integer
		eventID, err := strconv.Atoi(strings.TrimPrefix(idStr, "/"))
		if err != nil {
			http.Error(w, "Invalid event ID", http.StatusBadRequest)
			return
		}

		// Fetch the specific event
		event, found := getEventByID(eventID)
		if !found {
			http.Error(w, "Event not found", http.StatusNotFound)
			return
		}

		// Respond with JSON for the specific event
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(event)
		return
	}

	// If no event ID is provided, return all events
	events, err := getAllEvents()
	if err != nil {
		http.Error(w, "Error retrieving events: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Respond with JSON for all events
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"events": events,
	})
}
