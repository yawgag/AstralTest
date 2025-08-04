package storage

import (
	"AstralTest/internal/models/entity"
	"AstralTest/internal/storage/postgres"
	"AstralTest/pkg/appError"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type wcs struct {
	pool                postgres.DBPool
	fileSavingDirectory string
}

type FileStorage interface {
	SaveFile(fileUUID uuid.UUID, fileData *[]byte) error
	GetFilePath(fileUUID uuid.UUID) string
	RmFile(fileUUID uuid.UUID) error

	SaveDoc(ctx context.Context, doc *entity.Document) error
	DeleteDoc(ctx context.Context, id uuid.UUID) error
	GetDoc(ctx context.Context, id uuid.UUID) (*entity.Document, error)
	GetDocsList(ctx context.Context, ownerLogin, login, key, value string, limit int) ([]*entity.Document, error)
}

func NewFileStorage(pool postgres.DBPool, dir string) (FileStorage, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	return &wcs{
		fileSavingDirectory: dir,
		pool:                pool,
	}, nil
}

func (wc *wcs) SaveFile(fileUUID uuid.UUID, fileData *[]byte) error {
	fileUUIDStr := fileUUID.String()

	// create file path
	filePath := filepath.Join(wc.fileSavingDirectory, fileUUIDStr)
	if err := os.WriteFile(filePath, *fileData, 0644); err != nil {
		return appError.Internal()
	}
	return nil
}

func (wc *wcs) GetFilePath(fileUUID uuid.UUID) string {
	return filepath.Join(wc.fileSavingDirectory, fileUUID.String())
}

func (wc *wcs) RmFile(fileUUID uuid.UUID) error {
	filePath := filepath.Join(wc.fileSavingDirectory, fileUUID.String())
	// TODO: handle more error
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		log.Printf("Failed to delete file %s: %v", filePath, err)
	}
	return nil
}

func (wc *wcs) SaveDoc(ctx context.Context, doc *entity.Document) error {
	query := `insert into docs(id, name, mime, file, public, owner_login, grant_logins, json_data)
				values($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := wc.pool.Exec(ctx, query,
		doc.ID,
		doc.Name,
		doc.Mime,
		doc.File,
		doc.Public,
		doc.Owner,
		doc.Grant,
		doc.JsonData,
	)

	return err
}

func (wc *wcs) GetDoc(ctx context.Context, id uuid.UUID) (*entity.Document, error) {
	query := `select id, name, mime, file, public, owner_login, grant_logins, json_data
			from docs
			where id = $1`
	var doc entity.Document
	err := wc.pool.QueryRow(ctx, query, id).Scan(
		&doc.ID,
		&doc.Name,
		&doc.Mime,
		&doc.File,
		&doc.Public,
		&doc.Owner,
		&doc.Grant,
		&doc.JsonData)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, appError.BadRequest("file doesnt exist")
		}
		log.Println("[GetDoc] error: ", err.Error())
		return nil, appError.Internal()
	}
	return &doc, nil
}

func (wc *wcs) GetDocsList(ctx context.Context, ownerLogin, login, key, value string, limit int) ([]*entity.Document, error) {
	var args []interface{}

	query := `select id, name, mime, file, public, created, owner_login, grant_logins, json_data 
            	from docs 
             	where `

	// add login sorting
	if ownerLogin == "" {
		query += fmt.Sprintf(" owner_login = $%d", len(args)+1)
		args = append(args, login)
	} else {
		query += fmt.Sprintf(" owner_login = $%d and (public = true or $%d = ANY(grant_logins))", len(args)+1, len(args)+2)
		args = append(args, ownerLogin, login)
	}

	// add requset sorting values
	if key != "" && value != "" {
		query += fmt.Sprintf(" AND %s = $%d", key, len(args)+1)
		args = append(args, value)
	}
	// add sorting by name and creation time
	query += " ORDER BY name, created"

	// add limit
	query += fmt.Sprintf(" LIMIT $%d", len(args)+1)
	args = append(args, limit)

	rows, err := wc.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, appError.Internal()
	}
	defer rows.Close()

	var docs []*entity.Document
	for rows.Next() {
		var doc entity.Document
		err := rows.Scan(
			&doc.ID,
			&doc.Name,
			&doc.Mime,
			&doc.File,
			&doc.Public,
			&doc.Created,
			&doc.Owner,
			&doc.Grant,
			&doc.JsonData,
		)
		if err != nil {
			return nil, appError.Internal()
		}
		docs = append(docs, &doc)
	}

	return docs, nil

}

func (wc *wcs) DeleteDoc(ctx context.Context, id uuid.UUID) error {
	query := `delete from docs
				where id = $1`

	_, err := wc.pool.Exec(ctx, query, id)
	if err != nil {
		return appError.Internal()
	}
	return nil
}
