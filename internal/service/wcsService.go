package service

import (
	"AstralTest/internal/models/entity"
	"AstralTest/internal/storage"
	"AstralTest/internal/storage/cache"
	"AstralTest/pkg/appError"
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

type wcs struct {
	sessionStorage      storage.SessionStorage
	fileStorage         storage.FileStorage
	savingFileDirectory string
	cache               cache.Cache
}

type WcsService interface {
	HandleUploadingFile(ctx context.Context, doc entity.Document, fileData *[]byte) (*uuid.UUID, error)
	GetFile(ctx context.Context, token, fileId uuid.UUID, headOnly bool) (*entity.Document, string, error)
	GetFilesList(ctx context.Context, headOnly bool, token uuid.UUID, ownerLogin, key, value string, limit int) (json.RawMessage, error)
	DeleteFile(ctx context.Context, token, fileId uuid.UUID) error
}

func NewWcsService(sessionStorage storage.SessionStorage, fileStorage storage.FileStorage, savingFileDirectory string, cache cache.Cache) WcsService {
	return &wcs{
		sessionStorage:      sessionStorage,
		fileStorage:         fileStorage,
		savingFileDirectory: savingFileDirectory,
		cache:               cache,
	}
}

func (wc *wcs) HandleUploadingFile(ctx context.Context, doc entity.Document, fileData *[]byte) (*uuid.UUID, error) {
	token, err := uuid.Parse(doc.Token)
	if err != nil {
		return nil, appError.BadRequest("invalid meta.Token")
	}
	doc.Owner, err = wc.sessionStorage.GetSession(ctx, token)
	if err != nil {
		return nil, err
	}

	doc.ID = uuid.New()

	err = wc.fileStorage.SaveDoc(ctx, &doc)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			wc.fileStorage.DeleteDoc(ctx, doc.ID)
			if doc.File {
				wc.fileStorage.RmFile(doc.ID)
			}
		}
	}()

	if doc.File {
		err := wc.fileStorage.SaveFile(doc.ID, fileData)
		if err != nil {
			return nil, appError.Internal()
		}
	}

	// invalidate owner + grantees
	wc.cache.InvalidateOwnerList(doc.Owner)
	wc.cache.InvalidateGrant(doc.Owner, doc.Grant)

	return &doc.ID, nil
}

func (wc *wcs) GetFile(ctx context.Context, token, fileId uuid.UUID, headOnly bool) (*entity.Document, string, error) {
	userLogin, err := wc.sessionStorage.GetSession(ctx, token)
	if err != nil {
		return nil, "", err
	}

	cacheKey := makeFileKey(fileId)

	// Try cache
	if cached, ok := wc.cache.GetOwner(userLogin, cacheKey); ok {
		if headOnly {
			return nil, "", nil
		}
		var document entity.Document
		if err := json.Unmarshal(cached.Body, &document); err == nil {
			return &document, wc.fileStorage.GetFilePath(fileId), nil
		}
	}

	document, err := wc.fileStorage.GetDoc(ctx, fileId)
	if err != nil {
		return nil, "", err
	}
	if !userIsAllowedToFile(document, userLogin) {
		return nil, "", appError.Forbidden()
	}

	if !headOnly {
		body, _ := json.Marshal(document)
		wc.cache.SetOwner(userLogin, cacheKey, cache.CachedDocResp{
			Status: 200,
			Body:   body,
		})
	}

	var filePath string
	if document.File && !headOnly {
		filePath = wc.fileStorage.GetFilePath(fileId)
	}
	return document, filePath, nil
}

func (wc *wcs) GetFilesList(ctx context.Context, headOnly bool, token uuid.UUID, ownerLogin, key, value string, limit int) (json.RawMessage, error) {
	userLogin, err := wc.sessionStorage.GetSession(ctx, token)
	if err != nil {
		return nil, err
	}

	cacheKey := makeOwnerListKey(ownerLogin, key, value, userLogin)

	// check if cache exists
	if ownerLogin == "" {
		if cached, ok := wc.cache.GetOwner(userLogin, cacheKey); ok {
			if headOnly {
				return nil, nil
			}
			return cached.Body, nil
		}
	} else {
		if cached, ok := wc.cache.GetGrant(userLogin, ownerLogin, cacheKey); ok {
			if headOnly {
				return nil, nil
			}
			return cached.Body, nil
		}
	}

	res, err := wc.fileStorage.GetDocsList(ctx, ownerLogin, userLogin, key, value, limit)
	if err != nil {
		return nil, err
	}

	jsonOut, err := json.Marshal(res)
	if err != nil {
		return nil, appError.Internal()
	}

	// set cache
	if ownerLogin == "" {
		wc.cache.SetOwner(userLogin, cacheKey, cache.CachedDocResp{
			Status: 200,
			Body:   jsonOut,
		})
	} else {
		wc.cache.SetGrant(userLogin, ownerLogin, cacheKey, cache.CachedDocResp{
			Status: 200,
			Body:   jsonOut,
		})
	}

	return json.RawMessage(jsonOut), nil
}

func (wc *wcs) DeleteFile(ctx context.Context, token, fileId uuid.UUID) error {
	userLogin, err := wc.sessionStorage.GetSession(ctx, token)
	if err != nil {
		return appError.BadRequest("session doesn't exist")
	}

	doc, err := wc.fileStorage.GetDoc(ctx, fileId)
	if err != nil {
		return err
	}

	if doc.Owner != userLogin {
		return appError.Forbidden()
	}

	wc.cache.InvalidateOwnerList(doc.Owner)
	wc.cache.InvalidateGrant(doc.Owner, doc.Grant)

	if doc.File {
		err = wc.fileStorage.RmFile(fileId)
		if err != nil {
			return appError.Internal()
		}
	}

	err = wc.fileStorage.DeleteDoc(ctx, fileId)
	if err != nil {
		return appError.Internal()
	}

	return nil
}

func userIsAllowedToFile(doc *entity.Document, userLogin string) bool {
	if doc.Public || doc.Owner == userLogin {
		return true
	}
	for _, login := range doc.Grant {
		if login == userLogin {
			return true
		}
	}
	return false
}

func makeOwnerListKey(ownerLogin, key, value, viewerLogin string) cache.CacheKey {
	return cache.CacheKey(fmt.Sprintf("list:%s:%s:%s:viewedBy:%s", ownerLogin, key, value, viewerLogin))
}

func makeFileKey(fileId uuid.UUID) cache.CacheKey {
	return cache.CacheKey(fmt.Sprintf("file:%s", fileId.String()))
}
