package timeturner

import (
	"database/sql"
	"fmt"
	"github.com/coopernurse/gorp"
	"log"
	"os"
	"time"
)

const SCHEMA = `
CREATE TABLE IF NOT EXISTS Snapshot (
    Id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    UnixTimestamp INTEGER,
    Hostname VARCHAR(255),
    Title VARCHAR(255),
    Contents TEXT
);
`

type Snapshot struct {
	Id            int64
	UnixTimestamp int64
	Hostname      string
	Title         string
	Contents      string
}

func (snapshot *Snapshot) Timestamp() time.Time {
	return time.Unix(snapshot.UnixTimestamp, 0)
}

type Database struct {
	mapper  *gorp.DbMap
	nowFunc func() time.Time
}

func InitializeDatabase(connection *sql.DB, nowFunc func() time.Time) *Database {
	mapper := &gorp.DbMap{Db: connection, Dialect: gorp.SqliteDialect{}}
	if false {
		mapper.TraceOn("[gorp]", log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)) // TODO
	}
	mapper.AddTable(Snapshot{}).SetKeys(true, "Id")

	_, err := mapper.Exec(SCHEMA)
	if err != nil {
		panic(err)
	}

	return &Database{mapper, nowFunc}
}

func (database *Database) cleanOldSnapshots() {
	oldestAllowedTimestamp := database.nowFunc().AddDate(0, 0, -14)
	query := "DELETE FROM Snapshot WHERE UnixTimestamp < ?"
	database.mapper.Exec(query, oldestAllowedTimestamp.Unix())
}

func (database *Database) AddSnapshot(timestamp time.Time, hostname string, title string,
	contents string) {
	snapshot := &Snapshot{-1, timestamp.Unix(), hostname, title, contents}
	err := database.mapper.Insert(snapshot)
	if err != nil {
		panic(err)
	}

	database.cleanOldSnapshots()
}

func (database *Database) querySnapshots(query string, args ...interface{}) []Snapshot {
	var rows []Snapshot
	_, err := database.mapper.Select(&rows, query, args...)
	if err != nil {
		panic(err)
	}
	return rows
}

func uniqueTimestamps(snapshots []Snapshot, mapTimestamp func(time.Time) time.Time) []time.Time {
	timestamps := make([]time.Time, 0)
	seenMap := make(map[time.Time]bool)
	for _, snapshot := range snapshots {
		timestamp := mapTimestamp(snapshot.Timestamp())
		if _, seen := seenMap[timestamp]; !seen {
			timestamps = append(timestamps, timestamp)
			seenMap[timestamp] = true
		}
	}
	return timestamps
}

func (database *Database) GetAllDays() []time.Time {
	query := "SELECT DISTINCT UnixTimestamp FROM Snapshot ORDER BY UnixTimestamp"
	rows := database.querySnapshots(query)
	return uniqueTimestamps(rows, func(timestamp time.Time) time.Time {
		year, month, day := timestamp.Date()
		return time.Date(year, month, day, 0, 0, 0, 0, timestamp.Location())
	})
}

func (database *Database) GetTimestamps(day time.Time) []time.Time {
	query := "SELECT DISTINCT UnixTimestamp FROM Snapshot " +
		"WHERE UnixTimestamp >= ? AND UnixTimestamp < ? ORDER BY UnixTimestamp"
	rows := database.querySnapshots(query, day.Unix(), day.AddDate(0, 0, 1).Unix())
	return uniqueTimestamps(rows, func(timestamp time.Time) time.Time { return timestamp })
}

func (database *Database) GetSnapshots(timestamp time.Time) []Snapshot {
	query := "SELECT Id, UnixTimestamp, Hostname, Title FROM Snapshot WHERE UnixTimestamp = ? " +
		"ORDER BY Hostname, Title"
	rows := database.querySnapshots(query, timestamp.Unix())

	return rows
}

func (database *Database) GetSnapshotContents(timestamp time.Time, hostname string,
	title string) (contents string, ok bool) {
	query := "SELECT Contents FROM Snapshot WHERE UnixTimestamp = ? AND Hostname = ? AND Title = ?"
	rows := database.querySnapshots(query, timestamp.Unix(), hostname, title)
	if len(rows) == 0 {
		return "", false
	} else if len(rows) == 1 {
		return rows[0].Contents, true
	} else {
		panic(
			fmt.Sprintf("Multiple snapshots found: timestamp %q, hostname %q, title %q",
				timestamp, hostname, title,
			),
		)
	}
}
