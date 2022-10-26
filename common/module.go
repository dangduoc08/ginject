package common

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/dangduoc08/gooh/routing"
	"github.com/dangduoc08/gooh/utils"
)

type Module struct {
	*sync.Mutex
	singleInstance *Module
	Imports        []*Module
	Providers      []Provider
	Exports        []Provider
	Presenters     []Presenter
	Router         *routing.Route
	OnInit         func()
}

func (m *Module) Inject() *Module {
	m.Mutex.Lock()
	defer m.Mutex.Unlock()

	if m.singleInstance == nil {
		m.singleInstance = m
		if m.OnInit != nil {
			m.OnInit()
		}

		noInjectedFields := []string{
			"Rest",
			"common.Rest",
		}

		for _, subModule := range m.Imports {
			injectModule := subModule.Inject()

			if len(injectModule.Exports) > 0 {
				m.Providers = append(m.Providers, injectModule.Exports...)
			}

			m.Router.Group("/", injectModule.Router)
		}

		providerMap := map[string]Provider{}
		for i, provider := range m.Providers {
			providerKey := reflect.TypeOf(provider).String()
			m.Providers = append(m.Providers, provider.New())
			providerMap[providerKey] = m.Providers[i]
		}

		for i, presenter := range m.Presenters {
			presenterType := reflect.TypeOf(presenter)
			copyPresenter := reflect.New(presenterType)

			for j := 0; j < presenterType.NumField(); j++ {
				injectProviderKey := presenterType.Field(j).Type.String()
				fieldName := presenterType.Field(j).Name
				if utils.StrIsLower(fieldName[0:1])[0] {
					panic(fmt.Errorf("can't set value to unexported %v field of the %v presenter", fieldName, presenterType.Name()))
				}

				isUnneededInject := utils.ArrIncludes(noInjectedFields, injectProviderKey)

				if providerMap[injectProviderKey] != nil && !isUnneededInject {
					copyPresenter.Elem().Field(j).Set(reflect.ValueOf(providerMap[injectProviderKey]))
				} else {
					if isUnneededInject {
						continue
					}
					panic(fmt.Errorf("can't resolve dependencies of the %v provider. Please make sure that the argument dependency at index [%v] is available in the %v presenter", injectProviderKey, j, presenterType.Name()))
				}
			}

			m.Presenters[i] = copyPresenter.Interface().(Presenter).New()

			for pattern, handlers := range reflect.ValueOf(m.Presenters[i]).FieldByName(noInjectedFields[0]).Interface().(Rest).routerMap {
				m.Router.Add(pattern, handlers...)
			}
		}
	}

	return m
}
