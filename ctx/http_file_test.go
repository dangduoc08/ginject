package ctx

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"testing"

	"github.com/dangduoc08/ginject/internal/test"
)

func buildMultipartFileBody(t *testing.T, field, filename, content string) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile(field, filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	return &buf, mw.FormDataContentType()
}

func TestHTTPContext_File(t *testing.T) {
	buf, contentType := buildMultipartFileBody(t, "avatar", "photo.png", "fake-image-bytes")

	c := newTestHTTPContext()
	c.Request = httptest.NewRequest("POST", "/", buf)
	c.Request.Header.Set("Content-Type", contentType)

	if err := c.ParseMultipartForm(defaultMaxMemory); err != nil {
		t.Fatal(err)
	}

	f := c.File()
	if len(f["avatar"]) != 1 {
		t.Fatal(test.DiffMessage(len(f["avatar"]), 1, "File should expose the uploaded file under its field name"))
	}
	if f["avatar"][0].Filename != "photo.png" {
		t.Error(test.DiffMessage(f["avatar"][0].Filename, "photo.png", "uploaded file should keep its original filename"))
	}
}

func TestHTTPContext_FileNoMultipartForm(t *testing.T) {
	c := newTestHTTPContext()
	c.Request = httptest.NewRequest("GET", "/", nil)

	if c.File() != nil {
		t.Error(test.DiffMessage(c.File(), nil, "File should be nil when there is no multipart form"))
	}
}

type filePtrDTO struct {
	Avatar *DataFile `bind:"avatar"`
}

type fileSliceDTO struct {
	Photos []*DataFile `bind:"photos"`
}

func TestFile_BindPtr(t *testing.T) {
	buf, contentType := buildMultipartFileBody(t, "avatar", "photo.png", "fake-image-bytes")
	c := newTestHTTPContext()
	c.Request = httptest.NewRequest("POST", "/", buf)
	c.Request.Header.Set("Content-Type", contentType)
	if err := c.ParseMultipartForm(defaultMaxMemory); err != nil {
		t.Fatal(err)
	}

	result := c.File().Bind(filePtrDTO{})
	dto, ok := result.(filePtrDTO)
	if !ok {
		t.Fatal(test.DiffMessage(result, filePtrDTO{}, "Bind should return a filePtrDTO"))
	}
	if dto.Avatar == nil || dto.Avatar.Filename != "photo.png" {
		t.Error(test.DiffMessage(dto.Avatar, "photo.png", "Bind should populate the pointer field with the uploaded file"))
	}
}

func TestFile_BindSlice(t *testing.T) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for _, name := range []string{"a.png", "b.png"} {
		fw, err := mw.CreateFormFile("photos", name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fw.Write([]byte("x")); err != nil {
			t.Fatal(err)
		}
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}

	c := newTestHTTPContext()
	c.Request = httptest.NewRequest("POST", "/", &buf)
	c.Request.Header.Set("Content-Type", mw.FormDataContentType())
	if err := c.ParseMultipartForm(defaultMaxMemory); err != nil {
		t.Fatal(err)
	}

	result := c.File().Bind(fileSliceDTO{})
	dto, ok := result.(fileSliceDTO)
	if !ok {
		t.Fatal(test.DiffMessage(result, fileSliceDTO{}, "Bind should return a fileSliceDTO"))
	}
	if len(dto.Photos) != 2 {
		t.Fatal(test.DiffMessage(len(dto.Photos), 2, "Bind should populate the slice field with all uploaded files"))
	}
	if dto.Photos[0].Total != 2 {
		t.Error(test.DiffMessage(dto.Photos[0].Total, 2, "each bound DataFile should report the total count of uploaded files"))
	}
}

type fileValidatorDTO struct {
	Avatar *DataFile `bind:"avatar"`
}

func (fileValidatorDTO) IsValid(f *DataFile) bool {
	return f.Filename == "allowed.png"
}

func TestFile_BindPanicsWhenValidatorRejects(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("expected panic when the validator rejects the uploaded file")
		}
	}()

	buf, contentType := buildMultipartFileBody(t, "avatar", "rejected.png", "x")
	c := newTestHTTPContext()
	c.Request = httptest.NewRequest("POST", "/", buf)
	c.Request.Header.Set("Content-Type", contentType)
	if err := c.ParseMultipartForm(defaultMaxMemory); err != nil {
		t.Fatal(err)
	}

	c.File().Bind(fileValidatorDTO{})
}

type fileHandlerDTO struct {
	Avatar *DataFile `bind:"avatar"`
}

var storedContent string

func (fileHandlerDTO) Store(f *DataFile, src multipart.File) {
	b, _ := io.ReadAll(src)
	storedContent = string(b)
}

func TestFile_BindInvokesHandler(t *testing.T) {
	buf, contentType := buildMultipartFileBody(t, "avatar", "photo.png", "hello-bytes")
	c := newTestHTTPContext()
	c.Request = httptest.NewRequest("POST", "/", buf)
	c.Request.Header.Set("Content-Type", contentType)
	if err := c.ParseMultipartForm(defaultMaxMemory); err != nil {
		t.Fatal(err)
	}

	c.File().Bind(fileHandlerDTO{})
	if storedContent != "hello-bytes" {
		t.Error(test.DiffMessage(storedContent, "hello-bytes", "Bind should invoke the handler's Store with the file content"))
	}
}
