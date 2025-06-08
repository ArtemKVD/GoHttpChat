package main

import (
	"log"
	"net/http"
	"text/template"
	"time"

	chat "github.com/ArtemKVD/HttpChatGo/chat"
	news "github.com/ArtemKVD/HttpChatGo/news"
	db "github.com/ArtemKVD/HttpChatGo/pkg/DB"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

var name string
var pass string
var sessions = make(map[string]string)

const connectionDB = "user=postgres dbname=Users password=admin sslmode=disable"

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
	shutdown := make(chan struct{})
	server := &http.Server{
		Addr:    ":8081",
		Handler: mux,
	}

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
			log.Printf("get register page error: %v", err)
		}

		w.Header().Set("Content-Type", "text/html")
		tmpl.Execute(w, nil)
	})
	mux.HandleFunc("POST /register", func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			http.Error(w, "request error", http.StatusBadRequest)
			log.Printf("request error %v", err)
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
				log.Printf("Db not accept user %v: error:%v", name, err)
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
			log.Printf("get login page error: %v", err)
		}

		w.Header().Set("Content-Type", "text/html")

		tmpl.Execute(w, nil)

	})

	mux.HandleFunc("POST /login", func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()

		if err != nil {
			http.Error(w, "request error", http.StatusBadRequest)
			log.Printf("request error: %v", err)
			return
		}

		name := r.FormValue("nametologin")
		pass := r.FormValue("passtologin")

		Check, err := db.CheckLogPass(name, pass)
		if err != nil {
			log.Printf("error check login and pass with DB: %v", err)
		}

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
			log.Printf("Join error: %v", err)
			return
		}

		createSession(name, w)
		log.Printf("User %v login at %v", name, time.Now())

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
			log.Printf("get friend list user %v error: %v", username, err)
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
			log.Printf("Add friend fail user:%v friend:%v error: %v", username, friendname, err)
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
			log.Printf("message list delivery fail by %v to %v: %v", user, friend, err)
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
			log.Printf("Send message error: %v from user: %v to user: %v", err, user, userfriend)
			return
		}

		http.Redirect(w, r, "/chat/"+userfriend, http.StatusSeeOther)
	}))

	mux.HandleFunc("GET /news", authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		currentUser, _ := SessionUsername(r)

		friends, err := db.GetFriends(currentUser)
		if err != nil {
			http.Error(w, "friendlist error", http.StatusInternalServerError)
			log.Printf("Get friend list error by user:%v : %v", currentUser, err)
			return
		}

		var posts []news.Post
		posts, err = news.GetFriendsNews(friends)
		if err != nil {
			log.Printf("news error by user %v: %v", currentUser, err)
		}

		tmpl := template.Must(template.New("news").Parse(`
         <!DOCTYPE html>
        <html>
        <head>
            <title>news</title>
            <style>
                .news-container {
                    max-width: 800px;
                    margin: 0 auto;
                    padding: 20px;
                }
                .news-item {
                    border-radius: 8px;
                    padding: 15px;
                    margin-bottom: 15px;
                }
                .news-author {
                    font-weight: bold;
                    margin-bottom: 5px;
                }
                .news-content {
                    margin: 10px 0;
                }
                textarea {
                    min-height: 100px;
                    padding: 10px;
                    margin-bottom: 10px;
                }
            </style>
        </head>
        <body>
            <div class="news-container">
                <h1>News</h1>
                
                <div class="create-form">
                    <form method="POST" action="/news">
                        <textarea name="post_text" placeholder="write your post" required></textarea>
                        <button type="submit">Post</button>
                    </form>
                </div>


                {{if .Posts}}
                    {{range .Posts}}
                    <div class="news-item">
                        <div class="news-author">{{.Name}}</div>
                        <div class="news-content">{{.Text}}</div>
                    </div>
                    {{end}}
                {{else}}
                    <p>no news</p>
                {{end}}
            </div>
        </body>
        </html>
    `))

		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		if err := tmpl.Execute(w, struct {
			Posts []news.Post
		}{
			Posts: posts,
		}); err != nil {
			log.Printf("Error executing template: %v", err)
		}
	}))

	mux.HandleFunc("POST /news", authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		currentUser, _ := SessionUsername(r)

		postText := r.FormValue("post_text")

		err := news.Postcreate(news.Post{
			Name: currentUser,
			Text: postText,
		})
		if err != nil {
			log.Printf("Post create error by %v: %v", currentUser, err)
			http.Error(w, "Failed to create post", http.StatusInternalServerError)
			return
		}

		log.Printf("User %v created a news post at %v", currentUser, time.Now())
		http.Redirect(w, r, "/news", http.StatusSeeOther)
	}))

	mux.HandleFunc("GET /admin/login", func(w http.ResponseWriter, r *http.Request) {
		tmpl := template.Must(template.New("adminLogin").Parse(`
		<!DOCTYPE html>
		<html>
		<head>
		</head>
		<body>
			<form method="POST" action="/admin/login">
				<label>Username: <input type="text" name="username" required></label><br>
				<label>Password: <input type="password" name="password" required></label><br>
				<button type="submit">Login</button>
			</form>
		</body>
		</html>
		`))

		w.Header().Set("Content-Type", "text/html")
		tmpl.Execute(w, nil)
	})

	mux.HandleFunc("POST /admin/login", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		username := r.FormValue("username")
		password := r.FormValue("password")

		isAdmin, err := db.IsAdmin(username, password)
		if err != nil {
			log.Printf("Admin auth error: %v", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}

		if !isAdmin {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("wrong login or password"))
			return
		}

		createSession(username, w)
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
	})

	mux.HandleFunc("GET /admin", authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		username, _ := SessionUsername(r)
		users, err := db.Userlist()
		if err != nil {
			log.Printf("Get userlist error: %v", err)
			return
		}
		if username != "Admin" {
			http.Error(w, "You are not admin", http.StatusForbidden)
			return
		}

		tmpl := template.Must(template.New("adminPanel").Parse(`
		<!DOCTYPE html>
		<html>
		<head>
			<title>Admin Panel</title>
			<style>
				body { font-family: Arial, sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; }
				.user-list { margin: 20px 0; border: 1px solid #ddd; padding: 15px; }
				.user-item { padding: 8px 0; border-bottom: 1px solid #eee; }
				.block-btn { color: red; margin-left: 10px; }
			</style>
		</head>
		<body>
			<h1>Admin Panel</h1>
			
			<div class="user-list">
				<h2>User List</h2>
				{{range .Users}}
				<div class="user-item">
					{{.}}
					<form method="POST" action="/admin/block" style="display: inline;">
						<input type="hidden" name="username" value="{{.}}">
						<button type="submit" class="block-btn">Block</button>
					</form>
				</div>
				{{end}}
			</div>
			<form method="POST" action="/admin/shutdown">
				<button type="submit" class="shutdown-btn">Shutdown Server</button>
			</form>
		</body>
		</html>
		`))

		w.Header().Set("Content-Type", "text/html")
		tmpl.Execute(w, struct {
			Users []string
		}{
			Users: users,
		})
	}))

	mux.HandleFunc("POST /admin/block", authMiddleware(func(w http.ResponseWriter, r *http.Request) {

		username := r.FormValue("username")

		if err := db.Block(username); err != nil {
			http.Error(w, "block fail", http.StatusInternalServerError)
			log.Printf("block user error: %v", err)
			return
		}

		http.Redirect(w, r, "/admin", http.StatusSeeOther)
	}))

	mux.HandleFunc("POST /admin/shutdown", authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		admin, _ := SessionUsername(r)
		if admin != "Admin" {
			http.Error(w, "You are not admin", http.StatusForbidden)
			return
		}

		log.Printf("server shutdown")

		w.Header().Set("Content-Type", "text/html")

		close(shutdown)
	}))

	go func() {
		log.Printf("------------------------------------------------\nServer starting on :8081")
		err := server.ListenAndServe()
		if err != nil {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-shutdown

	log.Println("server is shut down")
}
