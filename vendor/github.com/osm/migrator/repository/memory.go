package repository

// memory holds the implementation of the Source interface.
type memory struct{ migrations map[int]string }

// FromMemory creates a new memory based repository.
//
// The memory based repository is useful if you want to embed the migrations
// within your application. It expects a map of all migrations and versions
// to be passed on invocation.
func FromMemory(migrations map[int]string) Source {
	return &memory{
		migrations: migrations,
	}
}

// Load returns all migrations that exists within the memory repository.
func (r *memory) Load() (map[int]string, error) {
	return r.migrations, nil
}
