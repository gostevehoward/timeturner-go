package timeturner

import (
	"bytes"
	"encoding/csv"
	"github.com/gorilla/mux"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"sort"
	"time"
)

const DATE_FORMAT = "2006-01-02"
const TIME_FORMAT = "15:04:05"
const DATETIME_FORMAT = "2006-01-02 15:04:05 MST"

func parseCsv(csvContents string) [][]string {
	reader := csv.NewReader(bytes.NewBufferString(csvContents))
	contents, err := reader.ReadAll()
	if err != nil {
		panic(err)
	} else {
		return contents
	}
}

func dumpCsv(contents [][]string) string {
	var csvContentsBuffer bytes.Buffer
	err := csv.NewWriter(&csvContentsBuffer).WriteAll(contents)
	if err != nil {
		panic(err)
	}
	return csvContentsBuffer.String()
}

type RequestInfo struct {
	Vars      map[string]string
	Timestamp time.Time
	Form      map[string]string
	Body      string
}

type Database interface {
	AddSnapshot(timestamp time.Time, hostname string, title string, contents [][]string)
	GetAllDays() []time.Time
	GetTimestamps(day time.Time) []time.Time
	GetSnapshots(timestamp time.Time) []Snapshot
	GetSnapshotWithContents(timestamp time.Time, hostname string, title string) (
		snapshot Snapshot, ok bool)
}

type App struct {
	Database  Database
	Router    *mux.Router
	Templates *template.Template
}

type Presenter struct {
	Database    Database
	RequestInfo RequestInfo
}

func (app App) WrapHandler(handler func(View)) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		log.Printf("Handling %v\n", request.URL)
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

		var body string
		if request.Method == "PUT" {
			contents := make([]byte, 1024*1024)
			bytesRead, err := request.Body.Read(contents)
			if err != nil {
				http.Error(
					writer,
					"Failed to read request body: "+err.Error(),
					http.StatusBadRequest,
				)
				return
			}
			body = string(contents[:bytesRead])
		}

		request.ParseForm()
		formValues := make(map[string]string)
		for key, values := range request.Form {
			formValues[key] = values[0]
		}

		presenter := Presenter{app.Database, RequestInfo{vars, timestamp, formValues, body}}
		view := View{app.Router, app.Templates, writer, presenter}

		handler(view)
	}
}

func (presenter Presenter) ListDays() []time.Time {
	return presenter.Database.GetAllDays()
}

func (presenter Presenter) ListTimes() (day time.Time, times []time.Time) {
	day = presenter.RequestInfo.Timestamp
	times = presenter.Database.GetTimestamps(presenter.RequestInfo.Timestamp)
	return
}

func (presenter Presenter) ListHostsAndTitles() (timestamp time.Time, hostMap map[string][]string) {
	snapshots := presenter.Database.GetSnapshots(presenter.RequestInfo.Timestamp)

	hostMap = make(map[string][]string)
	for _, snapshot := range snapshots {
		if _, ok := hostMap[snapshot.Hostname]; !ok {
			hostMap[snapshot.Hostname] = make([]string, 0)
		}
		hostMap[snapshot.Hostname] = append(hostMap[snapshot.Hostname], snapshot.Title)
	}

	return presenter.RequestInfo.Timestamp, hostMap
}

func (presenter Presenter) AddSnapshot() {
	presenter.Database.AddSnapshot(
		presenter.RequestInfo.Timestamp,
		presenter.RequestInfo.Vars["hostname"],
		presenter.RequestInfo.Vars["title"],
		parseCsv(presenter.RequestInfo.Body),
	)
}

type Column struct {
	Name         string
	IsSortColumn bool
	ReverseLink  bool
}

type SortableRows struct {
	data            [][]string
	sortColumnIndex int
	isReversed      bool
}

func (rows SortableRows) Len() int      { return len(rows.data) }
func (rows SortableRows) Swap(i, j int) { rows.data[i], rows.data[j] = rows.data[j], rows.data[i] }
func (rows SortableRows) Less(i, j int) bool {
	isLess := rows.data[i][rows.sortColumnIndex] < rows.data[j][rows.sortColumnIndex]
	if rows.isReversed {
		return !isLess
	} else {
		return isLess
	}
}

func findColumnIndex(columns []string, desiredColumn string) int {
	for index, columnName := range columns {
		if columnName == desiredColumn {
			return index
		}
	}
	return -1
}

func (presenter Presenter) ViewSnapshot() (
	snapshot Snapshot, columns []Column, data [][]string, ok bool) {
	snapshot, ok = presenter.Database.GetSnapshotWithContents(
		presenter.RequestInfo.Timestamp,
		presenter.RequestInfo.Vars["hostname"],
		presenter.RequestInfo.Vars["title"],
	)
	if !ok {
		return
	}

	contents := snapshot.Contents()
	columnNames := contents[0]
	data = contents[1:]

	sortColumn := presenter.RequestInfo.Form["sort"]
	_, isReversed := presenter.RequestInfo.Form["reverse"]
	var sortColumnIndex int
	for index, columnName := range columnNames {
		column := Column{columnName, false, false}
		if columnName == sortColumn {
			column.IsSortColumn = true
			column.ReverseLink = !isReversed
			sortColumnIndex = index
		}
		columns = append(columns, column)
	}

	if sortColumn != "" {
		sort.Sort(SortableRows{data, sortColumnIndex, isReversed})
	}

	ok = true
	return
}

func LoadTemplates() *template.Template {
	templateGlob := filepath.Join(".", "templates", "*.gohtml")
	return template.Must(
		template.New("root").Funcs(templateFunctions).ParseGlob(templateGlob),
	)
}

func MakeApp(database Database) App {
	router := mux.NewRouter()
	app := App{Database: database, Router: router, Templates: LoadTemplates()}

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
