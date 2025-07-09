package news

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	_ "github.com/lib/pq"
)

type Post struct {
	Name string
	Text string
}

var connectionDB string = fmt.Sprintf("host=%s user=%s dbname=%s password=%s sslmode=%s", os.Getenv("DB_HOST"), os.Getenv("DB_USER"), os.Getenv("DB_NAME"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_SSLMODE"))

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
        SELECT name, post
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
