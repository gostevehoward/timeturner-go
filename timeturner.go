package timeturner

import (
	"github.com/gorilla/mux"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"time"
)

const DATE_FORMAT = "2006-02-01"
const TIME_FORMAT = "03:04:05"

type Snapshot struct {
	Timestamp time.Time
	Hostname  string
	Title     string
}

func (snapshot *Snapshot) GetUrl(router *mux.Router) string {
	urlParameters := []string{
		"date", snapshot.Timestamp.Format(DATE_FORMAT),
		"time", snapshot.Timestamp.Format(TIME_FORMAT),
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
	Router    *mux.Router
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
		date := vars["date"]

		var timestamp time.Time
		var err error
		if time_, hasTime := vars["time"]; hasTime {
			timestamp, err = time.Parse(DATE_FORMAT+" "+TIME_FORMAT, date+" "+time_)
		} else {
			timestamp, err = time.Parse(DATE_FORMAT, date)
		}
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
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
		ListDaysContext{
			context.App.Router,
			[]time.Time{
				time.Date(2013, 10, 4, 0, 0, 0, 0, time.Local),
				time.Date(2013, 10, 5, 0, 0, 0, 0, time.Local),
			},
		},
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
			[]time.Time{
				time.Date(2013, 10, 4, 1, 2, 3, 0, time.Local),
				time.Date(2013, 10, 4, 4, 5, 6, 0, time.Local),
			},
		},
	)
}

type ListSnapshotsContext struct {
	*mux.Router
	Timestamp time.Time
	HostMap   map[string][]Snapshot
}

func ListSnapshots(context *BaseContext) {
	// testing only!
	ms := func(hostname string, title string) Snapshot {
		return Snapshot{context.Timestamp, hostname, title}
	}

	context.renderTemplate(
		"list snapshots",
		ListSnapshotsContext{
			context.App.Router,
			context.Timestamp,
			map[string][]Snapshot{
				"host1": []Snapshot{ms("host1", "Processes"), ms("host1", "Queries")},
				"host2": []Snapshot{ms("host2", "Processes"), ms("host2", "Queries")},
			},
		},
	)
}

type ViewSnapshotContext struct {
	Snapshot Snapshot
	Contents string
}

func HandleSnapshot(context *BaseContext) {
	vars := mux.Vars(context.Request)
	context.renderTemplate(
		"view snapshot",
		ViewSnapshotContext{
			Snapshot{context.Timestamp, vars["hostname"], vars["title"]},
			"hello world!",
		},
	)
}

func MakeApp() *TimeturnerApp {
	app := TimeturnerApp{}
	app.Router = mux.NewRouter()
	app.Router.HandleFunc("/", app.WrapHandler(ListDays)).Name("list days")
	app.Router.HandleFunc("/{date}/", app.WrapHandler(ListTimes)).Name("list times on day")
	app.Router.HandleFunc("/{date}/{time}", app.WrapHandler(ListSnapshots)).
		Name("list snapshots at time")
	app.Router.HandleFunc("/{date}/{time}/{hostname}/{title}", app.WrapHandler(HandleSnapshot)).
		Name("snapshot")

	templateGlob := filepath.Join(".", "templates", "*.gohtml")
	app.Templates = template.Must(
		template.New("root").Funcs(templateFunctions).ParseGlob(templateGlob),
	)

	return &app
}
