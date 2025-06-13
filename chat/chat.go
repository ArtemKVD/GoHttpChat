package chat

import (
	"database/sql"
	"sync"

	_ "github.com/lib/pq"
)

var mutex sync.Mutex

const connectionDB = "user=postgres dbname=Users password=admin sslmode=disable"

type Message struct {
	ID    int
	Sname string
	Gname string
	Text  string
}

func Send(User string, UserFriend string, message string) error {
	db, err := sql.Open("postgres", connectionDB)
	defer db.Close()
	db.Exec(`INSERT INTO messages (user_id, userfriend_id, message_text) VALUES ($1, $2, $3)`, User, UserFriend, message)
	if err != nil {
		panic("rw")
	}
	return err
}

func Messagelist(user1 string, user2 string) ([]Message, error) {
	db, err := sql.Open("postgres", connectionDB)
	if err != nil {
		return nil, err
	}
	Mlist, err := db.Query(`SELECT id, user_id, userfriend_id, message_text FROM messages WHERE(user_id = $1 AND userfriend_id = $2) OR (user_id = $2 AND userfriend_id = $1)`, user1, user2)
	if err != nil {
		return nil, err
	}
	var messages []Message
	for Mlist.Next() {
		var m Message
		err := Mlist.Scan(&m.ID, &m.Sname, &m.Gname, &m.Text)
		if err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, nil
}
