package timeturner

import (
	"database/sql"
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
	mapper.TraceOn("[gorp]", log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)) // TODO
	mapper.AddTable(Snapshot{}).SetKeys(true, "Id")

	_, err := mapper.Exec(SCHEMA)
	if err != nil {
		panic(err)
	}

	return &Database{mapper, nowFunc}
}

func (database *Database) cleanOldSnapshots() { // TODO
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

func (database *Database) GetAllDays() []time.Time {
	query := "SELECT DISTINCT UnixTimestamp FROM Snapshot ORDER BY UnixTimestamp"
	var rows []Snapshot
	_, err := database.mapper.Select(&rows, query)
	if err != nil {
		panic(err)
	}

	timestamps := make([]time.Time, 0)
	seenMap := make(map[time.Time]bool)
	for _, snapshot := range rows {
		year, month, day := snapshot.Timestamp().Date()
		date := time.Date(year, month, day, 0, 0, 0, 0, snapshot.Timestamp().Location())
		if _, seen := seenMap[date]; !seen {
			timestamps = append(timestamps, date)
			seenMap[date] = true
		}
	}
	return timestamps
}

func (database *Database) GetTimestamps(day time.Time) []time.Time {
	return []time.Time{}
}

func (database *Database) GetSnapshots(timestamp time.Time) []Snapshot {
	return []Snapshot{}
}

func (database *Database) GetSnapshotContents(timestamp time.Time, hostname string,
	title string) string {
	return ""
}
