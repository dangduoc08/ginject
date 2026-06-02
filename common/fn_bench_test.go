package common

import (
	"testing"
)

func BenchmarkGetFnName(b *testing.B) {
	handler := fnTestController{}.READ_users
	for range b.N {
		GetFuncName(handler)
	}
}

func BenchmarkParseFnNameToURL_Simple(b *testing.B) {
	for range b.N {
		ParseFuncNameToURL("READ_users")
	}
}

func BenchmarkParseFnNameToURL_Complex(b *testing.B) {
	for range b.N {
		ParseFuncNameToURL(
			"UPDATE_products_BY_productId_AND_productRanks_OF_categories_BY_categoryId_AND_categoryRank_OF_shops_BY_shopId_AND_shopRanks_VERSION_V_12",
		)
	}
}
