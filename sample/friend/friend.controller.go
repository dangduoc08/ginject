package friend

import (
	"github.com/dangduoc08/gooh/common"
	"github.com/dangduoc08/gooh/core"
)

type FriendController struct {
	common.Rest
	FriendProvider FriendProvider
	Logger         common.Logger
}

func (friendController FriendController) NewController() core.Controller {
	friendController.Prefix("v1")

	return friendController
}

func (friendController FriendController) READ_friends_BY_userID(
	readFriendsParamDTO ReadFriendsParamDTO,
) any {
	return friendController.FriendProvider.GetFriendsByUserID(readFriendsParamDTO.UserID)
}
