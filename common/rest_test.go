package common

import (
	"testing"

	"github.com/dangduoc08/gooh/utils"
)

func TestParseFnNameToURL(t *testing.T) {
	testCases := make(map[string][]string)

	testCases["READ_members_BY_user_name_AND_member_id_OF_club_users_BY_id"] = []string{
		"GET",
		"/club_users/{id}/members/{user_name}/{member_id}/",
	}

	testCases["UPDATE_products_BY_productId_AND_productRanks_OF_categories_BY_categoryId_AND_categoryRank_OF_shops_BY_shopId_AND_shopRanks"] = []string{
		"PUT",
		"/shops/{shopId}/{shopRanks}/categories/{categoryId}/{categoryRank}/products/{productId}/{productRanks}/",
	}

	testCases["CREATE_owned_lists_OF_users_BY_id"] = []string{
		"POST",
		"/users/{id}/owned_lists/",
	}

	testCases["UPDATE_ANY_OF_members_OF_ANY_OF_users_BY_id"] = []string{
		"PUT",
		"/users/{id}/*/members/*/",
	}

	testCases["READ_ANY_HTML_FILE_OF_members_OF_ANY_JPEG_FILE"] = []string{
		"GET",
		"/*.jpeg/members/*.html/",
	}

	testCases["DELETE_image_PNG_FILE_OF_members_OF_users_BY_id"] = []string{
		"DELETE",
		"/users/{id}/members/image.png/",
	}

	testCases["MODIFY_dm_events_OF_with_BY_participant_id_OF_dm_conversations"] = []string{
		"PATCH",
		"/dm_conversations/with/{participant_id}/dm_events/",
	}

	testCases["READ_me_ANY_bers_OF_us_ANY_ers_BY_id"] = []string{
		"GET",
		"/us*ers/{id}/me*bers/",
	}

	for fn, results := range testCases {
		method, route := ParseFnNameToURL(fn, RESTOperations)
		if method != results[0] {
			t.Errorf(utils.ErrorMessage(results[0], method, "method should be equal"))
		}

		if route != results[1] {
			t.Errorf(utils.ErrorMessage(results[1], route, "route should be equal"))
		}
	}
}
