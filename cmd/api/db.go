package main

import (
	"database/sql"
	"log"

	_ "github.com/jackc/pgconn"
	_ "github.com/jackc/pgx/v4"
	_ "github.com/jackc/pgx/v4/stdlib"
)

func openDB(dsn string) (*sql.DB, error) {
	// *sql.DB pointer to a pool of database connections
	// go's driver for sql is smart enough to open a connection pool
	// and use the connectios from the pool when necessary and
	// return them to the pool when they're not being used anymore
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	// if can't ping the database that I m not really connected
	err = db.Ping()
	if err != nil {
		return nil, err
	}

	return db, nil

}

func (app *application) connectToDB() (*sql.DB, error) {
	connection, err := openDB(app.DSN)
	if err != nil {
		return nil, err
	}
	log.Println("connected to postgres!")
	return connection, nil
}
