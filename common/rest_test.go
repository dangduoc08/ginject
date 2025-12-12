package common

import (
	"testing"

	"github.com/dangduoc08/ginject/utils"
)

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
		method, route, version := ParseFnNameToURL(fn, RESTOperations)
		if method != results[0] {
			t.Error(utils.ErrorMessage(results[0], method, "method should be equal"))
		}

		if route != results[1] {
			t.Error(utils.ErrorMessage(results[1], route, "route should be equal"))
		}

		if version != results[2] {
			t.Error(utils.ErrorMessage(results[2], version, "version should be equal"))
		}
	}
}
