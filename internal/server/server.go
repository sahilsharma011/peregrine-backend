package server

import (
	"net/http"
	"time"

	"github.com/NYTimes/gziphandler"
	"github.com/Pigmice2733/peregrine-backend/internal/config"
	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
	"github.com/Pigmice2733/peregrine-backend/internal/store"
	"github.com/Pigmice2733/peregrine-backend/internal/tba"
	"github.com/sirupsen/logrus"
)

//go:generate go run ../cmd/pack/pack.go -package server -in openapi.yaml -out openapi.go -name openAPI
var openAPI []byte

// Server is the scouting API server
type Server struct {
	config.Server

	TBA    *tba.Service
	Store  *store.Service
	Logger *logrus.Logger
	start  time.Time
}

func (s *Server) uptime() time.Duration {
	return time.Since(s.start)
}

// Run starts the server, and returns if it runs into an error
func (s *Server) Run() error {
	router := s.registerRoutes()

	var handler http.Handler = router
	handler = ihttp.LimitBody(handler)
	handler = gziphandler.GzipHandler(handler)
	handler = ihttp.Log(handler, s.Logger)
	handler = ihttp.Auth(handler, s.JWTSecret)
	handler = ihttp.CORS(handler, s.Origin)

	httpServer := &http.Server{
		Addr:              s.Listen,
		Handler:           handler,
		ReadTimeout:       time.Second * 15,
		ReadHeaderTimeout: time.Second * 15,
		WriteTimeout:      time.Second * 15,
		IdleTimeout:       time.Second * 30,
		MaxHeaderBytes:    4096,
	}

	s.start = time.Now()
	s.Logger.WithField("httpAddress", s.Listen).Info("serving http")
	return httpServer.ListenAndServe()
}
