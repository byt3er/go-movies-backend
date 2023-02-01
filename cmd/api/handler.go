package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"backend/internals/graph"
	"backend/internals/models"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v4"
)

// every handler in go takes two arguments
//first:: w http.ResponseWriter is where you write the final content you
// want to send tothe client.
//Second: r http.Request

// default route to our api
func (app *application) Home(w http.ResponseWriter, r *http.Request) {
	// just write hello world to the browser
	//fmt.Fprintf(w, "Hello, world from %s", app.Domain)

	// this is what we're going to send back
	var payload = struct {
		// specifiying the fields
		Status  string `json:"status"`
		Message string `json:"message"`
		Version string `json:"version"`
	}{
		Status:  "active",
		Message: "Go Movies up and running",
		Version: "1.0.0",
	}

	_ = app.writeJSON(w, http.StatusOK, &payload)

}

func (app *application) AllMovie(w http.ResponseWriter, r *http.Request) {
	movies, err := app.DB.AllMovie()
	if err != nil {
		app.errorJSON(w, err) // badRequst
		return
	}

	_ = app.writeJSON(w, http.StatusOK, movies)
}

func (app *application) authenticate(w http.ResponseWriter, r *http.Request) {
	//********* read json payload***********
	// because we're going to receive a username and a password
	// or an email and a password as JSON
	var requestPayload struct { // ==> this is what we're getting from the client
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	fmt.Println("/authenticate got hit.")
	err := app.readJSON(w, r, &requestPayload) // ==> &requestPayload is very important
	if err != nil {
		app.errorJSON(w, err, http.StatusBadRequest)
		return
	}

	// validate user against database
	user, err := app.DB.GetUserByEmail(requestPayload.Email)
	if err != nil {
		app.errorJSON(w, errors.New("invalid credentials"), http.StatusBadRequest)
		return
	}

	//check password
	valid, err := user.PasswordMatches(requestPayload.Password)
	if err != nil || !valid {
		app.errorJSON(w, errors.New("invalid credentials"), http.StatusBadRequest)
		return
	}

	//create a jwt user
	u := jwtUser{
		ID:        user.ID,
		FirstName: user.FirstName,
		LastName:  user.LastName,
	}

	// generate tokens
	tokens, err := app.auth.GenerateTokenPair(&u)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	// log.Println(tokens.Token)
	refreshCookie := app.auth.GetRefreshCookie(tokens.RefreshToken)

	// write cookie to the browser
	http.SetCookie(w, refreshCookie)

	// w.Write([]byte(tokens.Token))
	app.writeJSON(w, http.StatusAccepted, tokens)
	fmt.Println("/authenticate got hit.")
}

func (app *application) refreshToken(w http.ResponseWriter, r *http.Request) {
	// NOTE: now to get the cookies or the cookie we want, we actually
	// have to range through all of the cookies that are sent to us.
	// Remember, we sent that cookie as an HTTP only cookie.
	// so there is no access to it through Javascript
	// But every request made from a user that has that Cookie will include
	// in a request. So all we have to do is range through the cookies
	for _, cookie := range r.Cookies() {
		if cookie.Name == app.auth.CookieName {
			// then at this point we have to refresh using that cookie
			// we're going to take that refresh token that's embedded in that
			// cookie, validate it, and if it has expired, we'll issue new tokens
			claims := &Claims{}
			refreshToken := cookie.Value

			// parse the token to get the claim
			_, err := jwt.ParseWithClaims(refreshToken, claims, func(token *jwt.Token) (interface{}, error) {
				return []byte(app.JWTSecret), nil
			})
			// we have check this >err< , because if anything goes wrong with
			// this token, for example, if it's expired, the the User is not authorized
			if err != nil {
				app.errorJSON(w, errors.New("unathorized"), http.StatusUnauthorized)
				return
			}

			//***************
			//if we get this far, then we've managed to parse this cookie or
			// parse this token successfully from the cookie.
			// ********************

			// get the user id from the token claims
			userID, err := strconv.Atoi(claims.Subject)
			if err != nil {
				app.errorJSON(w, errors.New("unknown user"), http.StatusUnauthorized)
				return
			}
			// now I have the user id
			// try to refresh this user and give that user a new tokens
			// but before that make sure the user exits in the database

			user, err := app.DB.GetUserByID(userID)
			if err != nil {
				app.errorJSON(w, errors.New("unknown user"), http.StatusUnauthorized)
				return
			}

			u := jwtUser{
				ID:        user.ID,
				FirstName: user.FirstName,
				LastName:  user.LastName,
			}

			// Generate a new token pair
			tokenPairs, err := app.auth.GenerateTokenPair(&u)
			if err != nil {
				app.errorJSON(w, errors.New("error generating error"), http.StatusUnauthorized)
				return
			}

			//set the  refresh token cookie
			http.SetCookie(w, app.auth.GetRefreshCookie(tokenPairs.RefreshToken))

			// send back the JSON
			app.writeJSON(w, http.StatusOK, tokenPairs)

		}
	}

}

func (app *application) logout(w http.ResponseWriter, r *http.Request) {
	// just set a cookie
	http.SetCookie(w, app.auth.GetExpiredRefreshCookie())

	// set the header
	w.WriteHeader(http.StatusAccepted)
}

func (app *application) MovieCatalog(w http.ResponseWriter, r *http.Request) {
	movies, err := app.DB.AllMovie()
	if err != nil {
		app.errorJSON(w, err) // badRequst
		return
	}

	_ = app.writeJSON(w, http.StatusOK, movies)
}

// path: /movie/1
func (app *application) GetMovie(w http.ResponseWriter, r *http.Request) {
	// get the id from the URL
	id := chi.URLParam(r, "id")
	movieID, err := strconv.Atoi(id)
	if err != nil {
		log.Println("id:", id)
		app.errorJSON(w, err)
		return
	}

	movie, err := app.DB.OneMovie(movieID)
	if err != nil {
		app.errorJSON(w, err)
	}

	_ = app.writeJSON(w, http.StatusOK, movie)

}

func (app *application) MovieForEdit(w http.ResponseWriter, r *http.Request) {
	// get the id from the URL
	id := chi.URLParam(r, "id")
	movieID, err := strconv.Atoi(id)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	movie, genres, err := app.DB.OneMovieForEdit(movieID)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	var payload = struct {
		Movie  *models.Movie   `json:"movie"`
		Genres []*models.Genre `json:"genres"`
	}{
		movie,
		genres,
	}

	_ = app.writeJSON(w, http.StatusOK, payload)

}

func (app *application) AllGenres(w http.ResponseWriter, r *http.Request) {
	genres, err := app.DB.AllGenres()
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	_ = app.writeJSON(w, http.StatusOK, genres)
}

// receives JSON payload from the frontend and try to insert into the database
// and also try to go to a remote third-party api and look for an image for
// the movie
func (app *application) InsertMovie(w http.ResponseWriter, r *http.Request) {
	log.Println("Insert movie got hit!")
	var movie models.Movie

	err := app.readJSON(w, r, &movie)
	if err != nil {
		log.Println(err)
		app.errorJSON(w, err)
		return
	}
	// fmt.Printf("movie: %v\n", movie)
	// fmt.Printf("movie genres: %v", movie.Genres)
	// try to get an image
	movie = app.getPoster(movie)
	// I should have a new movie variable with a poster included
	movie.CreatedAt = time.Now()
	movie.UpdateAt = time.Now()

	// now we've inserted a movie into the database with an image(if we could
	// find one ) and get the id of the new movie in movies table
	newID, err := app.DB.InsertMovie(movie)
	if err != nil {
		log.Println("error inserting movie")
		app.errorJSON(w, err)
		return
	}

	// now handle genres
	err = app.DB.UpdateMovieGenre(newID, movie.GenresArray)
	if err != nil {
		log.Println("error updating movie genres", err)
		app.errorJSON(w, err)
		return
	}

	resp := JSONResponse{
		Error:   false,
		Message: "movie updated",
	}
	app.writeJSON(w, http.StatusAccepted, resp)
}

// we're getting the movie poster
func (app *application) getPoster(movie models.Movie) models.Movie {
	// must match the structure of JSON , receiving from remote source
	type TheMovieDB struct {
		Page    int `json:"page"`
		Results []struct {
			PosterPath string `json:"poster_path"`
		} `json:"results"`
		TotalPages int `json:"total_pages"`
	}

	client := &http.Client{}
	theUrl := fmt.Sprintf("https://api.themoviedb.org/3/search/movie/?api_key=%s", app.APIKey)
	log.Println(theUrl)
	req, err := http.NewRequest("GET", theUrl+"&query="+url.QueryEscape(movie.Title), nil)
	if err != nil {
		log.Println(err)
		return movie
	}

	// add headers to the request
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json") // go practice to specify

	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return movie
	}
	defer resp.Body.Close() // to avoid resource leak

	// read the body of the request(we are getting our response in the form
	// of bytes)
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return movie
	}

	// unmarshal the request body and put it into responseObject
	var responseObject TheMovieDB

	json.Unmarshal(bodyBytes, &responseObject)

	if len(responseObject.Results) > 0 {
		// I have atleast one movie in the response
		movie.Image = responseObject.Results[0].PosterPath
	}
	return movie
}

// update a movie
func (app *application) UpdateMovie(w http.ResponseWriter, r *http.Request) {
	// this is going to receive a paylaod.
	// and the payload describe the movie
	var payload models.Movie

	log.Println("UpdateMovie got hit!")

	err := app.readJSON(w, r, &payload)
	if err != nil {
		log.Println("failed to read json ", err)
		app.errorJSON(w, err)
		return
	}
	// if we reach to this point, we have the payload

	// get the existing record (movie) from the database
	movie, err := app.DB.OneMovie(payload.ID)
	if err != nil {
		log.Println("failed to get the movie from database ", err)
		app.errorJSON(w, err)
		return
	}

	movie.Title = payload.Title
	movie.ReleaseDate = payload.ReleaseDate
	movie.Description = payload.Description
	movie.MPAARating = payload.MPAARating
	movie.RunTime = payload.RunTime
	movie.UpdateAt = time.Now()

	err = app.DB.UpdateMovie(*movie)
	if err != nil {
		log.Println("failed to update a movie ", err)
		app.errorJSON(w, err)
		return
	}

	// handle the genres
	err = app.DB.UpdateMovieGenre(movie.ID, payload.GenresArray)
	if err != nil {
		log.Println("failed to update movie genres ", err)
		app.errorJSON(w, err)
		return
	}

	// response
	resp := JSONResponse{
		Error:   false,
		Message: "movie updated",
	}

	app.writeJSON(w, http.StatusAccepted, resp)
}

func (app *application) DeleteMovie(w http.ResponseWriter, r *http.Request) {
	// we're getting the {id} for the movie from the url params
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	err = app.DB.DeleteMovie(id)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	// response
	resp := JSONResponse{
		Error:   false,
		Message: "movie deleted",
	}
	app.writeJSON(w, http.StatusAccepted, resp)
}

// return a list of movies for a particular genre
func (app *application) AllMoviesByGenre(w http.ResponseWriter, r *http.Request) {
	log.Println("AllMoviesByGenre got hit")
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	log.Println("id:", id)
	if err != nil {
		log.Println("id not found in the url")
		app.errorJSON(w, err)
		return
	}

	movies, err := app.DB.AllMovie(id)
	if err != nil {
		log.Println("failed to get all movies by genre id")
		app.errorJSON(w, err)
		return
	}
	if len(movies) > 0 {
		app.writeJSON(w, http.StatusOK, movies)
	} else {
		log.Println("No movies found.")
		resp := JSONResponse{
			Error:   true,
			Message: "No movie found by genre",
		}
		app.writeJSON(w, http.StatusOK, resp)
	}

	log.Println("response StatusOK")
}

func (app *application) moviesGraphQL(w http.ResponseWriter, r *http.Request) {
	// we need to populate our Graph type with the movies
	movies, _ := app.DB.AllMovie()

	//Note: our request is not going to be in the form of JSON, instead will
	// be in the form of the syntax used by GraphQL, but it's still
	// going to be in the body of the request

	// get the query from the request
	q, _ := io.ReadAll(r.Body) //  []byte
	query := string(q)         // now I have query (q) as string

	log.Println(query)

	// create a new variable of type *graph.Graph

	g := graph.New(movies)

	// set the query string on the variable
	g.QueryString = query

	// perform the query
	resp, err := g.Query()
	if err != nil {
		log.Println("fail to perform the query ", err)
		app.errorJSON(w, err)
		return
	}

	// send the response
	json, _ := json.MarshalIndent(resp, "", "\t")
	log.Println("response:", string(json))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(json)
}
