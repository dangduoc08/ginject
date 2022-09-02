package routing

import (
	"fmt"
	"os"
	"testing"

	"github.com/dangduoc08/gooh/context"
)

func TestVisualize(test *testing.T) {
	if os.Getenv("v") == "true" {
		routerInstance := NewRouter()
		for _, route := range []string{
			"/v1/users/get/jobs/get",
			"/v2/users/{userId}/*/jobs/{jobId}/get",
			"/v1/users/{userId}/*/jobs/{jobId}/delete",
			"/v2/users/{userId}/*/jobs/{jobId}/*",
		} {
			routerInstance.add(route, func(ctx *context.Context) {})
		}
		jsonStr, err := routerInstance.visualize()
		if err != nil {
			fmt.Printf("Error: %s", err.Error())
		} else {
			fmt.Println(string(jsonStr))
		}
	} else {
		test.Skip()
	}
}
