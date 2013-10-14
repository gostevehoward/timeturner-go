package timeturner

import (
	"github.com/gorilla/mux"
	"html/template"
	"log"
	"net/http"
	"time"
)

type View struct {
	Router    *mux.Router
	Templates *template.Template
	Writer    http.ResponseWriter
	Presenter Presenter
}

type TemplateContext struct {
	Router *mux.Router
}

var templateFunctions = template.FuncMap{
	"formatDate":     func(date time.Time) string { return date.Format(DATE_FORMAT) },
	"formatTime":     func(date time.Time) string { return date.Format(TIME_FORMAT) },
	"formatDateTime": func(date time.Time) string { return date.Format(DATETIME_FORMAT) },
	"getSnapshotUrl": func(router *mux.Router, timestamp time.Time, hostname string, title string,
	) string {
		urlParameters := []string{
			"date", timestamp.Format(DATE_FORMAT),
			"time", timestamp.Format(TIME_FORMAT),
			"hostname", hostname,
			"title", title,
		}
		url, err := router.Get("view snapshot").URL(urlParameters...)
		if err != nil {
			panic(err)
		}
		return url.String()
	},
}

func (view View) renderTemplate(templateName string, templateContext interface{}) {
	err := view.Templates.ExecuteTemplate(view.Writer, templateName, templateContext)
	if err != nil {
		log.Printf("ERROR: Failed to render template %v: %v\n", templateName, err)
	}
}

func (view View) context() TemplateContext {
	return TemplateContext{view.Router}
}

type ListDaysContext struct {
	TemplateContext
	Days []time.Time
}

func (view View) ListDays() {
	days := view.Presenter.ListDays()
	view.renderTemplate("list days", ListDaysContext{view.context(), days})
}

type ListTimesContext struct {
	TemplateContext
	Date       time.Time
	Timestamps []time.Time
}

func (view View) ListTimes() {
	date, times := view.Presenter.ListTimes()
	view.renderTemplate("list times", ListTimesContext{view.context(), date, times})
}

type ListSnapshotsContext struct {
	TemplateContext
	Timestamp time.Time
	HostMap   map[string][]string
}

func (view View) ListSnapshots() {
	timestamp, hostMap := view.Presenter.ListHostsAndTitles()
	view.renderTemplate("list snapshots", ListSnapshotsContext{view.context(), timestamp, hostMap})
}

type ViewSnapshotContext struct {
	Snapshot Snapshot
	Columns  []Column
	Data     [][]string
}

func (view View) ViewSnapshot() {
	snapshot, columns, data, ok := view.Presenter.ViewSnapshot()
	if !ok {
		http.Error(view.Writer, "No such snapshot found", http.StatusNotFound)
	}
	view.renderTemplate("view snapshot", ViewSnapshotContext{snapshot, columns, data})
}

func (view View) AddSnapshot() {
	view.Presenter.AddSnapshot()
}
