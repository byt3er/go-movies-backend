package models

import "time"

// anytime we want to deal with a movie we can use this Movie type
type Movie struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	ReleaseDate time.Time `json:"release_date"`
	RunTime     int       `json:"runtime"`
	MPAARating  string    `json:"mpaa_rating"`
	Description string    `json:"description"`
	Image       string    `json:"image"`
	CreatedAt   time.Time `json:"-"` // "-" means don't include it in JSON
	UpdateAt    time.Time `json:"-"` // ingore this field in JSON
	Genres      []*Genre  `json:"genres,omitempty"`
	GenresArray []int     `json:"genres_array,omitempty"`
}

type Genre struct {
	ID        int       `json:"id"`
	Genre     string    `json:"genre"`
	Checked   bool      `json:"checked"`
	CreatedAt time.Time `json:"-"` // "-" means don't include it in JSON
	UpdateAt  time.Time `json:"-"` // ingore this field in JSON
}