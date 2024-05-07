package store

import (
	"fmt"
	"github.com/charmbracelet/charm/kv"
)

type KV struct {
	*kv.KV
	mock map[string]string
}

func New(namespace string) (*KV, error) {
	if namespace == ":memory:" {
		return &KV{
			mock: make(map[string]string),
		}, nil
	}
	db, err := kv.OpenWithDefaults(namespace)
	if err != nil {
		return nil, err
	}
	return &KV{
		KV: db,
	}, nil
}

func (db *KV) Get(key string) (string, error) {
	if db.KV == nil {
		v, ok := db.mock[key]
		if !ok {
			return "", fmt.Errorf("key not found")
		}
		return v, nil
	}
	v, err := db.KV.Get([]byte(key))
	return string(v), err
}

func (db *KV) Set(key string, value string) error {
	if db.KV == nil {
		db.mock[key] = value
		return nil
	}
	return db.KV.Set([]byte(key), []byte(value))
}

func (db *KV) Keys() ([]string, error) {
	out := []string{}
	if db.KV == nil {
		for k := range db.mock {
			out = append(out, k)
		}
		return out, nil
	}
	keys, err := db.KV.Keys()
	for k := range keys {
		out = append(out, fmt.Sprint(k))
	}
	return out, err
}
