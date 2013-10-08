package timeturner

import (
	"fmt"
	"github.com/gorilla/mux"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"time"
)

const DATE_FORMAT = "2006-01-02"
const TIME_FORMAT = "15:04:05"

func (snapshot *Snapshot) GetUrl(router *mux.Router) string {
	urlParameters := []string{
		"date", snapshot.Timestamp().Format(DATE_FORMAT),
		"time", snapshot.Timestamp().Format(TIME_FORMAT),
		"hostname", snapshot.Hostname,
		"title", snapshot.Title,
	}
	url, err := router.Get("snapshot").URL(urlParameters...)
	if err != nil {
		panic(err)
	}
	return url.String()
}

type TimeturnerApp struct {
	*Database
	*mux.Router
	Templates *template.Template
}

type BaseContext struct {
	Writer    http.ResponseWriter
	Request   *http.Request
	App       *TimeturnerApp
	Timestamp time.Time
}

func (app *TimeturnerApp) WrapHandler(handler func(*BaseContext)) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		log.Printf("Handling %q\n", request.URL)
		vars := mux.Vars(request)
		date, hasDate := vars["date"]
		time_, hasTime := vars["time"]

		var timestamp time.Time
		var err error
		if hasDate {
			if hasTime {
				timestamp, err = time.ParseInLocation(
					DATE_FORMAT+" "+TIME_FORMAT, date+" "+time_, time.Local,
				)
			} else {
				timestamp, err = time.ParseInLocation(DATE_FORMAT, date, time.Local)
			}
		}
		if err != nil {
			http.Error(writer, "Failed to parse timestamp: "+err.Error(), http.StatusBadRequest)
			return
		}

		handler(&BaseContext{writer, request, app, timestamp})
	}
}

var templateFunctions = template.FuncMap{
	"formatDate": func(date time.Time) string { return date.Format(DATE_FORMAT) },
	"formatTime": func(date time.Time) string { return date.Format(TIME_FORMAT) },
	"formatDateTime": func(date time.Time) string {
		return date.Format(DATE_FORMAT + " " + TIME_FORMAT)
	},
}

func (context *BaseContext) renderTemplate(templateName string, templateContext interface{}) {
	err := context.App.Templates.ExecuteTemplate(context.Writer, templateName, templateContext)
	if err != nil {
		log.Printf("ERROR: Failed to render template %q: %q\n", templateName, err)
	}
}

type ListDaysContext struct {
	*mux.Router
	Days []time.Time
}

func ListDays(context *BaseContext) {
	context.renderTemplate(
		"list days",
		ListDaysContext{context.App.Router, context.App.Database.GetAllDays()},
	)
}

type ListTimesContext struct {
	*mux.Router
	Date       time.Time
	Timestamps []time.Time
}

func ListTimes(context *BaseContext) {
	context.renderTemplate(
		"list times",
		ListTimesContext{
			context.App.Router,
			context.Timestamp,
			context.App.Database.GetTimestamps(context.Timestamp),
		},
	)
}

type ListSnapshotsContext struct {
	*mux.Router
	Timestamp time.Time
	HostMap   map[string][]*Snapshot
}

func ListSnapshots(context *BaseContext) {
	snapshots := context.App.Database.GetSnapshots(context.Timestamp)

	hostMap := make(map[string][]*Snapshot)
	for _, snapshot := range snapshots {
		if _, ok := hostMap[snapshot.Hostname]; !ok {
			hostMap[snapshot.Hostname] = make([]*Snapshot, 0)
		}
		hostMap[snapshot.Hostname] = append(hostMap[snapshot.Hostname], &snapshot)
	}

	context.renderTemplate(
		"list snapshots",
		ListSnapshotsContext{context.App.Router, context.Timestamp, hostMap},
	)
}

func addSnapshot(context *BaseContext) {
	vars := mux.Vars(context.Request)
	contents := make([]byte, 1024*1024)
	bytesRead, err := context.Request.Body.Read(contents)
	if err != nil {
		http.Error(
			context.Writer, "Failed to read request body: "+err.Error(), http.StatusBadRequest,
		)
	} else {
		context.App.Database.AddSnapshot(
			context.Timestamp, vars["hostname"], vars["title"], string(contents[:bytesRead]),
		)
	}
}

type ViewSnapshotContext struct {
	Snapshot *Snapshot
}

func viewSnapshot(context *BaseContext) {
	vars := mux.Vars(context.Request)
	snapshot, ok := context.App.Database.GetSnapshotWithContents(
		context.Timestamp, vars["hostname"], vars["title"],
	)
	if ok {
		context.renderTemplate("view snapshot", ViewSnapshotContext{&snapshot})
	} else {
		http.Error(context.Writer, fmt.Sprintf("No such snapshot found"), http.StatusNotFound)
	}
}

func HandleSnapshot(context *BaseContext) {
	if context.Request.Method == "PUT" {
		addSnapshot(context)
	} else {
		viewSnapshot(context)
	}
}

func MakeApp(database *Database) *TimeturnerApp {
	app := TimeturnerApp{Database: database, Router: mux.NewRouter()}

	app.Router.HandleFunc("/", app.WrapHandler(ListDays)).Name("list days")
	app.Router.HandleFunc("/{date}/", app.WrapHandler(ListTimes)).Name("list times on day")
	app.Router.HandleFunc("/{date}/{time}/", app.WrapHandler(ListSnapshots)).
		Name("list snapshots at time")
	app.Router.HandleFunc("/{date}/{time}/{hostname}/{title}/", app.WrapHandler(HandleSnapshot)).
		Name("snapshot")

	templateGlob := filepath.Join(".", "templates", "*.gohtml")
	app.Templates = template.Must(
		template.New("root").Funcs(templateFunctions).ParseGlob(templateGlob),
	)

	return &app
}
