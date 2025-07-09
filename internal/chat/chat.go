package chat

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"sync"

	_ "github.com/lib/pq"
)

var mutex sync.Mutex

var connectionDB string = fmt.Sprintf("host=%s user=%s dbname=%s password=%s sslmode=%s", os.Getenv("DB_HOST"), os.Getenv("DB_USER"), os.Getenv("DB_NAME"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_SSLMODE"))

type Message struct {
	ID    int
	Sname string
	Gname string
	Text  string
}

func Send(User string, UserFriend string, message string) error {
	db, err := sql.Open("postgres", connectionDB)
	if err != nil {
		log.Printf("DB connection error: %v", err)
		return err
	}
	defer db.Close()
	_, err = db.Exec(`INSERT INTO messages (user_id, userfriend_id, message_text) VALUES ($1, $2, $3)`, User, UserFriend, message)
	if err != nil {
		log.Printf("insert message error")
	}
	log.Println("Message saved")
	return err
}

func Messagelist(user1 string, user2 string) ([]Message, error) {
	db, err := sql.Open("postgres", connectionDB)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	Mlist, err := db.Query(`SELECT id, user_id, userfriend_id, message_text FROM messages WHERE(user_id = $1 AND userfriend_id = $2) OR (user_id = $2 AND userfriend_id = $1)`, user1, user2)
	if err != nil {
		return nil, err
	}
	defer Mlist.Close()
	var messages []Message
	for Mlist.Next() {
		var m Message
		err := Mlist.Scan(&m.ID, &m.Sname, &m.Gname, &m.Text)
		if err != nil {
			return nil, err
		}
		log.Printf("found message %v", m)
		messages = append(messages, m)
	}
	return messages, nil
}
