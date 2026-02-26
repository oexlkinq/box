package storage

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type Folder struct {
	storage *Storage
}

func (s *Storage) Allocate(size int64) (*Folder, error) {
	sizeQuota := s.maxStorageSize - size
	if sizeQuota < 0 {
		return nil, ErrFileIsTooLarge
	}

	// освобождение места для файла
	tx, err := s.dbc.Beginx()
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
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
		return nil, fmt.Errorf("query excess files: %w", err)
	}

	// удаление вытесненных файлов
	file := File{}
	for rows.Next() {
		err := rows.StructScan(&file)
		if err != nil {
			return nil, fmt.Errorf("scan file row: %w", err)
		}

		// удалить файл из файловой системы по полученному пути
		err = os.Remove(file.Path)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return nil, fmt.Errorf("remove file: %w", err)
			}

			slog.Warn(fmt.Sprintf(`file "%s" already doesn't exists`, err))
		}

		// удалить запись о файле из бд
		_, err = tx.Exec(`DELETE FROM files WHERE path = ?`, file.Path)
		if err != nil {
			return nil, fmt.Errorf("delete file row: %w", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return &Folder{s}, nil
}

var ErrFileIsTooLarge = errors.New("file is larger than max size of storage")

func (folder *Folder) Put(r io.Reader, url string) (*File, error) {
	err := folder.storage.Delete(url)
	if err != nil {
		return nil, fmt.Errorf("delete old file: %w", err)
	}

	tx, err := folder.storage.dbc.Beginx()
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}

	defer tx.Rollback()

	ts := int(time.Now().Unix())
	file := File{
		Ts:   ts,
		Path: strconv.Itoa(ts),
		Url:  url,
	}

	// записать файл на диск
	f, err := os.Create(filepath.Join(folder.storage.storagePath, file.Path))
	if err != nil {
		return nil, fmt.Errorf("create file: %w", err)
	}

	defer f.Close()

	file.Size, err = io.Copy(f, r)
	if err != nil {
		return nil, fmt.Errorf("write file: %w", err)
	}

	// сделать в бд запись о новом файле
	_, err = tx.NamedExec(`INSERT INTO files (ts, url, path, size) VALUES (:ts, :url, :path, :size)`, file)
	if err != nil {
		return nil, fmt.Errorf("insert file row: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return &file, nil
}
