package dbase

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

const connectionDB = "user=postgres dbname=Users password=admin sslmode=disable"

func UsernameInsert(name string, pass string) error {
	var a string
	db, err := sql.Open("postgres", connectionDB)
	defer db.Close()
	if err != nil {
		log.Fatal("error:", err)
		return err
	}
	CheckName := "SELECT name FROM UserLP WHERE name = $1"

	err = db.QueryRow(CheckName, name).Scan(&a)
	InsertQuery := "INSERT INTO UserLP (name, pass) VALUES ($1, $2)"
	result, err := db.Exec(InsertQuery, name, pass)
	if err != nil {
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		fmt.Println("pizdec1")
		return err
	}

	if rowsAffected == 0 {
		fmt.Println("pizdec2")
	}
	return err
}

func UsernameLogin(name string, pass string) {

}

func CheckLogPass(name string, pass string) bool {
	db, err := sql.Open("postgres", connectionDB)
	defer db.Close()
	if err != nil {
		log.Fatal("error", err)
	}

	var passcheck string

	err = db.QueryRow("SELECT pass FROM UserLP WHERE name = $1", name).Scan(&passcheck)
	return passcheck == pass
}
