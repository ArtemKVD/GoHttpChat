package main

import (
	"log"
	"net/http"
	"text/template"

	db "github.com/ArtemKVD/HttpChatGo/pkg/DB"
	_ "github.com/lib/pq"
)

var name string
var pass string

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

		name := r.FormValue("nametologin")
		pass := r.FormValue("passtologin")

		db.CheckLogPass(name, pass)
	})

	mux.HandleFunc("POST /login", func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()

		if err != nil {
			http.Error(w, "request error", http.StatusBadRequest)
			return
		}

		name := r.FormValue("nametologin")

		acces := "Welcome"

		tmpl, err := template.New("join").Parse(`
			<!DOCTYPE html>
			<html>
			<head>
			</head>
			<body>
				<h1>{{.Acces}} {{.Name}}!</h1>
				<a href="/login">Quit</a>
			</body>
			</html>
		`)

		w.Header().Set("Content-Type", "text/html")
		tmpl.Execute(w, struct {
			Name  string
			Acces string
		}{
			Name:  name,
			Acces: acces,
		})
	})

	err := http.ListenAndServe(":8081", mux)
	if err != nil {
		log.Fatal("Ошибка сервера: ", err)
	}

}
