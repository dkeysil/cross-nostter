package sqlstore

import "github.com/jmoiron/sqlx"

type SqlStore struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *SqlStore {
	return &SqlStore{db: db}
}
