package main

import (
	"log"
	"net/http"
	"text/template"

	chat "github.com/ArtemKVD/HttpChatGo/chat"
	news "github.com/ArtemKVD/HttpChatGo/news"
	db "github.com/ArtemKVD/HttpChatGo/pkg/DB"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

var name string
var pass string
var sessions = make(map[string]string)

func createSession(username string, w http.ResponseWriter) {
	sessionID := uuid.New().String()
	sessions[sessionID] = username

	http.SetCookie(w, &http.Cookie{
		Name:  "session",
		Value: sessionID,
		Path:  "/",
	})
}

func SessionUsername(r *http.Request) (string, bool) {
	cookie, err := r.Cookie("session")
	if err != nil {
		return "", false
	}
	username, ok := sessions[cookie.Value]
	return username, ok
}

func destroySession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   10000,
	})
}

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := SessionUsername(r); !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /register", func(w http.ResponseWriter, r *http.Request) {

		tmpl, err := template.New("register").Parse(`
			<!DOCTYPE html>
			<html>
			<head>
			</head>
			<body>
				<h1>sign up</h1>
				<form method="POST" action="/register">
					<label for="name">name:</label><br>
					<input type="text" id="name" name="name" required><br>
					<label for="pass">pass:</label><br>
					<input type="number" id="password" name="pass" required><br>
					<label for="pass2">pass2:</label><br>
					<input type="number" id="password2" name="pass2" required><br>
					<button>send</button>
				</form>
			</body>
			</html>
		`)

		if err != nil {
			log.Fatal("error:", err)
		}

		w.Header().Set("Content-Type", "text/html")
		tmpl.Execute(w, nil)
	})
	mux.HandleFunc("POST /register", func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			http.Error(w, "request error", http.StatusBadRequest)
			return
		}

		name := r.FormValue("name")
		pass := r.FormValue("pass")
		pass2 := r.FormValue("pass2")

		if pass != pass2 {
			http.Error(w, "password1 != password2", http.StatusInternalServerError)
		} else {
			err := db.UsernameInsert(name, pass)
			if err != nil {
				http.Error(w, "DB not accept you", http.StatusInternalServerError)
			}
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
			<!DOCTYPE html>
			<html>
			<body>
				<h1>You got account </h1>
				<p>Welcome, ` + name + `!</p>
				<a href="/login">login</a>
			</body>
			</html>
		`))
	})

	mux.HandleFunc("GET /login", func(w http.ResponseWriter, r *http.Request) {

		tmpl, err := template.New("login").Parse(`
			<!DOCTYPE html>
			<html>
			<head>
			</head>
			<body>
				<h1>login</h1>
				<form method="POST" action="/login">
					<label for="name">name:</label><br>
					<input type="text" id="nametologin" name="nametologin" required><br>
					<label for="pass">pass:</label><br>
					<input type="number" id="passwordtologin" name="passtologin" required><br>
					<button>send</button>
				</form>
			</body>
			</html>
		`)

		if err != nil {
			log.Fatal("error:", err)
		}

		w.Header().Set("Content-Type", "text/html")
		tmpl.Execute(w, nil)

	})

	mux.HandleFunc("POST /login", func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()

		if err != nil {
			http.Error(w, "request error", http.StatusBadRequest)
			return
		}

		name := r.FormValue("nametologin")
		pass := r.FormValue("passtologin")
		Check, err := db.CheckLogPass(name, pass)

		if !Check {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`
            <!DOCTYPE html>
            <html>
            <body>
                <h1>Invalid login or password</h1>
                <a href="/login">Try again</a>
            </body>
            </html>
        `))
			return
		}

		tmpl, err := template.New("join").Parse(`
        <!DOCTYPE html>
        <html>
        <head>
        </head>
        <body>
            <h1>hello {{.Name}}!</h1>
            <a href="/login">Quit</a>
			<a href="/friends">Manage Friends</a>
        </body>
        </html>
    	`)

		if err != nil {
			http.Error(w, "error:", http.StatusInternalServerError)
			return
		}

		createSession(name, w)
		http.Redirect(w, r, "/friends", http.StatusSeeOther)

		w.Header().Set("Content-Type", "text/html")
		tmpl.Execute(w, struct {
			Name string
		}{
			Name: name,
		})
	})
	mux.HandleFunc("GET /friends", func(w http.ResponseWriter, r *http.Request) {
		username, _ := SessionUsername(r)

		friends, err := db.GetFriends(username)
		if err != nil {
			http.Error(w, "Failed to get friends list", http.StatusInternalServerError)
			return
		}

		tmpl := template.Must(template.New("friends").Parse(`
        <!DOCTYPE html>
        <html>
        <head>
            <title>Friends</title>
        </head>
        <body>
            <h1>Your Friends, {{.Username}}</h1>
			<ul>
    			{{range .Friends}}
    			<li>
       				<a href="/chat/{{.}}">{{.}}</a>
    			</li>
    			{{end}}
			</ul>
            
            <h2>Add Friend</h2>
            <form method="POST" action="/add_friend">
                <input type="text" name="friendname" placeholder="friend name" required>
                <button type="submit">add friend</button>
            </form>
            
            <form method="GET" action="/login">
                <button type="submit">logout</button>
            </form>
			<form method="POST" action="/news">
                <button type="submit">NEWS</button>
            </form>

        </body>
        </html>
    `))

		w.Header().Set("Content-Type", "text/html")
		tmpl.Execute(w, struct {
			Username string
			Friends  []string
		}{
			Username: username,
			Friends:  friends,
		})
	})

	mux.HandleFunc("POST /add_friend", authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		username, _ := SessionUsername(r)
		friendname := r.FormValue("friendname")

		if err := db.AddFriend(username, friendname); err != nil {
			http.Error(w, "Failed to add friend", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/friends", http.StatusSeeOther)
	}))

	mux.HandleFunc("GET /chat/{friend}", authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		user, _ := SessionUsername(r)
		friend := r.PathValue("friend")
		messagelist, err := chat.Messagelist(user, friend)
		if err != nil {
			http.Error(w, "message fail", http.StatusInternalServerError)
		}
		tmpl := template.Must(template.New("chat").Parse(`
        <!DOCTYPE html>
        <html>
        <head>
        </head>
        <body>
            <h1>Chat with {{.Friend}}</h1>
			  <div id="messages">
                {{range .Messages}}
                <div>
                    {{.Sname}}: {{.Text}}
                </div>
                {{end}}
            </div>

            <form method="POST" action="/send_message">
                <input type="hidden" name="userfriend" value="{{.Friend}}">
                <textarea name="message" required></textarea>
                <button type="submit">Send</button>
            </form>
			<a href="/friends">Back to friendlist</a>
        </body>
        </html>
    `))

		w.Header().Set("Content-Type", "text/html")

		tmpl.Execute(w, struct {
			CurrentUser string
			Friend      string
			Messages    []chat.Message
		}{
			CurrentUser: user,
			Friend:      friend,
			Messages:    messagelist,
		})

	}))

	mux.HandleFunc("POST /send_message", authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		user, _ := SessionUsername(r)
		userfriend := r.FormValue("userfriend")
		message := r.FormValue("message")

		if err := chat.Send(user, userfriend, message); err != nil {
			http.Error(w, "Failed to send message", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/chat/"+userfriend, http.StatusSeeOther)
	}))

	mux.HandleFunc("GET /news/create", authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		_, ok := SessionUsername(r)
		if !ok {
			http.Error(w, "not authorized", http.StatusBadRequest)
			return
		}
		tmpl := template.Must(template.New("chat").Parse(`
		<!DOCTYPE html>
		<html>
		<head>
			<title>Create News Post</title>
			<style>
				.news-form {
					max-width: 600px;
					margin: 20px auto;
					padding: 20px;
					border: 1px solid #ddd;
					border-radius: 5px;
				}
				textarea {
					width: 100%;
					min-height: 100px;
					margin-bottom: 10px;
				}
			</style>
		</head>
		<body>
			<div class="news-form">
				<h1>Create New Post</h1>
				<form method="POST" action="/news/create">
					<textarea name="post_text" placeholder="What's new?" required></textarea>
					<button type="submit">Publish</button>
					<a href="/news" style="margin-left: 10px;">Cancel</a>
				</form>
			</div>
		</body>
		</html>`))

		w.Header().Set("Content-Type", "text/html")

		tmpl.Execute(w, nil)

		http.Redirect(w, r, "/news", http.StatusSeeOther)
	}))

	mux.HandleFunc("POST /news/create", authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		user, ok := SessionUsername(r)
		if !ok {
			http.Error(w, "Not authorized", http.StatusUnauthorized)
			return
		}
		postText := r.FormValue("post_text")

		newPost := news.Post{
			Name: user,
			Text: postText,
		}

		if err := news.Postcreate(newPost); err != nil {
			http.Error(w, "create fail", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/news", http.StatusSeeOther)
	}))

	http.ListenAndServe(":8081", mux)

}
