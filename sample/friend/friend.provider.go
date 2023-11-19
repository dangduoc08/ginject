package friend

import (
	"github.com/dangduoc08/gooh/common"
	"github.com/dangduoc08/gooh/core"
	"github.com/dangduoc08/gooh/modules/config"
)

type Friends struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type FriendProvider struct {
	ConfigService config.ConfigService
	Logger        common.Logger
	Friends       map[int][]Friends
}

func (friendProvider FriendProvider) NewProvider() core.Provider {
	friendProvider.Friends = map[int][]Friends{
		0: {
			{
				ID:   1,
				Name: "John",
			},
			{
				ID:   2,
				Name: "Jane",
			},
			{
				ID:   3,
				Name: "Nick",
			},
		},
		1: {
			{
				ID:   0,
				Name: "Mike",
			},
			{
				ID:   2,
				Name: "Jane",
			},
			{
				ID:   4,
				Name: "Jenifer",
			},
		},
		2: {
			{
				ID:   0,
				Name: "Mike",
			},
			{
				ID:   4,
				Name: "Jenifer",
			},
		},
	}

	return friendProvider
}

func (friendProvider FriendProvider) GetFriendsByUserID(
	userID int,
) any {
	if friends, ok := friendProvider.Friends[userID]; ok {
		return friends
	}
	return []any{}
}
