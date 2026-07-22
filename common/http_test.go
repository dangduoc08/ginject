package common

import (
	"testing"

	"github.com/dangduoc08/ginject/internal/test"
)

func TestGetPrefixes_NoHandlers(t *testing.T) {
	r := &HTTP{}
	r.Prefix("/api")
	prefixes := r.GetPrefixes()
	if len(prefixes) != 1 {
		t.Error(test.DiffMessage(len(prefixes), 1, "one prefix entry"))
		return
	}
	for k, v := range prefixes[0] {
		if k != "/api/" {
			t.Error(test.DiffMessage(k, "/api/", "prefix key"))
		}
		if v != "*" {
			t.Error(test.DiffMessage(v, "*", "wildcard value"))
		}
	}
}

func TestGetPrefixes_WithHandlers(t *testing.T) {
	r := &HTTP{}
	r.Prefix("/v1", fnTestController{}.READ_users, fnTestController{}.CREATE_orders)
	prefixes := r.GetPrefixes()
	if len(prefixes) != 2 {
		t.Error(test.DiffMessage(len(prefixes), 2, "two prefix entries for two handlers"))
		return
	}
	names := map[string]bool{}
	for _, pm := range prefixes {
		for _, v := range pm {
			names[v] = true
		}
	}
	if !names["READ_users"] {
		t.Error(test.DiffMessage(names, "READ_users present", "handler name"))
	}
	if !names["CREATE_orders"] {
		t.Error(test.DiffMessage(names, "CREATE_orders present", "handler name"))
	}
}

func TestGetPrefixes_Empty(t *testing.T) {
	r := &HTTP{}
	prefixes := r.GetPrefixes()
	if len(prefixes) != 0 {
		t.Error(test.DiffMessage(len(prefixes), 0, "no prefixes configured"))
	}
}

func TestAddPrefixesToRoute_WildcardApplies(t *testing.T) {
	r := &HTTP{}
	r.Prefix("/api")
	prefixes := r.GetPrefixes()
	got := r.addPrefixesToRoute("/users/", "READ_users", prefixes)
	want := "/api/users/"
	if got != want {
		t.Error(test.DiffMessage(got, want, "wildcard prefix prepended without double slash"))
	}
}

func TestAddPrefixesToRoute_SpecificFnMatch(t *testing.T) {
	r := &HTTP{}
	r.Prefix("/v1", fnTestController{}.READ_users)
	prefixes := r.GetPrefixes()
	got := r.addPrefixesToRoute("/users/", "READ_users", prefixes)
	want := "/v1/users/"
	if got != want {
		t.Error(test.DiffMessage(got, want, "matching fn prefix prepended"))
	}
}

func TestAddPrefixesToRoute_SpecificFnNoMatch(t *testing.T) {
	r := &HTTP{}
	r.Prefix("/v1", fnTestController{}.READ_users)
	prefixes := r.GetPrefixes()
	got := r.addPrefixesToRoute("/orders/", "CREATE_orders", prefixes)
	want := "/orders/"
	if got != want {
		t.Error(test.DiffMessage(got, want, "non-matching fn prefix not prepended"))
	}
}

func TestAddPrefixesToRoute_NoPrefixes(t *testing.T) {
	r := &HTTP{}
	got := r.addPrefixesToRoute("/items/", "READ_items", nil)
	want := "/items/"
	if got != want {
		t.Error(test.DiffMessage(got, want, "no prefixes leaves route unchanged"))
	}
}

func TestAddToRouters_InitMaps(t *testing.T) {
	r := &HTTP{}
	r.addToRouters("READ_users", "/users/", "", "GET", nil)
	if r.RouterMap == nil {
		t.Error(test.DiffMessage(r.RouterMap, "non-nil", "RouterMap initialized"))
	}
	if r.PatternToFuncNameMap == nil {
		t.Error(test.DiffMessage(r.PatternToFuncNameMap, "non-nil", "PatternToFuncNameMap initialized"))
	}
	if r.FuncNameToPatternMap == nil {
		t.Error(test.DiffMessage(r.FuncNameToPatternMap, "non-nil", "FuncNameToPatternMap initialized"))
	}
}

func TestAddHandlerToRouterMap_StoresPattern(t *testing.T) {
	orig := InsertedRoutes
	InsertedRoutes = make(map[string]string)
	defer func() { InsertedRoutes = orig }()

	r := &HTTP{}
	r.AddHandlerToRouterMap(nil, "READ_items", nil)
	if len(r.RouterMap) != 1 {
		t.Error(test.DiffMessage(len(r.RouterMap), 1, "one route added"))
	}
	if InsertedRoutes["/items/||/[GET]/"] == "" {
		t.Error(test.DiffMessage("", "READ_items", "InsertedRoutes populated"))
	}
}

func TestGetConfigurations_MatchesInserted(t *testing.T) {
	orig := InsertedRoutes
	InsertedRoutes = make(map[string]string)
	defer func() { InsertedRoutes = orig }()

	r := &HTTP{}
	r.AddHandlerToRouterMap(nil, "READ_items", nil)
	r.AddHandlerToRouterMap(nil, "CREATE_items", nil)

	cfgs := r.GetConfigurations()
	if len(cfgs) != 2 {
		t.Error(test.DiffMessage(len(cfgs), 2, "two configurations"))
	}
	methods := map[string]bool{}
	for _, c := range cfgs {
		methods[c.Method] = true
	}
	if !methods["GET"] {
		t.Error(test.DiffMessage(methods, "GET present", "GET config"))
	}
	if !methods["POST"] {
		t.Error(test.DiffMessage(methods, "POST present", "POST config"))
	}
}

func TestParseFnNameToURL(t *testing.T) {
	testCases := make(map[string][]string)

	testCases["READ_members_BY_user_name_AND_member_id_OF_club_users_BY_id"] = []string{
		"GET",
		"/club_users/{id}/members/{user_name}/{member_id}/",
		"",
	}

	testCases["UPDATE_products_BY_productId_AND_productRanks_OF_categories_BY_categoryId_AND_categoryRank_OF_shops_BY_shopId_AND_shopRanks_VERSION_V_12"] = []string{
		"PUT",
		"/shops/{shopId}/{shopRanks}/categories/{categoryId}/{categoryRank}/products/{productId}/{productRanks}/",
		"V_12",
	}

	testCases["CREATE_owned_lists_OF_users_BY_id_VERSION_"] = []string{
		"POST",
		"/users/{id}/owned_lists/",
		"",
	}

	testCases["UPDATE_ANY_OF_members_OF_ANY_OF_users_BY_id_VERSION_NEUTRAL"] = []string{
		"PUT",
		"/users/{id}/*/members/*/",
		"NEUTRAL",
	}

	testCases["READ_ANY_HTML_FILE_OF_members_OF_ANY_JPEG_FILE_VERSION_112____3"] = []string{
		"GET",
		"/*.jpeg/members/*.html/",
		"112_3",
	}

	testCases["DELETE_image_PNG_FILE_OF_members_OF_users_BY_id_VERSION_NEUTRAL__"] = []string{
		"DELETE",
		"/users/{id}/members/image.png/",
		"NEUTRAL",
	}

	testCases["MODIFY_dm_events_OF_with_BY_participant_id_OF_dm_conversations_VERSION_V2"] = []string{
		"PATCH",
		"/dm_conversations/with/{participant_id}/dm_events/",
		"V2",
	}

	testCases["READ_me_ANY_bers_OF_us_ANY_ers_BY_id_VERSION_v1_v1"] = []string{
		"GET",
		"/us*ers/{id}/me*bers/",
		"v1_v1",
	}

	for fn, results := range testCases {
		method, route, version := ParseFuncNameToURL(fn)
		if method != results[0] {
			t.Error(test.DiffMessage(results[0], method, "method should be equal"))
		}

		if route != results[1] {
			t.Error(test.DiffMessage(results[1], route, "route should be equal"))
		}

		if version != results[2] {
			t.Error(test.DiffMessage(results[2], version, "version should be equal"))
		}
	}
}
