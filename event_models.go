package main

import (
	"database/sql"
	"errors"
	"time"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

var db *sql.DB // Declare the global `db` variable

// Event - encapsulates information about an event
type Event struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Location    string    `json:"location"`
	Image       string    `json:"image"`
	Date        time.Time `json:"date"`
	Attending   []string  `json:"attending"`
	RSVPMessage string    `json:"-"`
}

// getEventByID - returns the event in `allEvents` that has
// the specified id and a boolean indicating whether or not
// it was found. If it is not found, returns an empty event
// and false.
func getEventByID(id int) (Event, bool) {
	var event Event
	row := db.QueryRow("SELECT ID, Title, Location, Image, Date, RSVPMessage FROM Event WHERE ID = ?", id)
	err := row.Scan(&event.ID, &event.Title, &event.Location, &event.Image, &event.Date, &event.RSVPMessage)
	if err != nil {
		if err == sql.ErrNoRows {
			return Event{}, false
		}
		panic(err)
	}

	// Fetch attendees for this event
	attendeeRows, err := db.Query("SELECT Name FROM Attendee INNER JOIN Event_Attendee ON Attendee.ID = Event_Attendee.AttendeeID WHERE Event_Attendee.EventID = ?", id)
	if err != nil {
		panic(err)
	}
	defer attendeeRows.Close()
	for attendeeRows.Next() {
		var attendee string
		attendeeRows.Scan(&attendee)
		event.Attending = append(event.Attending, attendee)
	}

	return event, true
}

// getAllEvents - returns slice of all events and an error status. Here it is
// just returns `nil` always for the error. In mgt660, we're using similar
// code that might actually return an error, but here it's always `nil`.
func getAllEvents() ([]Event, error) {
	rows, err := db.Query("SELECT ID, Title, Location, Image, Date, RSVPMessage FROM Event")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var event Event
		if err := rows.Scan(&event.ID, &event.Title, &event.Location, &event.Image, &event.Date, &event.RSVPMessage); err != nil {
			return nil, err
		}

		// Fetch attendees for each event
		attendeeRows, err := db.Query("SELECT Name FROM Attendee INNER JOIN Event_Attendee ON Attendee.ID = Event_Attendee.AttendeeID WHERE Event_Attendee.EventID = ?", event.ID)
		if err != nil {
			return nil, err
		}
		defer attendeeRows.Close()
		for attendeeRows.Next() {
			var attendee string
			attendeeRows.Scan(&attendee)
			event.Attending = append(event.Attending, attendee)
		}

		events = append(events, event)
	}
	return events, nil
}

// getMaxEventID returns the maximum of all
// the ids of the events in allEvents
func getMaxEventID() int {
	var maxID int
	err := db.QueryRow("SELECT MAX(ID) FROM Event").Scan(&maxID)
	if err != nil {
		return -1 // Return -1 if there's an error or no rows in the table
	}
	return maxID
}

// Adds an attendee to an event
func addAttendee(eventID int, email string) error {
	// Check if the event exists
	if _, exists := getEventByID(eventID); !exists {
		return errors.New("no such event")
	}

	// Insert or find the attendee
	var attendeeID int
	err := db.QueryRow("SELECT ID FROM Attendee WHERE Name = ?", email).Scan(&attendeeID)
	if err == sql.ErrNoRows {
		res, err := db.Exec("INSERT INTO Attendee (Name) VALUES (?)", email)
		if err != nil {
			return err
		}
		id, err := res.LastInsertId()
		if err != nil {
			return err
		}
		attendeeID = int(id)
	} else if err != nil {
		return err
	}

	// Link the attendee to the event
	_, err = db.Exec("INSERT OR IGNORE INTO Event_Attendee (EventID, AttendeeID) VALUES (?, ?)", eventID, attendeeID)
	return err
}

// Add an event to the list of events.
func addEvent(event Event) {
	// Insert the event into the database
	if event.ID == 0 {
		event.ID = getMaxEventID() + 1
	}
	res, err := db.Exec("INSERT INTO Event (ID, Title, Location, Image, Date, RSVPMessage) VALUES (?, ?, ?, ?, ?, ?)", event.ID, event.Title, event.Location, event.Image, event.Date, event.RSVPMessage)
	if err != nil {
		panic(err)
	}

	// Retrieve the new event ID
	id, err := res.LastInsertId()
	if err != nil {
		panic(err)
	}
	event.ID = int(id)

	// Insert attendees if any are provided
	for _, attendee := range event.Attending {
		addAttendee(event.ID, attendee)
	}
}

func initDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "./events.db")
	if err != nil {
		return nil, err
	}
	// Create tables if they don't exist
	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS Event (
            ID INTEGER PRIMARY KEY,
            Title TEXT NOT NULL,
            Location TEXT,
            Image TEXT,
            Date DATETIME,
            RSVPMessage TEXT
        );
        
        CREATE TABLE IF NOT EXISTS Attendee (
            ID INTEGER PRIMARY KEY AUTOINCREMENT,
            Name TEXT NOT NULL
        );
        
        CREATE TABLE IF NOT EXISTS Event_Attendee (
            EventID INTEGER,
            AttendeeID INTEGER,
            PRIMARY KEY (EventID, AttendeeID),
            FOREIGN KEY (EventID) REFERENCES Event(ID) ON DELETE CASCADE,
            FOREIGN KEY (AttendeeID) REFERENCES Attendee(ID) ON DELETE CASCADE
        );
    `)
	if err != nil {
		return nil, err
	}
	return db, nil
}

// init is run once when this file is first loaded. See
// https://golang.org/doc/effective_go.html#init
// https://medium.com/golangspec/init-functions-in-go-eac191b3860a
func init() {
	var err error
	db, err = initDB()
	if err != nil {
		panic(err)
	}

	newYorkTimeZone, err := time.LoadLocation("America/New_York")
	if err != nil {
		panic("Could not load timezone database on your system!")
	}

	// Check if the Event table is empty before inserting default events
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM event").Scan(&count)
	if err != nil {
		panic(err)
	}

	if count == 0 {
		defaultEvents := []Event{
			{
				ID:        1,
				Title:     "SOM House Party",
				Date:      time.Date(2025, 10, 17, 16, 30, 0, 0, newYorkTimeZone),
				Image:     "http://i.imgur.com/pXjrQ.gif",
				Location:  "Kyle's house",
				Attending: []string{"kyle.jensen@yale.edu", "kim.kardashian@yale.edu"},
			},
			{
				ID:        2,
				Title:     "BBQ party for hackers and nerds",
				Date:      time.Date(2025, 10, 19, 19, 0, 0, 0, newYorkTimeZone),
				Image:     "http://i.imgur.com/7pe2k.gif",
				Location:  "Judy Chevalier's house",
				Attending: []string{"kyle.jensen@yale.edu", "kim.kardashian@yale.edu"},
			},
			{
				ID:        3,
				Title:     "BBQ for managers",
				Date:      time.Date(2025, 12, 2, 18, 0, 0, 0, newYorkTimeZone),
				Image:     "http://i.imgur.com/CJLrRqh.gif",
				Location:  "Barry Nalebuff's house",
				Attending: []string{"kim.kardashian@yale.edu"},
			},
			// Here I didn't include an even #4 just to show that
			// events in a real system might be deleted and so you
			// would need to handle such cases. Eg. if somebody
			// tries to get event #4, you would typically return
			// a 404 error which means "not found".
			{
				ID:        5,
				Title:     "Cooking lessons for the busy business student",
				Date:      time.Date(2025, 12, 21, 19, 0, 0, 0, newYorkTimeZone),
				Image:     "http://i.imgur.com/02KT9.gif",
				Location:  "Yale Farm",
				Attending: []string{"homer.simpson@yale.edu"},
			},
		}
		for _, event := range defaultEvents {
			addEvent(event)
		}
	}
}
