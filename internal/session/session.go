package session

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	cache "github.com/ArtemKVD/HttpChatGo/pkg/redis"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

func HandleConnections(redisClient *cache.RedisClient, w http.ResponseWriter, r *http.Request) {
	Conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer Conn.Close()

	user, ok := SessionUsername(redisClient, r)
	if !ok {
		return
	}

	client := &Client{
		Conn: Conn,
		Send: make(chan []byte, 256),
		User: user,
	}

	mutex.Lock()
	Clients[client] = true
	mutex.Unlock()

	go client.WriteM()
	client.ReadM()
	ctx := context.Background()
	pubsub := redisClient.Subscribe(ctx, "chat_updates")
	defer pubsub.Close()

	go func() {
		for msg := range pubsub.Channel() {
			Conn.WriteMessage(websocket.TextMessage, []byte(msg.Payload))
		}
	}()
}

type Message struct {
	Sender       string `json:"sender"`
	SenderFriend string `json:"senderfriend"`
	Text         string `json:"text"`
}

func (c *Client) ReadM() {
	defer func() {
		mutex.Lock()
		delete(Clients, c)
		mutex.Unlock()
		c.Conn.Close()
	}()

	for {
		var msg Message
		err := c.Conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("Read error: %v", err)
			break
		}
		msg.Sender = c.User
		Broadcast <- msg
	}
}

func (c *Client) WriteM() {
	defer c.Conn.Close()
	for {
		message, ok := <-c.Send
		if !ok {
			return
		}
		err := c.Conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			log.Printf("Write error: %v", err)
			return
		}
	}
}

func CreateSession(redisClient *cache.RedisClient, username string, w http.ResponseWriter) error {
	sessionID := uuid.New().String()
	ctx := context.Background()

	err := redisClient.Set(ctx, "session:"+sessionID, username, 24*time.Hour)
	if err != nil {
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   86400,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	return nil
}

func SessionUsername(redisClient *cache.RedisClient, r *http.Request) (string, bool) {
	cookie, err := r.Cookie("session")
	if err != nil {
		return "", false
	}

	ctx := context.Background()
	username, err := redisClient.Get(ctx, "session:"+cookie.Value)
	if err != nil {
		return "", false
	}

	return username, true
}

func DestroySession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   10000,
	})
}

type Client struct {
	Conn *websocket.Conn
	Send chan []byte
	User string
}

var Clients = make(map[*Client]bool)

var Broadcast = make(chan Message)

var mutex = &sync.Mutex{}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}
