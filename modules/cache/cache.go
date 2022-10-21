package cache

type Cache struct{}

type Cacher interface {
	Get(string) interface{}
	Set(string, interface{})
	Del(string)
	Has(string) bool
}
