package transport

import (
	"AstralTest/internal/models/entity"
	"AstralTest/internal/models/response"
	"AstralTest/internal/service"
	"AstralTest/pkg/appError"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

// TODO: О названии wcs - web cache storage. Плохая фантазия, нужно придумать получше.
type WcsHandler struct {
	service service.WcsService
}

func NewWcsHandler(service service.WcsService) *WcsHandler {
	return &WcsHandler{
		service: service,
	}
}

type FileMetaData struct {
	Name   string   `json:"name"`
	File   bool     `json:"file"`
	Public bool     `json:"public"`
	Token  string   `json:"token"`
	Mime   string   `json:"mime"`
	Grant  []string `json:"grant"`
}

func (wc *WcsHandler) UploadFileHandler(w http.ResponseWriter, r *http.Request) {
	// check method
	if r.Method != http.MethodPost {
		writeError(w, appError.MethodNotAllowed())
		return
	}
	// check content-type
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		writeError(w, appError.BadRequest("expected multipart/form-data"))
		return
	}

	// max form size is 10mb
	const maxFormSize = 10 << 20
	if err := r.ParseMultipartForm(maxFormSize); err != nil {
		writeError(w, appError.BadRequest("failed to parse form"))
		return
	}

	// read and validate meta
	var metaData FileMetaData
	metaDataString := r.FormValue("meta")
	if metaDataString == "" {
		writeError(w, appError.BadRequest("meta field is required"))
		return
	}
	if err := json.Unmarshal([]byte(metaDataString), &metaData); err != nil {
		writeError(w, appError.BadRequest("invalid meta JSON: "+err.Error()))
		return
	}
	if metaData.Name == "" {
		writeError(w, appError.BadRequest("meta.name is required"))
		return
	}
	if metaData.Token == "" {
		writeError(w, appError.BadRequest("meta.token is required"))
		return
	}

	// TODO: converte token to uuid in entity.Document
	doc := entity.Document{
		Name:   metaData.Name,
		File:   metaData.File,
		Public: metaData.Public,
		Token:  metaData.Token,
		Mime:   metaData.Mime,
		Grant:  metaData.Grant,
	}

	// read file data
	var fileData []byte
	if metaData.File {
		temp, err := readFormFile(r)
		if err != nil {
			writeError(w, err)
			return
		}
		if len(temp) == 0 {
			writeError(w, appError.BadRequest("file is required (meta.file = true)"))
		}
		fileData = temp
	}

	// read json data if exists
	jsonDataString := r.FormValue("json")
	if jsonDataString != "" {
		var temp interface{}
		if err := json.Unmarshal([]byte(jsonDataString), &temp); err != nil {
			writeError(w, appError.BadRequest("invalid json field"))
			return
		}
		doc.JsonData = json.RawMessage(jsonDataString)
	}

	fileId, err := wc.service.HandleUploadingFile(r.Context(), doc, &fileData)
	if err != nil {
		writeError(w, err)
		return
	}

	// response to request
	resp := response.Standard{
		Data: &response.DataPayload{
			"file":    doc.Name,
			"file_id": fileId,
		},
	}

	if len(doc.JsonData) > 0 {
		(*resp.Data)["json"] = doc.JsonData
	}
	writeJSON(w, http.StatusOK, resp)
}

func readFormFile(r *http.Request) ([]byte, error) {
	file, header, err := r.FormFile("file")
	if err != nil {
		return nil, appError.BadRequest("file field is required when meta.file is true: " + err.Error())
	}
	defer file.Close()

	// TODO: maybe check safety of filename?
	// fileName := sanitizeFileName(header.Filename)
	if header.Filename == "" {
		return nil, appError.BadRequest("invalid file name")
	}

	// max file size
	const maxFileSize = 50 << 20 // 50 MB
	fileBytes, err := io.ReadAll(io.LimitReader(file, maxFileSize))
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, appError.BadRequest("file is too large (max 50 MB)")
		} else {
			return nil, appError.Internal()
		}
	}

	return fileBytes, nil
}

func (wc *WcsHandler) GetFileHandler(w http.ResponseWriter, r *http.Request) {
	var headOnly bool
	switch r.Method {
	case http.MethodGet:
		headOnly = false
	case http.MethodHead:
		headOnly = true
	default:
		writeError(w, appError.MethodNotAllowed())
		return
	}

	userToken, fileId, err := getFileFileOperationsData(r)
	if err != nil {
		writeError(w, err)
		return
	}

	fileData, filePath, err := wc.service.GetFile(r.Context(), *userToken, *fileId, headOnly)
	if err != nil {
		writeError(w, err)
		return
	}
	if fileData.File {
		w.Header().Set("Content-type", fileData.Mime)
		w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%s", url.PathEscape(fileData.Name)))

		if r.Method == http.MethodHead {
			return
		}

		// send file to user
		http.ServeFile(w, r, filePath)
	} else {
		resp := response.Standard{
			Data: &response.DataPayload{
				"json": fileData.JsonData,
			},
		}
		writeJSON(w, http.StatusOK, resp)
	}

}

func (wc *WcsHandler) GetFilesList(w http.ResponseWriter, r *http.Request) {
	var headOnly bool
	switch r.Method {
	case http.MethodGet:
		headOnly = false
	case http.MethodHead:
		headOnly = true
	default:
		writeError(w, appError.MethodNotAllowed())
		return
	}

	// take all parametrs
	tokenStr := r.URL.Query().Get("token")
	token, err := uuid.Parse(tokenStr)
	if err != nil {
		writeError(w, appError.BadRequest("bad user token"))
		return
	}
	ownerLogin := r.URL.Query().Get("login")
	key := r.URL.Query().Get("key")
	value := r.URL.Query().Get("value")
	var limit int
	limitStr := r.URL.Query().Get("limit")
	if limitStr == "" {
		limit = 100
	} else {
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			writeError(w, appError.BadRequest("limit value should me a number"))
			return
		}

		if limit < 1 {
			writeError(w, appError.BadRequest("limit value shuld be more than 0"))
			return
		}
	}

	out, err := wc.service.GetFilesList(r.Context(), headOnly, token, ownerLogin, key, value, limit)
	if err != nil {
		writeError(w, err)
		return
	}

	if headOnly {
		return
	}

	resp := response.Standard{
		Data: &response.DataPayload{
			"docs": out,
		},
	}

	writeJSON(w, http.StatusOK, resp)
}

func (wc *WcsHandler) DeleteDoc(w http.ResponseWriter, r *http.Request) {
	userToken, fileId, err := getFileFileOperationsData(r)
	if err != nil {
		writeError(w, err)
		return
	}

	err = wc.service.DeleteFile(r.Context(), *userToken, *fileId)
	if err != nil {
		writeError(w, err)
		return
	}

	resp := response.Standard{
		Data: &response.DataPayload{
			fileId.String(): true,
		},
	}
	writeJSON(w, http.StatusOK, resp)
}

func getFileFileOperationsData(r *http.Request) (*uuid.UUID, *uuid.UUID, error) {
	// get user token
	tokenStr := r.URL.Query().Get("token")
	token, err := uuid.Parse(tokenStr)
	if err != nil {
		return nil, nil, appError.BadRequest("bad user token")
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) != 4 || parts[3] == "" {
		return nil, nil, appError.BadRequest("bad request")
	}
	fileId, err := uuid.Parse(parts[3])
	if err != nil {
		return nil, nil, appError.BadRequest("bad file id")
	}
	return &token, &fileId, nil
}

// func sanitizeFileName(name string) string {
// 	// delete bad symbols
// 	name = strings.ReplaceAll(name, "/", "_")
// 	name = strings.ReplaceAll(name, "\\", "_")
// 	name = strings.ReplaceAll(name, "..", "_")

// 	// max length
// 	if len(name) > 255 {
// 		ext := filepath.Ext(name)
// 		name = name[:255-len(ext)] + ext
// 	}

// 	return name
// }
