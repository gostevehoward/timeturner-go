package timeturner

import (
	"github.com/gorilla/mux"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"time"
)

type App struct {
	Database  Database
	Router    *mux.Router
	Templates *template.Template
}

func parseTimestamp(urlVars map[string]string) (timestamp time.Time, err error) {
	date, hasDate := urlVars["date"]
	time_, hasTime := urlVars["time"]
	if hasDate {
		if hasTime {
			timestamp, err = time.ParseInLocation(
				DATE_FORMAT+" "+TIME_FORMAT, date+" "+time_, time.Local,
			)
		} else {
			timestamp, err = time.ParseInLocation(DATE_FORMAT, date, time.Local)
		}
	}
	return
}

func readRequestBody(request *http.Request) (string, error) {
	if request.Method == "PUT" {
		contents := make([]byte, 1024*1024)
		bytesRead, err := request.Body.Read(contents)
		if err != nil {
			return "", err
		}
		return string(contents[:bytesRead]), nil
	} else {
		return "", nil
	}
}

func readFormValues(request *http.Request) map[string]string {
	request.ParseForm()
	formValues := make(map[string]string)
	for key, values := range request.Form {
		formValues[key] = values[0]
	}
	return formValues
}

func (app App) WrapHandler(handler func(View)) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		log.Printf("Handling %v\n", request.URL)
		vars := mux.Vars(request)

		timestamp, err := parseTimestamp(vars)
		if err != nil {
			http.Error(writer, "Failed to parse timestamp: "+err.Error(), http.StatusBadRequest)
			return
		}

		body, err := readRequestBody(request)
		if err != nil {
			http.Error(writer, "Failed to read request body: "+err.Error(), http.StatusBadRequest)
			return
		}

		formValues := readFormValues(request)
		presenter := Presenter{app.Database, RequestInfo{vars, timestamp, formValues, body}}
		view := View{app.Router, app.Templates, writer, presenter}

		handler(view)
	}
}

func LoadTemplates(router *mux.Router) *template.Template {
	templateGlob := filepath.Join(".", "templates", "*.gohtml")
	return template.Must(
		template.New("root").Funcs(makeTemplateFunctions(router)).ParseGlob(templateGlob),
	)
}

func MakeApp(database Database) App {
	router := mux.NewRouter()
	app := App{Database: database, Router: router, Templates: LoadTemplates(router)}

	router.HandleFunc("/", app.WrapHandler(func(v View) { v.ListDays() })).
		Name("list days").
		Methods("GET")
	router.HandleFunc("/{date}/", app.WrapHandler(func(v View) { v.ListTimes() })).
		Name("list times on day").
		Methods("GET")
	router.HandleFunc("/{date}/{time}/", app.WrapHandler(func(v View) { v.ListSnapshots() })).
		Name("list snapshots at time").
		Methods("GET")

	snapshotRouter := router.PathPrefix("/{date}/{time}/{hostname}/{title}/").Subrouter()
	snapshotRouter.HandleFunc("/", app.WrapHandler(func(v View) { v.ViewSnapshot() })).
		Name("view snapshot").
		Methods("GET")
	snapshotRouter.HandleFunc("/", app.WrapHandler(func(v View) { v.AddSnapshot() })).
		Methods("PUT")

	return app
}
