package main

import (
	"log"
	"net/http"
	"text/template"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /login", func(w http.ResponseWriter, r *http.Request) {

		tmpl, err := template.New("login").Parse(`
			<!DOCTYPE html>
			<html>
			<head>
			</head>
			<body>
				<h1>sign up</h1>
				<form method="POST" action="/join">
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

	mux.HandleFunc("POST /", func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			http.Error(w, "request error", http.StatusBadRequest)
			return
		}

		name := r.FormValue("name")
		pass := r.FormValue("pass")
		pass2 := r.FormValue("pass2")

		acces := "Welcome"

		if pass != pass2 {
			acces = "password1 != password2;"
			name = "registration fail"
		}

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
			Pass  string
			Acces string
		}{
			Name:  name,
			Pass:  pass,
			Acces: acces,
		})
	})

	err := http.ListenAndServe(":8083", mux)
	if err != nil {
		log.Fatal("Ошибка сервера: ", err)
	}

}
