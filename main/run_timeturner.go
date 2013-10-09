package main

import (
	"database/sql"
	"flag"
	"github.com/gostevehoward/timeturner"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"net/http"
	"time"
)

var enableSqlLogging = flag.Bool("sql-logging", false, "Log all SQL queries")

func main() {
	connection, err := sql.Open("sqlite3", "./timeturner.sqlite")
	if err != nil {
		panic(err)
	}
	defer connection.Close()

	database := timeturner.InitializeDatabase(connection, time.Now, *enableSqlLogging)
	app := timeturner.MakeApp(database)
	http.Handle("/", app.Router)
	log.Print("Running on localhost:8080")
	http.ListenAndServe(":8080", nil)
}
