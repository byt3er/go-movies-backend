package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// create a Type that we can use for pretty much anything we need
// to send to the frontend
// Error = if that is true, there is some kind of error and if it's false
// everything worked as expected
// `json:"data,omitempty" means if data is not specified, if it doesn't have
// any value, then don't include that in the Json
type JSONResponse struct {
	Error   bool        `json:"error"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// writing Json
// w = ResponeWriter
// status = status code for the response
// data = payload
func (app *application) writeJSON(w http.ResponseWriter, status int, data interface{}, headers ...http.Header) error {
	// header ...http.Header ==> which means include zero or more things
	// of type http.Header, if that's not included, then we don't have any
	// extra header we want to send.
	out, err := json.Marshal(data)
	if err != nil {
		return err
	}
	// check if additional header given
	if len(headers) > 0 {
		// we are specifying an optional header
		for key, value := range headers[0] {
			w.Header()[key] = value
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, err = w.Write(out)
	if err != nil {
		return err
	}
	return nil

}

// use when people log in or when we manage our movies catalog
func (app *application) readJSON(w http.ResponseWriter, r *http.Request, data interface{}) error {
	// data is pointer and we're going to read into that when we consume the JSON

	// I don't want to ever receive/ refuse to receive any JSON file that's
	// bigger than 1MB
	maxBytes := 1024 * 1024                                  // one megabyte
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes)) // limits the size of the Body we read
	// no matter what I receive it'll be no longer than 1MB .

	dec := json.NewDecoder(r.Body) // json decoder

	// disallow unknown fields
	dec.DisallowUnknownFields()

	//try to  decode the data
	err := dec.Decode(data)
	if err != nil {
		// If I can't decode it, then something went wrong
		// either the data is too big or it's not JSON or for unknown fields
		return err
	}

	//check if there is one json file
	// if somebody sends me two Json files in a single request, then they're
	// doing something they not allowed to be doing
	err = dec.Decode(&struct{}{}) // struct{}{} ==> through away variable
	if err != io.EOF {
		// so if I only had one JSON file in the request body,
		// then I going to get an error that is EOF(end of file)
		// If I got anything else, then there's more than one json value
		// in the request body
		return errors.New("body must only contain a single JSON value")
	}

	// successfully read my JSOn into the variable
	// I don't have to return the data b/c I have received that as a pointer
	return nil
}

// write error messages as json
func (app *application) errorJSON(w http.ResponseWriter, err error, status ...int) error {
	// default status code
	statusCode := http.StatusBadRequest

	//check for optional parameter for status code
	if len(status) > 0 {
		statusCode = status[0]
	}

	var payload JSONResponse
	payload.Error = true
	payload.Message = err.Error()

	return app.writeJSON(w, statusCode, payload)
}
