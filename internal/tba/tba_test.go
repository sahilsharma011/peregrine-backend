package tba

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/Pigmice2733/peregrine-backend/internal/store"
	"github.com/gorilla/mux"
)

type tbaServer struct {
	*httptest.Server
	getEventsHandler  func(w http.ResponseWriter, r *http.Request)
	getMatchesHandler func(w http.ResponseWriter, r *http.Request)
}

const testingYear = 2018

func newInt(a int) *int {
	return &a
}

func newString(s string) *string {
	return &s
}

func newUnixPointer(time time.Time) *store.UnixTime {
	unix := store.NewUnixFromTime(time)
	return &unix
}

func newTBAServer() *tbaServer {
	ts := new(tbaServer)

	r := mux.NewRouter()
	r.HandleFunc("/events/"+strconv.Itoa(testingYear), func(w http.ResponseWriter, r *http.Request) { ts.getEventsHandler(w, r) })
	r.HandleFunc("/event/{eventKey}/matches/simple", func(w http.ResponseWriter, r *http.Request) { ts.getMatchesHandler(w, r) })

	ts.Server = httptest.NewServer(r)

	return ts
}

func TestGetEvents(t *testing.T) {
	server := newTBAServer()
	defer server.Close()

	APIKey := "notARealKey"

	s := Service{URL: server.URL, APIKey: APIKey}

	testCases := []struct {
		getEventsHandler func(w http.ResponseWriter, r *http.Request)
		events           []store.Event
		expectErr        bool
	}{
		{
			getEventsHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			events:    nil,
			expectErr: true,
		},
		{
			getEventsHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("X-TBA-Auth-Key") != APIKey {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`
				[
                    {
						"key": "key1",
						"short_name": "event1",
						"district": null,
						"week": null,
						"start_date": "2018-04-02",
						"end_date": "2018-04-04",
						"webcasts": [
							{
								"channel": "nefirst_blue",
								"type": "twitch"					
							}
						],
						"lat": 41.9911025,
						"lng": -70.993044,
						"location_name": "location1",
						"timezone": "America/Los_Angeles"
					}
				]
				`))

				if err != nil {
					t.Errorf("failed to write test data")
				}
			},
			events: []store.Event{
				{
					ID:        "key1",
					Name:      "event1",
					Week:      nil,
					District:  nil,
					StartDate: store.NewUnixFromTime(time.Date(2018, 4, 2, 7, 0, 0, 0, time.UTC)),
					EndDate:   store.NewUnixFromTime(time.Date(2018, 4, 4, 7, 0, 0, 0, time.UTC)),
					Location: store.Location{
						Lat:  41.9911025,
						Lon:  -70.993044,
						Name: "location1",
					},
					Webcasts: []store.Webcast{{
						Type: store.Twitch,
						URL:  "https://www.twitch.tv/nefirst_blue",
					}},
				},
			},
			expectErr: false,
		},
		{
			getEventsHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("X-TBA-Auth-Key") != "notARealKey" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`
				[
                    {
						"key": "key2",
						"short_name": "Event",
						"district": {
							"abbreviation": "ABC",
							"display_name": "Display name, not abbreviation",
							"key": "a_key",
							"year": 2018
						},
						"week": 5,
						"start_date": "2018-05-06",
						"end_date": "2018-05-07",
						"webcasts": [{
							"channel": "rXP6Vz9-Jjg",
							"type": "youtube"
						}, {
							"channel": "firstinspires12",
							"type": "twitch"
						}
					    ],
						"lat": 42.0,
						"lng": 0.0,
						"location_name": "answer"
					},
					{
						"key": "key3",
						"short_name": "PIGMICE_IS_BEST",
						"district": {
							"abbreviation": "PNW",
							"display_name": "Display name, not abbreviation",
							"key": "a_key",
							"year": 2018
						},
						"week": 2,
						"start_date": "2018-11-19",
						"end_date": "2018-11-23",
						"webcasts": [
							{
								"channel": "fakeIFRAME",
								"type": "iframe"									
							},
							{
								"channel": "gmsHpsSavuc",
								"type": "youtube"									
							}],
						"lat": 45.52,
						"lng": -122.681944,
						"location_name": "Portland",
						"timezone": "America/Los_Angeles"
				    }
				]
				`))

				if err != nil {
					t.Errorf("failed to write test data")
				}
			},
			events: []store.Event{{
				ID:        "key2",
				Name:      "Event",
				District:  newString("ABC"),
				Week:      newInt(5),
				StartDate: store.NewUnixFromTime(time.Date(2018, 5, 6, 0, 0, 0, 0, time.UTC)),
				EndDate:   store.NewUnixFromTime(time.Date(2018, 5, 7, 0, 0, 0, 0, time.UTC)),
				Location: store.Location{
					Lat:  42.0,
					Lon:  0.0,
					Name: "answer",
				},
				Webcasts: []store.Webcast{{
					Type: store.Youtube,
					URL:  "https://www.youtube.com/watch?v=rXP6Vz9-Jjg",
				}, {
					Type: store.Twitch,
					URL:  "https://www.twitch.tv/firstinspires12",
				}},
			}, {
				ID:        "key3",
				Name:      "PIGMICE_IS_BEST",
				District:  newString("PNW"),
				Week:      newInt(2),
				StartDate: store.NewUnixFromTime(time.Date(2018, 11, 19, 8, 0, 0, 0, time.UTC)),
				EndDate:   store.NewUnixFromTime(time.Date(2018, 11, 23, 8, 0, 0, 0, time.UTC)),
				Location: store.Location{
					Lat:  45.52,
					Lon:  -122.681944,
					Name: "Portland",
				},
				Webcasts: []store.Webcast{{
					Type: store.Youtube,
					URL:  "https://www.youtube.com/watch?v=gmsHpsSavuc",
				}},
			}},
			expectErr: false,
		},
	}

	for index, tt := range testCases {
		server.getEventsHandler = tt.getEventsHandler

		events, err := s.GetEvents(testingYear)
		if tt.expectErr != (err != nil) {
			t.Errorf("test #%v - got error: %v, expected error: %v", index+1, err, tt.expectErr)
		}

		if !reflect.DeepEqual(events, tt.events) {
			t.Errorf("test #%v - got events: %#v\n    expected: %#v", index+1, events, tt.events)
		}
	}
}

func TestGetMatches(t *testing.T) {
	server := newTBAServer()
	defer server.Close()

	APIKey := "alsoNotARealKey"

	s := Service{URL: server.URL, APIKey: APIKey}

	testCases := []struct {
		getMatchesHandler func(w http.ResponseWriter, r *http.Request)
		eventID           string
		matches           []store.Match
		expectErr         bool
	}{
		{
			getMatchesHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			eventID:   "none",
			matches:   nil,
			expectErr: true,
		},
		{
			getMatchesHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("X-TBA-Auth-Key") != APIKey {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`
				[
                    {
						"key": "key1",
						"alliances": {
							"red": {
								"score": 220,
								"team_keys": ["frc254", "frc1234", "frc00"]
							},
							"blue": {
								"score": 500,
								"team_keys": ["frc2733", "frc9876", "frc1"]
							}
						},
						"predicted_time": 1520272800,
						"actual_time": 1520274000
					},
					{
						"key": "key2",
						"alliances": {
							"red": {
								"score": 120,
								"team_keys": ["frc0", "frc1", "frc2"]
							},
							"blue": {
								"score": 600,
								"team_keys": ["frc2", "frc7", "frc3"]
							}
						},
						"predicted_time": 1525272780,
						"actual_time": 1525273980
					}
				]
				`))

				if err != nil {
					t.Errorf("failed to write test data")
				}
			},
			eventID: "2018alhu",
			matches: []store.Match{
				{
					ID:            "key1",
					EventID:       "2018alhu",
					PredictedTime: newUnixPointer(time.Date(2018, 3, 5, 18, 0, 0, 0, time.UTC)),
					ActualTime:    newUnixPointer(time.Date(2018, 3, 5, 18, 20, 0, 0, time.UTC)),
					RedScore:      newInt(220),
					BlueScore:     newInt(500),
					RedAlliance:   []string{"frc254", "frc1234", "frc00"},
					BlueAlliance:  []string{"frc2733", "frc9876", "frc1"},
				},
				{
					ID:            "key2",
					EventID:       "2018alhu",
					PredictedTime: newUnixPointer(time.Date(2018, 5, 2, 14, 53, 0, 0, time.UTC)),
					ActualTime:    newUnixPointer(time.Date(2018, 5, 2, 15, 13, 0, 0, time.UTC)),
					RedScore:      newInt(120),
					BlueScore:     newInt(600),
					RedAlliance:   []string{"frc0", "frc1", "frc2"},
					BlueAlliance:  []string{"frc2", "frc7", "frc3"},
				},
			},
			expectErr: false,
		},
		{
			getMatchesHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("X-TBA-Auth-Key") != APIKey {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`
				[
                    {
						"key": "key1",
						"alliances": {
							"red": {
								"score": -1,
								"team_keys": ["frc254", "frc1234", "frc00"]
							},
							"blue": {
								"score": -1,
								"team_keys": ["frc2733", "frc9876", "frc1"]
							}
						},
						"predicted_time": 1520272800,
						"actual_time": 1520274000
					},
					{
						"key": "key2",
						"alliances": {
							"red": {
								"score": null,
								"team_keys": ["frc0", "frc1", "frc2"]
							},
							"blue": {
								"score": null,
								"team_keys": ["frc2", "frc7", "frc3"]
							}
						},
						"predicted_time": 1525272780,
						"actual_time": null
					}
				]
				`))

				if err != nil {
					t.Errorf("failed to write test data")
				}
			},
			eventID: "2018alhu",
			matches: []store.Match{
				{
					ID:            "key1",
					EventID:       "2018alhu",
					PredictedTime: newUnixPointer(time.Date(2018, 3, 5, 18, 0, 0, 0, time.UTC)),
					ActualTime:    newUnixPointer(time.Date(2018, 3, 5, 18, 20, 0, 0, time.UTC)),
					RedScore:      nil,
					BlueScore:     nil,
					RedAlliance:   []string{"frc254", "frc1234", "frc00"},
					BlueAlliance:  []string{"frc2733", "frc9876", "frc1"},
				},
				{
					ID:            "key2",
					EventID:       "2018alhu",
					PredictedTime: newUnixPointer(time.Date(2018, 5, 2, 14, 53, 0, 0, time.UTC)),
					ActualTime:    nil,
					RedScore:      nil,
					BlueScore:     nil,
					RedAlliance:   []string{"frc0", "frc1", "frc2"},
					BlueAlliance:  []string{"frc2", "frc7", "frc3"},
				},
			},
			expectErr: false,
		},
	}

	for index, tt := range testCases {
		server.getMatchesHandler = tt.getMatchesHandler

		matches, err := s.GetMatches(tt.eventID)
		if tt.expectErr != (err != nil) {
			t.Errorf("test #%v - got error: %v, expected error: %v", index+1, err, tt.expectErr)
		}

		if !reflect.DeepEqual(matches, tt.matches) {
			t.Errorf("test #%v - got matches: %#v\n    expected: %#v", index+1, matches, tt.matches)
		}
	}
}
