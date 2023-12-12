package context

import (
	"fmt"
	"mime/multipart"

	"github.com/dangduoc08/gooh/exception"
	"github.com/dangduoc08/gooh/utils"
)

type FileHandler interface {
	IsValid(*FileData) bool
	Store(*FileData, multipart.File)
}

type FileData struct {
	Index    int
	Size     int64
	Key      string
	Filename string
	Type     string
}

type File map[string][]*multipart.FileHeader

func (c *Context) File() File {
	if c.file != nil {
		return c.file
	}

	if c.Request.MultipartForm != nil {
		c.file = c.Request.MultipartForm.File
	}

	return c.file
}

func (files File) Bind(s any) any {
	keys, newStructuredData := BindFile(files, s)

	if fileHandler, ok := s.(FileHandler); ok {
		for key, fileHeaders := range files {
			if utils.ArrIncludes[string](keys, key) {
				for i, fileHeader := range fileHeaders {
					isValid := fileHandler.IsValid(&FileData{
						Index:    i,
						Size:     fileHeader.Size,
						Key:      key,
						Filename: fileHeader.Filename,
						Type:     fileHeader.Header.Get("Content-Type"),
					})

					if !isValid {
						panic(exception.BadRequestException(fmt.Sprintf(
							"Invalid file upload. Please make sure '%v' is a supported file",
							fileHeader.Filename,
						)))
					}
				}
			}
		}

		for key, fileHeaders := range files {
			if utils.ArrIncludes[string](keys, key) {
				for i, fileHeader := range fileHeaders {
					f, err := fileHeader.Open()
					if err != nil {
						f.Close()
						panic(exception.BadRequestException(err.Error()))
					}
					fileHandler.Store(&FileData{
						Index:    i,
						Size:     fileHeader.Size,
						Key:      key,
						Filename: fileHeader.Filename,
						Type:     fileHeader.Header.Get("Content-Type"),
					}, f)
					f.Close()
				}
			}
		}
	}

	return newStructuredData
}
