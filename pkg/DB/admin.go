package dbase

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

func IsAdmin(username string, password string) (bool, error) {
	db, err := sql.Open("postgres", connectionDB)

	if err != nil {
		log.Fatal("error", err)
	}
	defer db.Close()

	var passcheck string

	err = db.QueryRow("SELECT pass FROM UserLP WHERE name = $1", username).Scan(&passcheck)
	if err != nil {
		return false, nil
	}
	return CheckPassword(password, passcheck), err
}

func Block(username string) error {
	db, err := sql.Open("postgres", connectionDB)
	defer db.Close()
	if err != nil {
		return err
	}
	_, err = db.Exec("DELETE FROM UserLP WHERE name = $1", username)
	return err
}
