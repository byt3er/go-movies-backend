package repository

import (
	"backend/internals/models"
	"database/sql"
)

// pretty much everthing in go is an interface
type DatabaseRepo interface {
	Connection() *sql.DB
	AllMovie(genre ...int) ([]*models.Movie, error)
	GetUserByEmail(email string) (*models.User, error)
	GetUserByID(id int) (*models.User, error)
	OneMovie(id int) (*models.Movie, error)
	OneMovieForEdit(id int) (*models.Movie, []*models.Genre, error)
	AllGenres() ([]*models.Genre, error)
	InsertMovie(movie models.Movie) (int, error)
	UpdateMovieGenre(id int, genresIDs []int) error
	UpdateMovie(movie models.Movie) error
	DeleteMovie(id int) error
}
