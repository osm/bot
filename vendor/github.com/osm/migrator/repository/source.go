package repository

// Source represents a migration repository.
//
// The load method is responsible for loading a set of migrations.
// It should return a map with all the mirations.
// The key of the map should be the version of the migration.
// The value for each key should contain the SQL statements for the migration.
type Source interface {
	Load() (map[int]string, error)
}
