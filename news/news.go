package news

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/lib/pq"
)

type Post struct {
	Name string
	Text string
}

const connectionDB = "user=postgres dbname=Users password=admin sslmode=disable"

func Postcreate(postinput Post) error {
	db, err := sql.Open("postgres", connectionDB)
	if err != nil {
		return err
	}
	defer db.Close()
	db.Exec("INSERT INTO news(name, post) VALUES ($1, $2)", postinput.Name, postinput.Text)
	return err
}

func GetFriendsNews(friends []string) ([]Post, error) {
	db, err := sql.Open("postgres", connectionDB)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	sqlval := make([]string, len(friends))
	friendlist := make([]interface{}, len(friends))
	for i, friend := range friends {
		sqlval[i] = fmt.Sprintf("$%d", i+1)
		friendlist[i] = friend
	}

	query := fmt.Sprintf(`
        SELECT name, post,
        FROM news 
        WHERE name IN (%s)
        LIMIT 100
    `, strings.Join(sqlval, ","))

	rows, err := db.Query(query, friendlist...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var news []Post
	for rows.Next() {
		var newpost Post
		if err := rows.Scan(&newpost.Name, &newpost.Text); err != nil {
			return nil, err
		}
		news = append(news, newpost)
	}

	return news, nil
}
