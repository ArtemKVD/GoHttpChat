package dbase

import (
	"database/sql"
	"log"
	"time"

	_ "github.com/lib/pq"
)

const connectionDB = "host=postgres user=postgres dbname=Users password=admin sslmode=disable"

func WaitPostgres() {
	db, err := sql.Open("postgres", connectionDB)
	if err != nil {
		log.Printf("db not open")
	}
	defer db.Close()

	for i := 0; i < 5; i++ {
		err := db.Ping()
		if err == nil {
			log.Print("Connected to Postgresql")
			return
		}
		time.Sleep(5 * time.Second)
	}
	log.Print("Not connected")
}

func UsernameInsert(name string, pass string) error {
	var a string
	db, err := sql.Open("postgres", connectionDB)

	if err != nil {
		log.Fatal("error:", err)
		return err
	}

	defer db.Close()

	CheckName := "SELECT name FROM UserLP WHERE name = $1"

	err = db.QueryRow(CheckName, name).Scan(&a)

	if err != nil && err != sql.ErrNoRows {
		log.Print("Db query error", err)
		return err
	}
	InsertQuery := "INSERT INTO UserLP (name, pass) VALUES ($1, $2)"
	result, err := db.Exec(InsertQuery, name, pass)
	if err != nil {
		log.Print("db insert error")
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Println("pizdec1")
		return err
	}

	if rowsAffected == 0 {
		log.Println("pizdec2")
	}
	return err
}

func CheckLogPass(name string, pass string) (bool, error) {
	db, err := sql.Open("postgres", connectionDB)

	if err != nil {
		log.Fatal("error", err)
	}
	defer db.Close()

	var passcheck string

	err = db.QueryRow("SELECT pass FROM UserLP WHERE name = $1", name).Scan(&passcheck)
	if err != nil {
		return false, nil
	}
	return CheckPassword(pass, passcheck), err
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
	rows, err := db.Query("SELECT name FROM UserLP WHERE name != 'Admin'")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []string
	for rows.Next() {
		var username string
		err := rows.Scan(&username)
		if err != nil {
			log.Printf("Scan error: %v", err)
			continue
		}
		users = append(users, username)
	}

	log.Printf("Admin get users list")
	return users, nil

}

func GetUserPasswordHash(username string) (string, error) {
	db, err := sql.Open("postgres", connectionDB)
	if err != nil {
		return "", err
	}
	defer db.Close()

	var hash string
	err = db.QueryRow("SELECT pass FROM UserLP WHERE name = $1", username).Scan(&hash)
	return hash, err
}
