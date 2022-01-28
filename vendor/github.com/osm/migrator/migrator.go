package migrator

import (
	"database/sql"
	"fmt"
	"sort"

	"github.com/osm/migrator/repository"
)

// ToLatest migrates the database to the latest version.
func ToLatest(db *sql.DB, repo repository.Source) error {
	return run(db, repo, -1)
}

// ToVersion migrates the database to a specific version.
//
// Setting the version to a version earlier than the database
// currently has is not supported.
func ToVersion(db *sql.DB, repo repository.Source, toVersion int) error {
	return run(db, repo, toVersion)
}

// isSqlite checks if the connection is a sqlite connection.
func isSqlite(db *sql.DB) bool {
	var v string
	if err := db.QueryRow("SELECT sqlite_version()").Scan(&v); err != nil {
		return false
	}

	return true
}

// run executes the migrations found in the repository.
func run(db *sql.DB, repo repository.Source, toVersion int) error {
	// Make sure that the db connection is alive
	err := db.Ping()
	if err != nil {
		return err
	}

	// Cast version to int when we are using a sqlite database.
	var currentVersionQuery string
	if isSqlite(db) {
		currentVersionQuery = "SELECT version FROM migration ORDER BY cast(version AS int) DESC"
	} else {
		currentVersionQuery = "SELECT version FROM migration ORDER BY version DESC"
	}

	// Get the current version of the database schema
	// If we can't find a migration table we assume that it has never been
	// executed before and therefore we set the version to -1.
	var currentVersion int
	err = db.QueryRow(currentVersionQuery).Scan(&currentVersion)
	if err != nil {
		currentVersion = -1
	}

	// Load all the migrations from the repository
	migrations, err := repo.Load()
	if err != nil {
		return err
	}

	// versions contains a slice of all migration versions
	var versions []int

	// Store each version in the slice
	for v := range migrations {
		versions = append(versions, v)
	}

	// Now, let's sort the versions
	sort.Ints(versions)

	// Iterate over each version
	for _, v := range versions {
		// Don't apply a version that already has been applied
		if currentVersion != -1 && currentVersion >= v {
			continue
		}

		// Execute the migration
		if err := executeQuery(db, migrations[v]); err != nil {
			return err
		}

		// Insert the migration version into the migration table
		// We don't care about SQL injections here since we know that the input
		// always will be an integer.
		if err := executeQuery(db, fmt.Sprintf("INSERT INTO migration VALUES(%d);", v)); err != nil {
			return err
		}

		// Make sure we don't migrate too far when set toVersion is set.
		if toVersion > 0 && v == toVersion {
			break
		}
	}

	return nil
}

// executeQuery executes the given query within a new transaction.
func executeQuery(db *sql.DB, query string) error {
	// Start a new transaction
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	// Execute the migration
	// Rollback if there's an error
	_, err = tx.Exec(query)
	if err != nil {
		// We don't check for errors on the rollback
		// We are more interested of the error that caused the rollback
		tx.Rollback()
		return err
	}

	// Commit the transaction
	// If there's an error, return it
	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}
