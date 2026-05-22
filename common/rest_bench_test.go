package common

import "testing"

func BenchmarkGetPrefixes_NoHandlers(b *testing.B) {
	r := &REST{}
	r.Prefix("/api/v1")
	r.Prefix("/internal")
	b.ResetTimer()
	for range b.N {
		r.GetPrefixes()
	}
}

func BenchmarkGetPrefixes_WithHandlers(b *testing.B) {
	r := &REST{}
	r.Prefix("/api", fnTestController{}.READ_users, fnTestController{}.CREATE_orders)
	b.ResetTimer()
	for range b.N {
		r.GetPrefixes()
	}
}

func BenchmarkAddPrefixesToRoute(b *testing.B) {
	r := &REST{}
	r.Prefix("/api/v1")
	r.Prefix("/internal")
	prefixes := r.GetPrefixes()
	b.ResetTimer()
	for range b.N {
		r.addPrefixesToRoute("/users/", "READ_users", prefixes)
	}
}

func BenchmarkGetConfigurations(b *testing.B) {
	orig := InsertedRoutes
	InsertedRoutes = make(map[string]string)
	defer func() { InsertedRoutes = orig }()

	r := &REST{}
	for _, fn := range []string{
		"READ_items", "CREATE_items", "UPDATE_items", "DELETE_items",
		"READ_users", "CREATE_users", "UPDATE_users", "DELETE_users",
		"READ_orders", "CREATE_orders", "UPDATE_orders", "DELETE_orders",
	} {
		r.AddHandlerToRouterMap(nil, fn, nil)
	}
	b.ResetTimer()
	for range b.N {
		r.GetConfigurations()
	}
}
