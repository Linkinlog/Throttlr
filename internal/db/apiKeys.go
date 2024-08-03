package db

import "database/sql"

func NewKeyStore(db *sql.DB) *KeyStore {
	return &KeyStore{db: db}
}

type KeyStore struct {
	db *sql.DB
}

func (ks *KeyStore) Valid(key string) bool {
	var valid bool
	_ = ks.db.QueryRow("SELECT valid FROM api_keys WHERE key = ?", key).Scan(&valid)
	return valid
}

func (ks *KeyStore) IdFromKey(key string) (string, error) {
	var id string
	err := ks.db.QueryRow("SELECT id FROM api_keys WHERE key = ?", key).Scan(&id)
	return id, err
}
