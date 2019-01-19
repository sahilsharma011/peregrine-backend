package server

import (
	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
	"github.com/gorilla/mux"
)

func (s *Server) registerRoutes() *mux.Router {
	r := mux.NewRouter()

	r.Handle("/", s.healthHandler()).Methods("GET")
	r.Handle("/openapi.yaml", s.openAPIHandler()).Methods("GET")

	r.Handle("/authenticate", s.authenticateHandler()).Methods("POST")
	r.Handle("/refresh", s.refreshHandler()).Methods("POST")

	r.Handle("/users", s.createUserHandler()).Methods("POST")
	r.Handle("/users", ihttp.ACL(s.getUsersHandler(), true, true, true)).Methods("GET")
	r.Handle("/users/{id}", ihttp.ACL(s.getUserByIDHandler(), false, false, true)).Methods("GET")
	r.Handle("/users/{id}", ihttp.ACL(s.patchUserHandler(), false, false, true)).Methods("PATCH")
	r.Handle("/users/{id}", ihttp.ACL(s.deleteUserHandler(), false, false, true)).Methods("DELETE")

	r.Handle("/schemas", ihttp.ACL(s.getSchemasHandler(), false, false, false)).Methods("GET")
	r.Handle("/schemas", ihttp.ACL(s.createSchemaHandler(), true, true, true)).Methods("POST")
	// TODO(franklin): this endpoint doesn't really make sense RESTfully. Year should prob
	// be a query param on /schemas or something
	r.Handle("/schemas/year/{year}", ihttp.ACL(s.getSchemaByYearHandler(), false, false, false)).Methods("GET")
	r.Handle("/schemas/{id}", ihttp.ACL(s.getSchemaByIDHandler(), false, false, false)).Methods("GET")

	r.Handle("/events", s.eventsHandler()).Methods("GET")
	r.Handle("/events", ihttp.ACL(s.upsertEventHandler(), true, true, true)).Methods("PUT")
	r.Handle("/events/{eventKey}", s.eventHandler()).Methods("GET")

	r.Handle("/events/{eventKey}/stats", s.eventStats()).Methods("GET")

	r.Handle("/events/{eventKey}/matches", s.matchesHandler()).Methods("GET")
	r.Handle("/events/{eventKey}/matches", ihttp.ACL(s.createMatchHandler(), true, true, true)).Methods("POST")
	r.Handle("/events/{eventKey}/matches/{matchKey}", s.matchHandler()).Methods("GET")

	r.Handle("/events/{eventKey}/teams", s.teamsHandler()).Methods("GET")
	r.Handle("/events/{eventKey}/teams/{teamKey}", s.teamInfoHandler()).Methods("GET")

	r.Handle("/events/{eventKey}/matches/{matchKey}/reports/{teamKey}", ihttp.ACL(s.getReports(), false, false, false)).Methods("GET")
	r.Handle("/events/{eventKey}/matches/{matchKey}/reports/{teamKey}", ihttp.ACL(s.putReport(), false, true, true)).Methods("PUT")

	r.Handle("/realms", s.realmsHandler()).Methods("GET")
	// TODO(franklin): replace with PUT
	r.Handle("/realms", ihttp.ACL(s.createRealmHandler(), true, true, true)).Methods("POST")
	r.Handle("/realms/{id}", s.realmHandler()).Methods("GET")
	r.Handle("/realms/{id}", ihttp.ACL(s.deleteRealmHandler(), true, true, true)).Methods("DELETE")

	return r
}
