package main

import (
	"context"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"crypto/tls"

	chat "github.com/ArtemKVD/HttpChatGo/internal/chat"
	metrics "github.com/ArtemKVD/HttpChatGo/internal/metrics"
	news "github.com/ArtemKVD/HttpChatGo/internal/news"
	sertificate "github.com/ArtemKVD/HttpChatGo/internal/sertificate"
	session "github.com/ArtemKVD/HttpChatGo/internal/session"
	templates "github.com/ArtemKVD/HttpChatGo/internal/templates"
	db "github.com/ArtemKVD/HttpChatGo/pkg/DB"
	cache "github.com/ArtemKVD/HttpChatGo/pkg/redis"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var mutex = sync.Mutex{}
var redisClient *cache.RedisClient

type Message struct {
	Sender       string `json:"sender"`
	SenderFriend string `json:"senderfriend"`
	Text         string `json:"text"`
}

const connectionDB = "host=postgres user=postgres dbname=Users password=admin sslmode=disable"

func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		ctx := context.Background()
		username, err := redisClient.Get(ctx, "session:"+cookie.Value)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		ctx = context.WithValue(r.Context(), "username", username)
		next(w, r.WithContext(ctx))
	}
}
func main() {
	http.Handle("/metrics", promhttp.Handler())

	go func() {
		err := http.ListenAndServe(":2112", nil)
		if err != nil {
			log.Fatalf("Failed to start metrics server: %v", err)
		}
	}()

	db.WaitPostgres()
	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		redisHost = "redis"
	}
	redisClient = cache.NewRedisClient(redisHost+":6379", "")
	if redisClient == nil {
		log.Fatal("Failed to initialize Redis client")
	}
	cache.WaitRedis(redisClient)

	if err := redisClient.Ping(context.Background()); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	mux := http.NewServeMux()

	shutdown := make(chan struct{})

	templates, err := templates.LoadTemplates()

	for _, t := range templates.Templates() {
		log.Println("Loaded template:", t.Name())
	}

	if err != nil {
		log.Printf("Error load templates %v", err)
	}
	mux.HandleFunc("GET /ws", func(w http.ResponseWriter, r *http.Request) {
		session.HandleConnections(redisClient, w, r)
	})
	mux.HandleFunc("GET /register", func(w http.ResponseWriter, r *http.Request) {
		err := templates.ExecuteTemplate(w, "auth/register.html", nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Printf("template error %v", err)
		}
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
			http.Error(w, "password1 != password2", http.StatusBadRequest)
			return
		}

		hashedpass, err := db.HashPassword(pass)
		if err != nil {
			http.Error(w, "password hash error", http.StatusInternalServerError)
			return
		}

		err = db.UsernameInsert(name, hashedpass)
		if err != nil {
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}

		err = session.CreateSession(redisClient, name, w)
		if err != nil {
			log.Println("CreateSession error")
		}

		http.Redirect(w, r, "/friends", http.StatusSeeOther)
	})

	mux.HandleFunc("GET /login", func(w http.ResponseWriter, r *http.Request) {
		err := templates.ExecuteTemplate(w, "auth/login.html", nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

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
			log.Printf(pass)
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

		session.CreateSession(redisClient, name, w)
		metrics.ActiveUsers.Inc()
		log.Printf("User %v login at %v", name, time.Now())

		http.Redirect(w, r, "/friends", http.StatusSeeOther)

		w.Header().Set("Content-Type", "text/html")
		tmpl.Execute(w, struct {
			Name string
		}{
			Name: name,
		})
	})
	mux.HandleFunc("POST /logout", func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err == nil {
			ctx := context.Background()
			metrics.ActiveUsers.Dec()
			redisClient.Del(ctx, "session:"+cookie.Value)
		}

		http.SetCookie(w, &http.Cookie{
			Name:   "session",
			Value:  "",
			MaxAge: -1,
		})
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	})
	mux.HandleFunc("GET /friends", func(w http.ResponseWriter, r *http.Request) {
		username, ok := session.SessionUsername(redisClient, r)
		if !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		friends, err := db.GetFriends(username)
		if err != nil {
			http.Error(w, "Failed to get friends list", http.StatusInternalServerError)
			log.Printf("get friend list user %v error: %v", username, err)
			return
		}

		err = templates.ExecuteTemplate(w, "chat/friends.html", struct {
			Username string
			Friends  []string
		}{
			Username: username,
			Friends:  friends,
		})

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("POST /add_friend", AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		username, _ := session.SessionUsername(redisClient, r)
		friendname := r.FormValue("friendname")

		if err := db.AddFriend(username, friendname); err != nil {
			http.Error(w, "Failed to add friend", http.StatusInternalServerError)
			log.Printf("Add friend fail user:%v friend:%v error: %v", username, friendname, err)
			return
		}

		http.Redirect(w, r, "/friends", http.StatusSeeOther)
	}))

	mux.HandleFunc("GET /chat/{friend}", AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		user, _ := session.SessionUsername(redisClient, r)
		friend := r.PathValue("friend")
		messagelist, err := chat.Messagelist(user, friend)
		if err != nil {
			http.Error(w, "message fail", http.StatusInternalServerError)
			log.Printf("message list delivery fail by %v to %v: %v", user, friend, err)
		}
		log.Print("Messages loaded", len(messagelist))

		err = templates.ExecuteTemplate(w, "chat/chat.html", struct {
			CurrentUser string
			Friend      string
			Messages    []chat.Message
		}{
			CurrentUser: user,
			Friend:      friend,
			Messages:    messagelist,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

	}))

	mux.HandleFunc("POST /send_message", AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			log.Println("Form parse error")
		}
		user, ok := session.SessionUsername(redisClient, r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		userfriend := r.FormValue("userfriend")
		message := r.FormValue("message")
		//log.Println("send message: user, friend and message loaded")
		err = redisClient.CacheMessage(r.Context(), "room1", []byte(message), 24*time.Hour)
		if err != nil {
			log.Printf("redis error")
		}
		log.Printf("Saving message from %s %s %s", user, userfriend, message)
		if err := chat.Send(user, userfriend, message); err != nil {
			http.Error(w, "Failed to send message", http.StatusInternalServerError)
			log.Printf("Send message error: %v from user: %v to user: %v", err, user, userfriend)
			return
		}
		metrics.MessagesSend.WithLabelValues(user, userfriend).Inc()
		session.Broadcast <- session.Message{
			Sender:       user,
			SenderFriend: userfriend,
			Text:         message,
		}

		http.Redirect(w, r, "/chat/"+userfriend, http.StatusSeeOther)
	}))

	mux.HandleFunc("GET /news", AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		currentUser, _ := session.SessionUsername(redisClient, r)

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

		err = templates.ExecuteTemplate(w, "newsh/news.html", struct {
			Posts []news.Post
		}{
			Posts: posts,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

	}))

	mux.HandleFunc("POST /news", AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		currentUser, _ := session.SessionUsername(redisClient, r)

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
		err := templates.ExecuteTemplate(w, "admin/adminlogin.html", nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
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

		session.CreateSession(redisClient, username, w)
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
	})

	mux.HandleFunc("GET /admin", AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		username, _ := session.SessionUsername(redisClient, r)
		users, err := db.Userlist()
		if err != nil {
			log.Printf("Get userlist error: %v", err)
			return
		}
		if username != "admin" {
			http.Error(w, "You are not admin", http.StatusForbidden)
			return
		}

		err = templates.ExecuteTemplate(w, "admin/panel.html", struct {
			Users []string
		}{
			Users: users,
		})

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}))

	mux.HandleFunc("POST /admin/block", AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {

		username := r.FormValue("username")

		if err := db.Block(username); err != nil {
			http.Error(w, "block fail", http.StatusInternalServerError)
			log.Printf("block user error: %v", err)
			return
		}

		http.Redirect(w, r, "/admin", http.StatusSeeOther)
	}))

	mux.HandleFunc("POST /admin/shutdown", AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		admin, _ := session.SessionUsername(redisClient, r)
		if admin != "admin" {
			http.Error(w, "You are not admin", http.StatusForbidden)
			return
		}

		log.Printf("server shutdown")

		w.Header().Set("Content-Type", "text/html")

		close(shutdown)
	}))
	cert, err := sertificate.GenerateSelfSignedCert()
	if err != nil {
		log.Fatalf("Failed to generate certificate: %v", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	server := &http.Server{
		Addr:      ":8444",
		Handler:   mux,
		TLSConfig: tlsConfig,
	}

	go func() {
		for msg := range session.Broadcast {
			mutex.Lock()
			for client := range session.Clients {
				if client.User == msg.SenderFriend || client.User == msg.Sender {
					response := map[string]string{
						"sender": msg.Sender,
						"text":   msg.Text,
					}
					jsonMsg, _ := json.Marshal(response)
					client.Send <- jsonMsg
				}
			}
			mutex.Unlock()
		}
	}()

	go func() {
		http.ListenAndServe(":8080", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "https://"+r.Host+r.URL.String(), http.StatusMovedPermanently)
		}))
	}()

	log.Println("HTTPS server running on :8444")
	server.ListenAndServeTLS("", "")

	<-shutdown

	log.Println("server is shut down")
}
