package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (app *application) routes() http.Handler {
	// create a router mux(multiplexer)
	mux := chi.NewRouter()

	// middleware (this will apply to every request that
	// comes into our application.)

	//****************************************************
	//********** Middlewares *****************************
	// Recoverer : all this does is when your
	// application panics for some reason , it will
	// log it along with backtrace and showing you where
	// the error took place.
	// it will send back the nessecary header, which
	// is HTTP 500 , there is some kind of internal
	// server error and then it bring things back up
	// so your application doesn't grind to halt
	mux.Use(middleware.Recoverer)
	mux.Use(app.enableCORS)

	// anytime you get Get() request to "/" path
	// go to the hander app.Home
	mux.Get("/", app.Home)

	mux.Post("/authenticate", app.authenticate) // b/c we're sending JSON file

	//get request by default will include the refresh token cookie if
	// it exists in the user browser
	mux.Get("/refresh", app.refreshToken)
	mux.Get("/logout", app.logout)

	mux.Get("/movies", app.AllMovie)
	mux.Get("/movies/{id}", app.GetMovie)

	mux.Get("/genres", app.AllGenres)
	mux.Get("/movies/genres/{id}", app.AllMoviesByGenre)

	// we've just have one path to one handler and we'll use that for all
	// of our GrapQL related requests
	mux.Post("/graph", app.moviesGraphQL)
	//******************************************
	//******** Routes **************************

	mux.Route("/admin", func(mux chi.Router) {
		// jwt validation middleware
		mux.Use(app.authRequired)

		// protected routes
		mux.Get("/movies", app.MovieCatalog) // real route is "/admin/movies" but "/admin" part is not required
		mux.Get("/movies/{id}", app.MovieForEdit)

		mux.Put("/movies/0", app.InsertMovie)      // insert a new movie
		mux.Patch("/movies/{id}", app.UpdateMovie) // update an existing movie

		// delete a movie
		mux.Delete("/movies/{id}", app.DeleteMovie)
	})
	return mux
}
