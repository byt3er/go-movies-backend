package main

import (
	"backend/internals/repository"
	"backend/internals/repository/dbrepo"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"
)

const port = 8080

type application struct {
	DSN    string
	Domain string
	// DB     *sql.DB //=> is a pool of database connections
	DB           repository.DatabaseRepo
	auth         Auth
	JWTSecret    string
	JWTIssuer    string
	JWTAudience  string
	CookieDomain string
	APIKey       string
}

func main() {
	// set application config
	// an application config is nothing more than a
	// type that stores bit of information that my application
	// is going to need
	// like how do I connect to database
	// where is my database repository
	// where is my JWT secret(the secret we're going
	// to use to sign to our jwt tokens)
	var app application

	// read from the command line
	// flag package is part of the standard library
	flag.StringVar(&app.DSN, "dsn", "host=localhost port=5432 user=postgres password=postgres dbname=movies sslmode=disable timezone=UTC connect_timeout=5", "Postgres connection string") // pgx connection string
	flag.StringVar(&app.JWTSecret, "jwt-secret", "verysecret", "signing secret")
	flag.StringVar(&app.JWTIssuer, "jwt-issuer", "example.com", "signing issuer")
	flag.StringVar(&app.JWTAudience, "jwt-audience", "example.com", "signing audience")
	flag.StringVar(&app.CookieDomain, "cookie-domain", "localhost", "cookie domain")
	flag.StringVar(&app.Domain, "domain", "example.com", " domain")
	flag.StringVar(&app.APIKey, "api-key", "3859630f1b7f23836cf6030336669b4a", "api key")
	flag.Parse() // parses everything that we read from the command line

	// connect to database
	conn, err := app.connectToDB()
	// conn is the pool of database connection
	// should close them when we're done with them
	// and if we don't, we'll have a resource leak, and
	// we'll be leaving connections to Postgres open,
	// even when we don't need them anymore.
	// we need to close the pool of connections,
	// the way we want to close them is just before the application exits
	// defer app.DB.Close()
	if err != nil {
		log.Fatal(err)
	}
	//app.DB = conn
	//defer app.DB.Close()
	app.DB = &dbrepo.PostgresDBRepo{DB: conn}
	//defer conn.Close()
	defer app.DB.Connection().Close()

	app.auth = Auth{
		Issuer:        app.JWTIssuer,
		Audience:      app.JWTAudience,
		Secret:        app.JWTSecret,
		TokenExpiry:   time.Minute * 15, // 15 mins
		RefreshExpiry: time.Hour * 24,   // for 24 hours
		CookiePath:    "/",              // the root level of our application
		CookieName:    "__Host-refresh_token",
		CookieDomain:  app.CookieDomain,
	}

	log.Println("Starting application on port ", port)

	//http.HandleFunc("/", Hello)

	// start a web server
	err = http.ListenAndServe(fmt.Sprintf(":%d", port), app.routes())
	if err != nil {
		// unable to start the server. Just die and log the error
		log.Fatal(err)
	}
}
