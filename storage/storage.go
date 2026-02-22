package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"time"

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
	url TEXT NOT NULL,
	path TEXT NOT NULL,
	size INT NOT NULL,
	PRIMARY KEY (ts, url, path)
)
`

func New(storagePath string, dbFilePath string, maxStorageSize int64) (*Storage, error) {
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

var ErrFileIsTooBig = errors.New("file is bigger than max size of storage")

func (s *Storage) Create(r io.Reader, url string, size int64) error {
	sizeQuota := s.maxStorageSize - size
	if sizeQuota < 0 {
		return ErrFileIsTooBig
	}

	// освобождение места для файла
	tx, err := s.dbc.Beginx()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	defer tx.Rollback()

	// выбор самых старых файлов которые должны быть вытеснены новым файлом
	rows, err := tx.Queryx(`
		SELECT path
		FROM (
			SELECT
				SUM(size) OVER (ORDER BY ts DESC) as csize,
				path
			FROM files
		) t
		WHERE t.csize >= ?
	`, sizeQuota)
	if err != nil {
		return fmt.Errorf("query excess files: %w", err)
	}

	// удаление вытесненных файлов
	file := File{}
	for rows.Next() {
		err := rows.StructScan(&file)
		if err != nil {
			return fmt.Errorf("scan file row: %w", err)
		}

		// удалить файл из файловой системы по полученному пути
		err = os.Remove(file.Path)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("remove file: %w", err)
			}

			slog.Warn(fmt.Sprintf(`file "%s" already doesn't exists`, err))
		}

		// удалить запись о файле из бд
		_, err = tx.Exec(`DELETE FROM files WHERE path = ?`, file.Path)
		if err != nil {
			return fmt.Errorf("delete file row: %w", err)
		}
	}

	file.Ts = int(time.Now().Unix())
	file.Path = filepath.Join(s.storagePath, strconv.Itoa(file.Ts))
	file.Size = size
	file.Url = url

	// сделать запись в бд
	_, err = tx.NamedExec(`INSERT INTO files (ts, url, path, size) VALUES (:ts, :url, :path, :size)`, file)
	if err != nil {
		return fmt.Errorf("insert file row: %w", err)
	}

	// записать файл на диск
	f, err := os.Create(file.Path)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}

	defer f.Close()

	_, err = io.Copy(f, r)
	if err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return tx.Commit()
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

	file, err := os.Open(path)
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
		return fmt.Errorf("delete file row: %w", err)
	}

	// удалить файл из файловой системы по полученному пути
	err = os.Remove(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove file: %w", err)
		}

		slog.Warn(fmt.Sprintf(`file "%s" already doesn't exists`, err))
	}

	return tx.Commit()
}
