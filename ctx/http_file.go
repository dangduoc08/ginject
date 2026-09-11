package ctx

import (
	"fmt"
	"mime/multipart"

	"github.com/dangduoc08/ginject/exception"
)

type fileValidator interface {
	IsValid(*DataFile) bool
}

type fileHandler interface {
	Store(*DataFile, multipart.File)
}

type DataFile struct {
	*multipart.FileHeader
	Key      string
	Filename string
	Type     string
	Dest     string
	Index    int
	Size     int64
	Total    int
}

type File map[string][]*multipart.FileHeader

func (c *HTTPContext) File() File {
	if c.file != nil {
		return c.file
	}

	if c.MultipartForm != nil {
		c.file = c.MultipartForm.File
	}

	return c.file
}

// storeDataFile closes the opened part even when Store panics. Panicking with
// an exception is this framework's error idiom, so the deferred close is what
// keeps a rejected upload from leaking the file handle.
func storeDataFile(handler fileHandler, dataFile *DataFile) {
	src, err := dataFile.Open()
	if err != nil {
		panic(exception.BadRequestException(err.Error()))
	}
	defer func() { _ = src.Close() }()

	handler.Store(dataFile, src)
}

func (files File) Bind(s any) any {
	filteredFile, newStructuredData := BindFile(files, s)

	if fileValidator, ok := s.(fileValidator); ok {
		for _, dataFileArr := range filteredFile {
			for _, dataFile := range dataFileArr {
				isValid := fileValidator.IsValid(dataFile)

				if !isValid {
					panic(exception.BadRequestException(fmt.Sprintf(
						"Invalid file upload. Please make sure '%v' is a supported file",
						dataFile.Filename,
					)))
				}
			}
		}
	}

	if fileHandler, ok := s.(fileHandler); ok {
		for _, dataFileArr := range filteredFile {
			for _, dataFile := range dataFileArr {
				storeDataFile(fileHandler, dataFile)
			}
		}
	}

	return newStructuredData
}
