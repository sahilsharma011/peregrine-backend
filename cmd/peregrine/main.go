package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/Pigmice2733/peregrine-backend/internal/config"
	"github.com/Pigmice2733/peregrine-backend/internal/server"
	"github.com/Pigmice2733/peregrine-backend/internal/store"
	"github.com/Pigmice2733/peregrine-backend/internal/tba"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	var basePath = flag.String("basePath", ".", "Path to the etc directory where the config file is.")

	flag.Parse()

	if err := run(*basePath); err != nil {
		fmt.Printf("got error: %v\n", err)
		os.Exit(1)
	}
}

func run(basePath string) error {
	c, err := config.Open(basePath)
	if err != nil {
		return errors.Wrap(err, "opening config")
	}

	tba := tba.Service{
		URL:    c.TBA.URL,
		APIKey: c.TBA.APIKey,
	}

	sto, err := store.New(c.Database)
	if err != nil {
		return errors.Wrap(err, "opening postgres server")
	}

	if c.SeedUser != nil {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(c.SeedUser.Password), bcrypt.DefaultCost)
		if err != nil {
			return errors.Wrap(err, "creating seed user hashed password")
		}

		u := store.User{
			Username:       c.SeedUser.Username,
			HashedPassword: string(hashedPassword),
			Realm:          c.SeedUser.Realm,
			FirstName:      c.SeedUser.FirstName,
			LastName:       c.SeedUser.LastName,
			Roles:          c.SeedUser.Roles,
		}

		err = sto.CreateUser(u)
		_, ok := err.(*store.ErrExists)
		if err != nil && ok {
			return errors.Wrap(err, "creating seed user")
		}
	}

	if c.Server.Year == 0 {
		c.Server.Year = time.Now().Year()
	}

	jwtSecret := make([]byte, 64)
	if _, err := rand.Read(jwtSecret); err != nil {
		return errors.Wrap(err, "generating jwt secret")
	}

	logger := logrus.New()
	if c.Server.LogJSON {
		logger.Formatter = &logrus.JSONFormatter{}
	}

	s := &server.Server{
		TBA:       tba,
		Store:     sto,
		Logger:    logger,
		Server:    c.Server,
		JWTSecret: jwtSecret,
	}

	return errors.Wrap(s.Run(), "running server")
}
