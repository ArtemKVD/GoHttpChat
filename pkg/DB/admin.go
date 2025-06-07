package dbase

import (
	"database/sql"

	_ "github.com/lib/pq"
)

// admin

func IsAdmin(username string, password string) (bool, error) {
	var pass string
	db, err := sql.Open("postgres", connectionDB)
	defer db.Close()
	if err != nil {
		return false, err
	}
	err = db.QueryRow("SELECT pass FROM UserLP WHERE name = $1", username).Scan(&pass)
	if err != nil {
		return false, err
	}

	return username == "Admin" && pass == password, err
}

//block/remove

func Block(username string) error {
	db, err := sql.Open("postgres", connectionDB)
	defer db.Close()
	if err != nil {
		return err
	}
	_, err = db.Exec("DELETE FROM UserLP WHERE name = $1", username)
	return err
}
