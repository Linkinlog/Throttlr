package db

import "database/sql"

func NewKeyStore(db *sql.DB) *KeyStore {
	return &KeyStore{db: db}
}

type KeyStore struct {
	db *sql.DB
}

func (ks *KeyStore) Exists(key string) (bool, int) {
	var exists bool
	var id int
	_ = ks.db.QueryRow("SELECT EXISTS(SELECT id FROM api_keys WHERE key = ?)", key).Scan(&exists)
	if exists {
		_ = ks.db.QueryRow("SELECT id FROM api_keys WHERE key = ?", key).Scan(&id)
	}
	return exists, id
}

func (ks *KeyStore) Valid(id int) bool {
	var valid bool
	_ = ks.db.QueryRow("SELECT valid FROM api_keys WHERE id = ?", id).Scan(&valid)
	return valid
}

func (ks *KeyStore) IdFromKey(key string) (string, error) {
	var id string
	err := ks.db.QueryRow("SELECT id FROM api_keys WHERE key = ?", key).Scan(&id)
	return id, err
}
