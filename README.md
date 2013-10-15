# Timeturner

Timeturner lets you view detailed system snapshots for the recent past. It helps you dig into system
problems by answering questions like *What process was using all that memory?* or *What were all
those queries that started running?*

Timeturner accepts snapshots through a simple REST API, stores them in a SQLite database, and serves
them through a web interface.

## Adding data

Simply PUT your snapshot data as a CSV (with a header row) to `/<date>/<time>/<hostname>/<title>`, e.g.,

```bash
curl -X PUT --data-binary "name,quote
steve,hello world
howard,goodbye world
" 'http://localhost:8080/2013-10-05/15:32:44/stevebox/quotes/'
```
