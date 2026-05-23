package httpclient

import (
	"github.com/dangduoc08/ginject/core"
)

// ClientService is the injectable DI provider wrapping a Client.
// Embed it in controllers or providers to access HTTP client methods directly.
type ClientService struct {
	Backend Client
}

func (cs ClientService) NewProvider() core.Provider { return cs }

func (cs *ClientService) Get(path string) RequestBuilder    { return cs.Backend.Get(path) }
func (cs *ClientService) Post(path string) RequestBuilder   { return cs.Backend.Post(path) }
func (cs *ClientService) Put(path string) RequestBuilder    { return cs.Backend.Put(path) }
func (cs *ClientService) Patch(path string) RequestBuilder  { return cs.Backend.Patch(path) }
func (cs *ClientService) Delete(path string) RequestBuilder { return cs.Backend.Delete(path) }
func (cs *ClientService) Head(path string) RequestBuilder   { return cs.Backend.Head(path) }
func (cs *ClientService) Options(path string) RequestBuilder {
	return cs.Backend.Options(path)
}
func (cs *ClientService) Download(rawURL, filepath string) error {
	return cs.Backend.Download(rawURL, filepath)
}
func (cs *ClientService) DownloadWithProgress(rawURL, filepath string, fn func(Progress)) error {
	return cs.Backend.DownloadWithProgress(rawURL, filepath, fn)
}
