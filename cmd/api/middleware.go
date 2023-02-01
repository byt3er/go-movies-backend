package main

import "net/http"

// middlewares
// middleware is logic that runs on a request
// so the before it's given to the handler
// middleware runs against it
// (here all we're doing is modifying the request
// as it comes in.)
func (app *application) enableCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// set header
		// any http request should be permitted by our system
		//w.Header().Set("Access-Control-Allow-Origin", "http://localhost:8080")
		w.Header().Set("Access-Control-Allow-Origin", "http://*")
		//w.Header().Set("Access-Control-Allow-Origin", "https://learn-code.ca")

		// if we have a request named "OPTIONS"
		// OPTIONS request
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			// with forms we use GET and POST
			// but with REST api we need to allow other as well
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, X-CSRF-Token, Authorization")
			return
		} else {
			h.ServeHTTP(w, r)
		}
	})
}

func (app *application) authRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// we don't care about the token and claims
		// we only need to check for error
		_, _, err := app.auth.GetTokenFromHeaderAndVerify(w, r)
		if err != nil {
			// then the user is not authorize
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
