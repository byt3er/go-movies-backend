package graph

import (
	"backend/internals/models"
	"errors"
	"strings"

	"github.com/graphql-go/graphql"
)

type Graph struct {
	// we have to populate this with our entire list of movies
	Movies []*models.Movie

	// this will be the string we receice in order to process whatever
	// it is we want to do in the backend. i.e like to get an individual movies,
	// or to get a full list of movies, whatever it may be
	QueryString string

	Config graphql.SchemaConfig
	fields graphql.Fields

	movieType *graphql.Object
}

// It's a factory method used to create a new instance of the graph type
// this is what we're going to use to populate our graph variable when
// we create one
func New(movies []*models.Movie) *Graph {
	// now this will be a relatively long function because we have to
	// define the object for our movie
	//and when I'm defining the movie object, I'm going to be describing
	// its fields and the fields in this object obviously must match database
	// field names

	var movieType = graphql.NewObject(
		graphql.ObjectConfig{
			Name: "Movie",
			Fields: graphql.Fields{
				// fields name must match database columns
				"id": &graphql.Field{
					Type: graphql.Int,
				},
				"title": &graphql.Field{
					Type: graphql.String,
				},
				"description": &graphql.Field{
					Type: graphql.String,
				},
				"release_date": &graphql.Field{
					Type: graphql.DateTime,
				},
				"runtime": &graphql.Field{
					Type: graphql.Int,
				},
				"mpaa_rating": &graphql.Field{
					Type: graphql.String,
				},
				"created_at": &graphql.Field{
					Type: graphql.DateTime,
				},
				"updated_at": &graphql.Field{
					Type: graphql.DateTime,
				},
				"image": &graphql.Field{
					Type: graphql.String,
				},
			},
		},
	)
	// the movieType acually has the information for a given movie or
	// for all the movies, depending on what you're doing with it.

	// fields defines the available actions on the data
	// like: list, search , get
	// so to define this variable fields, we need to populate it with the
	// kinds of things we're going to do with our data
	var fields = graphql.Fields{
		// action {list}
		//** List Directive **
		"list": &graphql.Field{
			Type:        graphql.NewList(movieType), // that what're dealing with, that our data
			Description: "Get all movies",

			// what happens when we execute this action(list)
			Resolve: func(params graphql.ResolveParams) (interface{}, error) {
				return movies, nil
			},
		},

		//** Search Directive ***
		// we want to search for particular movies
		"search": &graphql.Field{
			Type:        graphql.NewList(movieType),
			Description: "Search movies by title",
			// this one is going to take arguments
			// obviously, if you're searching for something,
			// you need to specify the argument that you're
			// searching for.
			Args: graphql.FieldConfigArgument{
				"titleContains": &graphql.ArgumentConfig{
					Type: graphql.String,
				},
			},

			// then we have resolve
			// how do you do this?
			Resolve: func(params graphql.ResolveParams) (interface{}, error) {
				// here we're going to returning a subset of the list of movies
				// so we're not returning the entire list
				// we 're just returning anything that matches our search parameter

				// what we're going to return
				var theList []*models.Movie
				search, ok := params.Args["titleContains"].(string)
				// then we range through all of our movies
				if ok {
					// we're just going through the entire list
					for _, currentMovie := range movies {
						// check to see if the title we're looking at contains
						// the characters we're searching for.
						// we also have to make sure that it's case insensitive
						if strings.Contains(strings.ToLower(currentMovie.Title), strings.ToLower(search)) {
							// if we found a match
							theList = append(theList, currentMovie)
						}
					}
				}
				return theList, nil
			},
		},

		// get one paricular movie
		"get": &graphql.Field{
			Type:        movieType,
			Description: "Get movie by id",
			// we're trying to get one movie, So we need to have it's ID
			Args: graphql.FieldConfigArgument{
				"id": &graphql.ArgumentConfig{
					Type: graphql.Int,
				},
			},
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				id, ok := p.Args["id"].(int)
				if ok {
					for _, movie := range movies {
						if movie.ID == id {
							// we found the movie
							return movie, nil
						}
					}
				}
				// we didn't find it
				return nil, nil
			},
		},
	}

	return &Graph{
		Movies:    movies,
		fields:    fields,
		movieType: movieType,
	}

}

// this method allow us to perform queries
func (g *Graph) Query() (*graphql.Result, error) {
	rootQuery := graphql.ObjectConfig{Name: "RootQuery", Fields: g.fields}
	schemaConfig := graphql.SchemaConfig{Query: graphql.NewObject(rootQuery)}
	schema, err := graphql.NewSchema(schemaConfig)
	if err != nil {
		return nil, err
	}

	params := graphql.Params{Schema: schema, RequestString: g.QueryString}
	resp := graphql.Do(params)

	// check for error(it's a different the way to check for errors)
	if len(resp.Errors) > 0 {
		return nil, errors.New("error executed query")
	}

	return resp, nil

}
