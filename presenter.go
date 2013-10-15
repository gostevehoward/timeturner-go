package timeturner

import (
	"sort"
	"time"
)

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

type Presenter struct {
	Database    Database
	RequestInfo RequestInfo
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
