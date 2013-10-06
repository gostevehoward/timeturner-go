package main

import (
	"github.com/gostevehoward/timeturner"
	"log"
	"net/http"
)

func main() {
	app := timeturner.MakeApp()
	http.Handle("/", app.Router)
	log.Print("Running on localhost:8080")
	http.ListenAndServe(":8080", nil)
}
