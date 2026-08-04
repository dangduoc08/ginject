package memorybroker

import ptrn "github.com/dangduoc08/ginject/pattern"

type complexGroup struct {
	pattern  ptrn.Pattern
	subsByID map[string]*subscription
}
