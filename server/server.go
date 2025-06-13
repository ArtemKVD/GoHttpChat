package main

import (
	"encoding/json"
	"html/template"
	"log"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"

	chat "github.com/ArtemKVD/HttpChatGo/chat"
	news "github.com/ArtemKVD/HttpChatGo/news"
	db "github.com/ArtemKVD/HttpChatGo/pkg/DB"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	_ "github.com/lib/pq"
)

var name string
var pass string
var sessions = make(map[string]string)

var clients = make(map[*Client]bool)
var broadcast = make(chan Message)
var mutex = &sync.Mutex{}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Client struct {
	conn *websocket.Conn
	send chan []byte
	user string
}

type Message struct {
	Sender       string `json:"sender"`
	SenderFriend string `json:"senderfriend"`
	Text         string `json:"text"`
}

const connectionDB = "user=postgres dbname=Users password=admin sslmode=disable"

func (c *Client) ReadM() {
	defer func() {
		mutex.Lock()
		delete(clients, c)
		mutex.Unlock()
		c.conn.Close()
	}()

	for {
		var msg Message
		err := c.conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("Read error: %v", err)
			break
		}
		msg.Sender = c.user
		broadcast <- msg
	}
}

func (c *Client) WriteM() {
	defer c.conn.Close()
	for {
		message, ok := <-c.send
		if !ok {
			return
		}
		err := c.conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			log.Printf("Write error: %v", err)
			return
		}
	}
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	user, ok := SessionUsername(r)
	if !ok {
		return
	}

	client := &Client{
		conn: conn,
		send: make(chan []byte, 256),
		user: user,
	}

	mutex.Lock()
	clients[client] = true
	mutex.Unlock()

	go client.WriteM()
	client.ReadM()
}

func loadTemplates() (*template.Template, error) {
	tmpl := template.New("").Funcs(template.FuncMap{})

	err := filepath.Walk("views", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(path, ".html") {
			bytes, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			name := strings.TrimPrefix(
				strings.ReplaceAll(path, "\\", "/"),
				"views/",
			)

			if _, err := tmpl.New(name).Parse(string(bytes)); err != nil {
				log.Fatalf("failed to parse %s error:%v", name, err)
			}
		}
		return nil
	})

	return tmpl, err
}

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

func GenerateSelfSignedCert() (tls.Certificate, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Localhost"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(365 * 24 * time.Hour),
		DNSNames:  []string{"localhost"},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)

	if err != nil {
		return tls.Certificate{}, err
	}

	return tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  priv,
	}, nil
}

func main() {
	mux := http.NewServeMux()

	shutdown := make(chan struct{})

	templates, err := loadTemplates()

	for _, t := range templates.Templates() {
		log.Println("Loaded template:", t.Name())
	}

	if err != nil {
		log.Printf("Error load templates %v", err)
	}
	mux.HandleFunc("GET /ws", handleConnections)
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

		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
        <!DOCTYPE html>
        <html>
        <body>
            <h1>Registration successful</h1>
            <p>Welcome, ` + name + `!</p>
            <a href="/login">Login</a>
        </body>
        </html>
    `))
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
		HashPass, err := db.GetUserPasswordHash(name)

		if err != nil {
			log.Printf("error check hash password")
		}
		Check, err := db.CheckLogPass(name, HashPass)
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

	mux.HandleFunc("POST /send_message", authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		user, _ := SessionUsername(r)
		userfriend := r.FormValue("userfriend")
		message := r.FormValue("message")

		if err := chat.Send(user, userfriend, message); err != nil {
			http.Error(w, "Failed to send message", http.StatusInternalServerError)
			log.Printf("Send message error: %v from user: %v to user: %v", err, user, userfriend)
			return
		}
		broadcast <- Message{
			Sender:       user,
			SenderFriend: userfriend,
			Text:         message,
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

		err = templates.ExecuteTemplate(w, "newsh/news.html", struct {
			Posts []news.Post
		}{
			Posts: posts,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
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

		err = templates.ExecuteTemplate(w, "admin/panel.html", struct {
			Users []string
		}{
			Users: users,
		})

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
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
	cert, err := GenerateSelfSignedCert()
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
		for msg := range broadcast {
			mutex.Lock()
			for client := range clients {
				if client.user == msg.SenderFriend || client.user == msg.Sender {
					response := map[string]string{
						"sender": msg.Sender,
						"text":   msg.Text,
					}
					jsonMsg, _ := json.Marshal(response)
					client.send <- jsonMsg
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
