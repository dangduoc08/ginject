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
	Index    int
	Size     int64
	Total    int
	Key      string
	Filename string
	Type     string
	Dest     string
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
				src, err := dataFile.FileHeader.Open()
				if err != nil {
					src.Close()
					panic(exception.BadRequestException(err.Error()))
				}
				fileHandler.Store(dataFile, src)
				src.Close()
			}
		}
	}

	return newStructuredData
}
