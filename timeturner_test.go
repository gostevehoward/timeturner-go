package timeturner

import (
	"testing"
	"time"
)

type FakeDatabase struct {
	findSnapshotOk bool
}

func (db FakeDatabase) AddSnapshot(timestamp time.Time, hostname string, title string,
	contents [][]string) {
}
func (db FakeDatabase) GetAllDays() []time.Time                 { return nil }
func (db FakeDatabase) GetTimestamps(day time.Time) []time.Time { return nil }
func (db FakeDatabase) GetSnapshots(timestamp time.Time) []Snapshot {
	return []Snapshot{
		{Hostname: "host1", Title: "processes"},
		{Hostname: "host1", Title: "queries"},
		{Hostname: "host2", Title: "processes"},
	}
}
func (db FakeDatabase) GetSnapshotWithContents(timestamp time.Time, hostname string, title string) (
	snapshot Snapshot, ok bool) {
	if db.findSnapshotOk {
		return Snapshot{
			UnixTimestamp: 123,
			Hostname:      "host1",
			Title:         "processes",
			CsvContents:   "name,value\nkey2,2\nkey1,1\n",
		}, true
	} else {
		return Snapshot{}, false
	}
}

func setUpPresenter() (*FakeDatabase, Presenter) {
	requestInfo := RequestInfo{
		Timestamp: time.Date(2013, 10, 6, 0, 0, 0, 0, time.Local),
		Form:      make(map[string]string),
	}
	db := &FakeDatabase{}
	return db, Presenter{db, requestInfo}
}

func areStringsEqual(slice1 []string, slice2 []string) bool {
	if len(slice1) != len(slice2) {
		return false
	}
	for index, value1 := range slice1 {
		if value1 != slice2[index] {
			return false
		}
	}
	return true
}

func TestListHostsAndTitles(t *testing.T) {
	_, presenter := setUpPresenter()
	_, seenHostMap := presenter.ListHostsAndTitles()
	ok := len(seenHostMap) == 2 &&
		areStringsEqual(seenHostMap["host1"], []string{"processes", "queries"}) &&
		areStringsEqual(seenHostMap["host2"], []string{"processes"})
	if !ok {
		t.Fatalf("Unexpected host map %v", seenHostMap)
	}
}

func TestViewSnapshot(t *testing.T) {
	fakeDb, presenter := setUpPresenter()
	fakeDb.findSnapshotOk = true
	_, columns, data, ok := presenter.ViewSnapshot()
	if !ok {
		t.Fatalf("Got !ok for snapshot that exists")
	}
	if !(columns[0].Name == "name" && columns[1].Name == "value") {
		t.Fatalf("Unexpected columns %v", columns)
	}
	isDataOk := areStringsEqual(data[0], []string{"key2", "2"}) &&
		areStringsEqual(data[1], []string{"key1", "1"})
	if !isDataOk {
		t.Fatalf("Unexpected data %v", data)
	}
}

func TestViewSnapshotWithSorting(t *testing.T) {
	fakeDb, presenter := setUpPresenter()
	fakeDb.findSnapshotOk = true
	presenter.RequestInfo.Form["sort"] = "name"
	_, _, data, _ := presenter.ViewSnapshot()
	isDataOk := areStringsEqual(data[0], []string{"key1", "1"}) &&
		areStringsEqual(data[1], []string{"key2", "2"})
	if !isDataOk {
		t.Fatalf("Unexpected data %v", data)
	}
}

func TestViewSnapshotNotFound(t *testing.T) {
	fakeDb, presenter := setUpPresenter()
	fakeDb.findSnapshotOk = false
	_, _, _, ok := presenter.ViewSnapshot()
	if ok {
		t.Fatalf("Got ok for snapshot that doesn't exist")
	}
}
