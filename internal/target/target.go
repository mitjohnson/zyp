package target

type Kind string

const (
	KindFile     Kind = "file"
	KindSQLite   Kind = "sqlite"
	KindPostgres Kind = "postgres"
)

type Target struct {
	Name       string
	Kind       Kind
	Source     string
	Repository string
	Compress   bool
}
