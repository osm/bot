## About

Migrator is a simple and easy to use database migration library.

## Supported database engines

All databases that has a go driver should work out of the box.

It has been successfully tested on Postgres and SQLite.

See the following link for a complete list of drivers: https://github.com/golang/go/wiki/SQLDrivers

## Migration repositories

The migrations are stored within a `Repository`, it can either be file based or stored directly in memory. Check within the repositores directory for more information.

## Usage example

```go
package main

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
	"github.com/osm/migrator"
	"github.com/osm/migrator/repository"
)

func main() {
	// Initialize a new test db
	db, err := sql.Open("sqlite3", "./test.db")
	if err != nil {
		panic(err)
	}

	// Create a new mem repo
	repo := repository.FromMemory(map[int]string{
		1: "CREATE TABLE migration (version text NOT NULL PRIMARY KEY);\n",
		2: "CREATE TABLE foo (version text NOT NULL PRIMARY KEY);\n",
		3: "INSERT INTO foo VALUES(123);\n",
	})

	// Migrate the database to the latest version
	if err := migrator.ToLatest(db, repo); err != nil {
		panic(err)
	}
}
```
