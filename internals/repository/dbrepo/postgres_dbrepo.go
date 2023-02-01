package dbrepo

import (
	"backend/internals/models"
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"
)

type PostgresDBRepo struct {
	DB *sql.DB // holds connections to the database
}

const dbTimeout = time.Second * 3 // I'm going to give you 3 seconds to interact with the database
// if id takes longer than 3 seconds I'm going to simply cancel you request.

func (m *PostgresDBRepo) Connection() *sql.DB {
	return m.DB
}

func (m *PostgresDBRepo) AllMovie(genre ...int) ([]*models.Movie, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	where := ""
	if len(genre) > 0 {
		where = fmt.Sprintf("where id in (select movie_id from movies_genres where genre_id = %d)", genre[0])
	}

	// lets write some SQL that will connect to the database and
	// get a list of all movies
	query := fmt.Sprintf(`
		select 
			id, title, release_date, runtime,
			mpaa_rating, description, coalesce(image, ''),
			created_at, updated_at
		from 
			movies %s
		order by
			title
	`, where)
	rows, err := m.DB.QueryContext(ctx, query)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	// close the rows when you're done with them
	// otherwise that connection to the database stays open and its
	// stays open for too long and if you do this too often ,
	// you will run out of database connections,
	// and everything runs into a halt
	defer rows.Close()

	var movies []*models.Movie

	for rows.Next() {
		var movie models.Movie
		err := rows.Scan(
			&movie.ID,
			&movie.Title,
			&movie.ReleaseDate,
			&movie.RunTime,
			&movie.MPAARating,
			&movie.Description,
			&movie.Image,
			&movie.CreatedAt,
			&movie.UpdateAt,
		)
		if err != nil {
			return nil, err
		}
		movies = append(movies, &movie)
	}

	return movies, nil
}

// to get movies that are being displayed to the public
func (m *PostgresDBRepo) OneMovie(id int) (*models.Movie, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	query := `select id, title, release_date, runtime, mpaa_rating,
		description, coalesce(image, ''), created_at, updated_at
		from movies where id = $1`

	row := m.DB.QueryRowContext(ctx, query, id)

	var movie models.Movie

	err := row.Scan(
		&movie.ID,
		&movie.Title,
		&movie.ReleaseDate,
		&movie.RunTime,
		&movie.MPAARating,
		&movie.Description,
		&movie.Image,
		&movie.CreatedAt,
		&movie.UpdateAt,
	)

	if err != nil {
		return nil, err
	}

	// get genres, if any
	query = `select g.id, g.genre from movies_genres mg
		left join genres g on (mg.genre_id = g.id)
		where mg.movie_id = $1
		order by g.genre`

	rows, err := m.DB.QueryContext(ctx, query, id)
	if err != nil && err != sql.ErrNoRows { // there is no row
		return nil, err
	}
	// close the connection and return it to the pool to be used somewhere else.
	defer rows.Close()

	var genres []*models.Genre
	for rows.Next() {
		var g models.Genre
		err := rows.Scan(
			&g.ID,
			&g.Genre,
		)
		if err != nil {
			return nil, err
		}
		genres = append(genres, &g)
	}
	movie.Genres = genres

	return &movie, err
}

func (m *PostgresDBRepo) OneMovieForEdit(id int) (*models.Movie, []*models.Genre, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	query := `select id, title, release_date, runtime, mpaa_rating,
		description, coalesce(image, ''), created_at, updated_at
		from movies where id = $1`

	row := m.DB.QueryRowContext(ctx, query, id)

	var movie models.Movie

	err := row.Scan(
		&movie.ID,
		&movie.Title,
		&movie.ReleaseDate,
		&movie.RunTime,
		&movie.MPAARating,
		&movie.Description,
		&movie.Image,
		&movie.CreatedAt,
		&movie.UpdateAt,
	)

	if err != nil {
		return nil, nil, err
	}

	// get genres, if any
	query = `select g.id, g.genre from movies_genres mg
		left join genres g on (mg.genre_id = g.id)
		where mg.movie_id = $1
		order by g.genre`

	rows, err := m.DB.QueryContext(ctx, query, id)
	if err != nil && err != sql.ErrNoRows { // there is no row
		return nil, nil, err
	}
	// close the connection and return it to the pool to be used somewhere else.
	defer rows.Close()

	var genres []*models.Genre
	var genresArray []int
	for rows.Next() {
		var g models.Genre
		err := rows.Scan(
			&g.ID,
			&g.Genre,
		)
		if err != nil {
			return nil, nil, err
		}
		genres = append(genres, &g)
		genresArray = append(genresArray, g.ID)
	}
	movie.Genres = genres
	movie.GenresArray = genresArray

	var allGenres []*models.Genre

	query = `select id, genre from genres order by genre`
	gRows, err := m.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, nil, err
	}
	defer gRows.Close()

	for gRows.Next() {
		var g models.Genre
		err := gRows.Scan(
			&g.ID,
			&g.Genre,
		)
		if err != nil {
			return nil, nil, err
		}
		allGenres = append(allGenres, &g)
	}

	return &movie, allGenres, err
}

func (m *PostgresDBRepo) GetUserByEmail(email string) (*models.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	query := `select id, email, first_name, last_name, password,
				created_at, updated_at from users where email = $1`

	var user models.User
	row := m.DB.QueryRowContext(ctx, query, email)

	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.FirstName,
		&user.LastName,
		&user.Password,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (m *PostgresDBRepo) GetUserByID(id int) (*models.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()
	query := `select id, email, first_name, last_name, password,
				created_at, updated_at from users where id = $1`

	var user models.User
	row := m.DB.QueryRowContext(ctx, query, id)

	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.FirstName,
		&user.LastName,
		&user.Password,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (m *PostgresDBRepo) AllGenres() ([]*models.Genre, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	query := `select id, genre, created_at, updated_at from genres order by genre`

	rows, err := m.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var genres []*models.Genre

	for rows.Next() {
		var g models.Genre
		err := rows.Scan(
			&g.ID,
			&g.Genre,
			&g.CreatedAt,
			&g.UpdateAt,
		)
		if err != nil {
			return nil, err
		}

		genres = append(genres, &g)
	}
	return genres, nil
}

func (m *PostgresDBRepo) InsertMovie(movie models.Movie) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	stmt := `insert into movies (title, description, release_date, runtime,
			mpaa_rating, created_at, updated_at, image) values ($1, $2, $3, 
			$4, $5, $6, $7, $8) returning id`

	var newID int

	err := m.DB.QueryRowContext(ctx, stmt,
		movie.Title,
		movie.Description,
		movie.ReleaseDate,
		movie.RunTime,
		movie.MPAARating,
		movie.CreatedAt,
		movie.UpdateAt,
		movie.Image).Scan(&newID)

	if err != nil {
		log.Println(err)
		return 0, err
	}

	return newID, nil
}

func (m *PostgresDBRepo) UpdateMovie(movie models.Movie) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	stmt := `update movies set title = $1, description = $2, release_date = $3, 
			runtime = $4, mpaa_rating = $5, updated_at = $6, image = $7 
			where id = $8`

	_, err := m.DB.ExecContext(ctx, stmt,
		movie.Title,
		movie.Description,
		movie.ReleaseDate,
		movie.RunTime,
		movie.MPAARating,
		movie.UpdateAt,
		movie.Image,
		movie.ID,
	)
	if err != nil {
		return err
	}

	return nil

}

func (m *PostgresDBRepo) UpdateMovieGenre(id int, genresIDs []int) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	// the easy way to do this is to delete from the table
	stmt := `delete from movies_genres where movie_id = $1`

	_, err := m.DB.ExecContext(ctx, stmt, id)
	if err != nil {
		return err
	}
	// at this point in the execution, there are no rows in movies_genres
	// that have the movie_id that we received as call to this function

	// insert the new genres
	for _, n := range genresIDs {
		stmt := `insert into movies_genres (movie_id, genre_id) values ($1, $2)`
		_, err := m.DB.ExecContext(ctx, stmt, id, n)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *PostgresDBRepo) DeleteMovie(id int) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	// delete the movie from the movies
	stmt := `delete from movies where id = $1`
	// we don't have to delete the genres because in the database
	// we have the genres setup with movies table with foreign key relations
	// so when we delete a movie it's genres gets deleted automatically

	_, err := m.DB.ExecContext(ctx, stmt, id)
	if err != nil {
		return err
	}
	s := `some text,`
	s = strings.TrimRight(s, ",")
	return nil
}
