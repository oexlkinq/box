package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

type Storage struct {
	dbc            *sqlx.DB
	maxStorageSize int64
	storagePath    string
}

type File struct {
	Ts   int
	Url  string
	Path string
	Size int64
}

const initQuery = `
CREATE TABLE IF NOT EXISTS files(
	ts INT NOT NULL,
	url TEXT PRIMARY KEY,
	path TEXT NOT NULL,
	size INT NOT NULL
)
`

func New(storagePath string, dbFilePath string, maxStorageSize int64) (*Storage, error) {
	err := os.MkdirAll(storagePath, 0700)
	if err != nil {
		return nil, fmt.Errorf("create storage path dir: %w", err)
	}

	err = os.MkdirAll(filepath.Base(dbFilePath), 0700)
	if err != nil {
		return nil, fmt.Errorf("create db file dir: %w", err)
	}

	dbc, err := sqlx.Connect("sqlite", dbFilePath)
	if err != nil {
		return nil, fmt.Errorf("connect sqlite db: %w", err)
	}

	_, err = dbc.Exec(initQuery)
	if err != nil {
		return nil, fmt.Errorf("initial table creation: %w", err)
	}

	return &Storage{
		dbc:            dbc,
		maxStorageSize: maxStorageSize,
		storagePath:    storagePath,
	}, nil
}

var ErrFileNotExists = errors.New("file not exists")

func (s *Storage) Get(url string) (*os.File, error) {
	path := ""
	err := s.dbc.Get(&path, `SELECT path FROM files WHERE url = ?`, url)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrFileNotExists
		}

		return nil, fmt.Errorf("get file path: %w", err)
	}

	file, err := os.Open(filepath.Join(s.storagePath, path))
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}

	return file, nil
}

func (s *Storage) Delete(url string) error {
	tx, err := s.dbc.Beginx()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	path := ""

	// удалить запись о файле из бд
	err = tx.Get(&path, `DELETE FROM files WHERE url = ? RETURNING path`, url)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return tx.Commit()
		}

		return fmt.Errorf("delete file row: %w", err)
	}

	// удалить файл из файловой системы по полученному пути
	err = os.Remove(filepath.Join(s.storagePath, path))
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove file: %w", err)
		}
	}

	return tx.Commit()
}

func (s *Storage) List(prefix string) ([]File, error) {
	files := []File{}
	err := s.dbc.Select(&files, `SELECT ts, url, path, size FROM files WHERE url like ?||'/%'`, prefix)
	if err != nil {
		return nil, fmt.Errorf("get file path: %w", err)
	}

	return files, nil
}
