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

func CheckLogPass(name string, pass string) (bool, error) {
	db, err := sql.Open("postgres", connectionDB)
	defer db.Close()
	if err != nil {
		log.Fatal("error", err)
	}

	var passcheck string

	err = db.QueryRow("SELECT pass FROM UserLP WHERE name = $1", name).Scan(&passcheck)
	return passcheck == pass, err
}

func AddFriend(username string, friendname string) error {
	db, err := sql.Open("postgres", connectionDB)
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec("INSERT INTO friends (user_id, friend_id) VALUES ($1, $2)", username, friendname)
	return err
}

func GetFriends(username string) ([]string, error) {
	db, err := sql.Open("postgres", connectionDB)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query("SELECT friend_id FROM friends WHERE user_id = $1", username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var friends []string
	for rows.Next() {
		var friend string
		if err := rows.Scan(&friend); err != nil {
			return nil, err
		}
		friends = append(friends, friend)
	}
	return friends, nil
}

func Userlist() ([]string, error) {
	db, err := sql.Open("postgres", connectionDB)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.Query("SELECT name FROM UserLP")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []string
	for rows.Next() {
		var username string
		users = append(users, username)
	}
	return users, err
}
