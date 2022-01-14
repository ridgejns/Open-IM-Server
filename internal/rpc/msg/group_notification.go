package msg

import (
	"Open_IM/pkg/common/config"
	"Open_IM/pkg/common/constant"
	imdb "Open_IM/pkg/common/db/mysql_model/im_mysql_model"
	"Open_IM/pkg/common/log"
	"Open_IM/pkg/common/token_verify"
	utils2 "Open_IM/pkg/common/utils"
	pbGroup "Open_IM/pkg/proto/group"
	open_im_sdk "Open_IM/pkg/proto/sdk_ws"
	"Open_IM/pkg/utils"
	"encoding/json"
)

//message GroupCreatedTips{
//  GroupInfo Group = 1;
//  GroupMemberFullInfo Creator = 2;
//  repeated GroupMemberFullInfo MemberList = 3;
//  uint64 OperationTime = 4;
//} creator->group

func setOpUserInfo(opUserID, groupID string, groupMemberInfo *open_im_sdk.GroupMemberFullInfo) error {
	if token_verify.IsMangerUserID(opUserID) {
		u, err := imdb.GetUserByUserID(opUserID)
		if err != nil {
			return utils.Wrap(err, "GetUserByUserID failed")
		}
		utils.CopyStructFields(groupMemberInfo, u)
		groupMemberInfo.GroupID = groupID
	} else {
		u, err := imdb.GetGroupMemberInfoByGroupIDAndUserID(groupID, opUserID)
		if err != nil {
			return utils.Wrap(err, "GetGroupMemberInfoByGroupIDAndUserID failed")
		}
		if err = utils2.GroupMemberDBCopyOpenIM(groupMemberInfo, u); err != nil {
			return utils.Wrap(err, "")
		}
	}
	return nil
}

func setGroupInfo(groupID string, groupInfo *open_im_sdk.GroupInfo) error {
	group, err := imdb.GetGroupInfoByGroupID(groupID)
	if err != nil {
		return utils.Wrap(err, "GetGroupInfoByGroupID failed")
	}
	err = utils2.GroupDBCopyOpenIM(groupInfo, group)
	if err != nil {
		return utils.Wrap(err, "GetGroupMemberNumByGroupID failed")
	}
	return nil
}

func setGroupMemberInfo(groupID, userID string, groupMemberInfo *open_im_sdk.GroupMemberFullInfo) error {
	groupMember, err := imdb.GetGroupMemberInfoByGroupIDAndUserID(groupID, userID)
	if err != nil {
		return utils.Wrap(err, "")
	}
	if err = utils2.GroupMemberDBCopyOpenIM(groupMemberInfo, groupMember); err != nil {
		return utils.Wrap(err, "")
	}
	return nil
}

//func setGroupPublicUserInfo(operationID, groupID, userID string, publicUserInfo *open_im_sdk.PublicUserInfo) {
//	group, err := imdb.GetGroupMemberInfoByGroupIDAndUserID(groupID, userID)
//	if err != nil {
//		log.NewError(operationID, "FindGroupMemberInfoByGroupIdAndUserId failed ", err.Error(), groupID, userID)
//		return
//	}
//	utils.CopyStructFields(publicUserInfo, group)
//}

//创建群后调用
func GroupCreatedNotification(operationID, opUserID, groupID string, initMemberList []string) {
	GroupCreatedTips := open_im_sdk.GroupCreatedTips{Group: &open_im_sdk.GroupInfo{},
		Creator: &open_im_sdk.GroupMemberFullInfo{}}
	if err := setOpUserInfo(GroupCreatedTips.Creator.UserID, groupID, GroupCreatedTips.Creator); err != nil {
		log.NewError(operationID, "setOpUserInfo failed ", err.Error(), GroupCreatedTips.Creator.UserID, groupID, GroupCreatedTips.Creator)
		return
	}
	err := setGroupInfo(groupID, GroupCreatedTips.Group)
	if err != nil {
		log.NewError(operationID, "setGroupInfo failed ", groupID, GroupCreatedTips.Group)
		return
	}
	for _, v := range initMemberList {
		var groupMemberInfo open_im_sdk.GroupMemberFullInfo
		setGroupMemberInfo(groupID, v, &groupMemberInfo)
		GroupCreatedTips.MemberList = append(GroupCreatedTips.MemberList, &groupMemberInfo)
	}
	var tips open_im_sdk.TipsComm
	tips.Detail, _ = json.Marshal(GroupCreatedTips)
	tips.DefaultTips = config.Config.Notification.GroupCreated.DefaultTips.Tips
	var n NotificationMsg
	n.SendID = opUserID
	n.RecvID = groupID
	n.ContentType = constant.GroupCreatedNotification
	n.SessionType = constant.GroupChatType
	n.MsgFrom = constant.SysMsgType
	n.OperationID = operationID
	n.Content, _ = json.Marshal(tips)
	log.NewInfo(operationID, "Notification ", n)
	Notification(&n)
}

//message ReceiveJoinApplicationTips{
//  GroupInfo Group = 1;
//  PublicUserInfo Applicant  = 2;
//  string 	Reason = 3;
//}  apply->all managers GroupID              string   `protobuf:"bytes,1,opt,name=GroupID" json:"GroupID,omitempty"`
//	ReqMessage           string   `protobuf:"bytes,2,opt,name=ReqMessage" json:"ReqMessage,omitempty"`
//	OpUserID             string   `protobuf:"bytes,3,opt,name=OpUserID" json:"OpUserID,omitempty"`
//	OperationID          string   `protobuf:"bytes,4,opt,name=OperationID" json:"OperationID,omitempty"`
//申请进群后调用
func JoinApplicationNotification(req *pbGroup.JoinGroupReq) {
	JoinGroupApplicationTips := open_im_sdk.JoinGroupApplicationTips{Group: &open_im_sdk.GroupInfo{}, Applicant: &open_im_sdk.PublicUserInfo{}}
	err := setGroupInfo(req.GroupID, JoinGroupApplicationTips.Group)
	if err != nil {
		log.NewError(req.OperationID, "setGroupInfo failed ", err.Error(), req.GroupID, JoinGroupApplicationTips.Group)
		return
	}

	apply, err := imdb.GetUserByUserID(req.OpUserID)
	if err != nil {
		log.NewError(req.OperationID, "FindUserByUID failed ", err.Error(), req.OpUserID)
		return
	}
	utils.CopyStructFields(JoinGroupApplicationTips.Applicant, apply)
	JoinGroupApplicationTips.Reason = req.ReqMessage

	var tips open_im_sdk.TipsComm
	tips.Detail, _ = json.Marshal(JoinGroupApplicationTips)
	tips.DefaultTips = "JoinGroupApplicationTips"
	var n NotificationMsg
	n.SendID = req.OpUserID
	n.ContentType = constant.JoinApplicationNotification
	n.SessionType = constant.SingleChatType
	n.MsgFrom = constant.SysMsgType
	n.OperationID = req.OperationID
	n.Content, _ = json.Marshal(tips)
	managerList, err := imdb.GetOwnerManagerByGroupID(req.GroupID)
	if err != nil {
		log.NewError(req.OperationID, "GetOwnerManagerByGroupId failed ", err.Error(), req.GroupID)
		return
	}
	for _, v := range managerList {
		n.RecvID = v.UserID
		log.NewInfo(req.OperationID, "Notification ", n)
		Notification(&n)
	}
}

//message ApplicationProcessedTips{
//  GroupInfo Group = 1;
//  GroupMemberFullInfo OpUser = 2;
//  int32 Result = 3;
//  string 	Reason = 4;
//}
//处理进群请求后调用
func ApplicationProcessedNotification(req *pbGroup.GroupApplicationResponseReq) {
	ApplicationProcessedTips := open_im_sdk.ApplicationProcessedTips{Group: &open_im_sdk.GroupInfo{}, OpUser: &open_im_sdk.GroupMemberFullInfo{}}
	if err := setGroupInfo(req.GroupID, ApplicationProcessedTips.Group); err != nil {
		log.NewError(req.OperationID, "setGroupInfo failed ", err.Error(), req.GroupID, ApplicationProcessedTips.Group)
		return
	}
	if err := setOpUserInfo(req.OpUserID, req.GroupID, ApplicationProcessedTips.OpUser); err != nil {
		log.Error(req.OperationID, "setOpUserInfo failed", req.OpUserID, req.GroupID, ApplicationProcessedTips.OpUser)
		return
	}
	ApplicationProcessedTips.Reason = req.HandledMsg
	ApplicationProcessedTips.Result = req.HandleResult

	var tips open_im_sdk.TipsComm
	tips.Detail, _ = json.Marshal(ApplicationProcessedTips)
	tips.DefaultTips = "ApplicationProcessedNotification"
	var n NotificationMsg
	n.SendID = req.OpUserID
	n.ContentType = constant.ApplicationProcessedNotification
	n.SessionType = constant.SingleChatType
	n.MsgFrom = constant.SysMsgType
	n.OperationID = req.OperationID
	n.RecvID = req.FromUserID
	n.Content, _ = json.Marshal(tips)
	Notification(&n)
}

//message MemberInvitedTips{
//  GroupInfo Group = 1;
//  GroupMemberFullInfo OpUser = 2;
//  GroupMemberFullInfo InvitedUser = 3;
//  uint64 OperationTime = 4;
//}
//被邀请进群后调用
func MemberInvitedNotification(operationID, groupID, opUserID, reason string, invitedUserIDList []string) {
	ApplicationProcessedTips := open_im_sdk.MemberInvitedTips{Group: &open_im_sdk.GroupInfo{}, OpUser: &open_im_sdk.GroupMemberFullInfo{}}
	if err := setGroupInfo(groupID, ApplicationProcessedTips.Group); err != nil {
		log.Error(operationID, "setGroupInfo failed ", err.Error(), groupID, ApplicationProcessedTips.Group)
		return
	}
	if err := setOpUserInfo(opUserID, groupID, ApplicationProcessedTips.OpUser); err != nil {
		log.Error(operationID, "setOpUserInfo failed ", err.Error(), opUserID, groupID, ApplicationProcessedTips.OpUser)
		return
	}
	for _, v := range invitedUserIDList {
		var groupMemberInfo open_im_sdk.GroupMemberFullInfo
		if err := setGroupMemberInfo(groupID, v, &groupMemberInfo); err != nil {
			log.Error(operationID, "setGroupMemberInfo faield ", err.Error(), groupID)
			continue
		}
		ApplicationProcessedTips.InvitedUserList = append(ApplicationProcessedTips.InvitedUserList, &groupMemberInfo)
	}
	var tips open_im_sdk.TipsComm
	tips.Detail, _ = json.Marshal(ApplicationProcessedTips)
	tips.DefaultTips = "MemberInvitedNotification"
	var n NotificationMsg
	n.SendID = opUserID
	n.ContentType = constant.MemberInvitedNotification
	n.SessionType = constant.GroupChatType
	n.MsgFrom = constant.SysMsgType
	n.OperationID = operationID
	n.Content, _ = json.Marshal(tips)
	n.RecvID = groupID
	Notification(&n)
}

//message MemberKickedTips{
//  GroupInfo Group = 1;
//  GroupMemberFullInfo OpUser = 2;
//  GroupMemberFullInfo KickedUser = 3;
//  uint64 OperationTime = 4;
//}
//被踢后调用
func MemberKickedNotification(req *pbGroup.KickGroupMemberReq, kickedUserIDList []string) {
	MemberKickedTips := open_im_sdk.MemberKickedTips{Group: &open_im_sdk.GroupInfo{}, OpUser: &open_im_sdk.GroupMemberFullInfo{}}
	if err := setGroupInfo(req.GroupID, MemberKickedTips.Group); err != nil {
		log.Error(req.OperationID, "setGroupInfo failed ", err.Error(), req.GroupID, MemberKickedTips.Group)
		return
	}
	if err := setOpUserInfo(req.OpUserID, req.GroupID, MemberKickedTips.OpUser); err != nil {
		log.Error(req.OperationID, "setOpUserInfo failed ", err.Error(), req.OpUserID, req.GroupID, MemberKickedTips.OpUser)
		return
	}
	for _, v := range kickedUserIDList {
		var groupMemberInfo open_im_sdk.GroupMemberFullInfo
		if err := setGroupMemberInfo(req.GroupID, v, &groupMemberInfo); err != nil {
			log.Error(req.OperationID, "setGroupMemberInfo failed ", err.Error(), req.GroupID, v)
			continue
		}
		MemberKickedTips.KickedUserList = append(MemberKickedTips.KickedUserList, &groupMemberInfo)
	}

	var tips open_im_sdk.TipsComm
	tips.Detail, _ = json.Marshal(MemberKickedTips)
	tips.DefaultTips = "MemberKickedNotification"
	var n NotificationMsg
	n.SendID = req.OpUserID
	n.ContentType = constant.MemberKickedNotification
	n.SessionType = constant.GroupChatType
	n.MsgFrom = constant.SysMsgType
	n.OperationID = req.OperationID
	n.Content, _ = json.Marshal(tips)
	n.RecvID = req.GroupID
	Notification(&n)

	for _, v := range kickedUserIDList {
		m := n
		m.SessionType = constant.SingleChatType
		m.RecvID = v
		Notification(&m)
	}
}

//message GroupInfoChangedTips{
//  int32 ChangedType = 1; //bitwise operators: 1:groupName; 10:Notification  100:Introduction; 1000:FaceUrl
//  GroupInfo Group = 2;
//  GroupMemberFullInfo OpUser = 3;
//}

//群信息改变后掉用
func GroupInfoChangedNotification(operationID, opUserID, groupID string, changedType int32) {
	GroupInfoChangedTips := open_im_sdk.GroupInfoChangedTips{Group: &open_im_sdk.GroupInfo{}, OpUser: &open_im_sdk.GroupMemberFullInfo{}}
	if err := setGroupInfo(groupID, GroupInfoChangedTips.Group); err != nil {
		log.Error(operationID, "setGroupInfo failed ", err.Error(), groupID, GroupInfoChangedTips.Group)
		return
	}
	if err := setOpUserInfo(opUserID, groupID, GroupInfoChangedTips.OpUser); err != nil {
		log.Error(operationID, "setOpUserInfo failed ", err.Error(), opUserID, groupID, GroupInfoChangedTips.OpUser)
		return
	}
	GroupInfoChangedTips.ChangedType = changedType
	var tips open_im_sdk.TipsComm
	tips.Detail, _ = json.Marshal(GroupInfoChangedTips)
	tips.DefaultTips = "GroupInfoChangedNotification"
	var n NotificationMsg
	n.SendID = opUserID
	n.ContentType = constant.GroupInfoChangedNotification
	n.SessionType = constant.GroupChatType
	n.MsgFrom = constant.SysMsgType
	n.OperationID = operationID
	n.Content, _ = json.Marshal(tips)
	n.RecvID = groupID
	Notification(&n)
}

//message MemberLeaveTips{
//  GroupInfo Group = 1;
//  GroupMemberFullInfo LeaverUser = 2;
//  uint64 OperationTime = 3;
//}

//群成员退群后调用
func MemberLeaveNotification(req *pbGroup.QuitGroupReq) {
	MemberLeaveTips := open_im_sdk.MemberLeaveTips{Group: &open_im_sdk.GroupInfo{}, LeaverUser: &open_im_sdk.GroupMemberFullInfo{}}
	if err := setGroupInfo(req.GroupID, MemberLeaveTips.Group); err != nil {
		log.Error(req.OperationID, "setGroupInfo failed ", err.Error(), req.GroupID, MemberLeaveTips.Group)
		return
	}
	if err := setOpUserInfo(req.OpUserID, req.GroupID, MemberLeaveTips.LeaverUser); err != nil {
		log.Error(req.OperationID, "setOpUserInfo failed ", err.Error(), req.OpUserID, req.GroupID, MemberLeaveTips.LeaverUser)
		return
	}

	var tips open_im_sdk.TipsComm
	tips.Detail, _ = json.Marshal(MemberLeaveTips)
	tips.DefaultTips = "MemberLeaveNotification"
	var n NotificationMsg
	n.SendID = req.OpUserID
	n.ContentType = constant.MemberLeaveNotification
	n.SessionType = constant.GroupChatType
	n.MsgFrom = constant.SysMsgType
	n.OperationID = req.OperationID
	n.Content, _ = json.Marshal(tips)
	n.RecvID = req.GroupID
	Notification(&n)

	m := n
	n.SessionType = constant.SingleChatType
	n.RecvID = req.OpUserID
	Notification(&m)
}

//message MemberEnterTips{
//  GroupInfo Group = 1;
//  GroupMemberFullInfo EntrantUser = 2;
//  uint64 OperationTime = 3;
//}
//群成员主动申请进群，管理员同意后调用，
func MemberEnterNotification(req *pbGroup.GroupApplicationResponseReq) {
	MemberLeaveTips := open_im_sdk.MemberEnterTips{Group: &open_im_sdk.GroupInfo{}, EntrantUser: &open_im_sdk.GroupMemberFullInfo{}}
	if err := setGroupInfo(req.GroupID, MemberLeaveTips.Group); err != nil {
		log.Error(req.OperationID, "setGroupInfo failed ", err.Error(), req.GroupID, MemberLeaveTips.Group)
		return
	}
	if err := setOpUserInfo(req.OpUserID, req.GroupID, MemberLeaveTips.EntrantUser); err != nil {
		log.Error(req.OperationID, "setOpUserInfo failed ", err.Error(), req.OpUserID, req.GroupID, MemberLeaveTips.EntrantUser)
		return
	}
	var tips open_im_sdk.TipsComm
	tips.Detail, _ = json.Marshal(MemberLeaveTips)
	tips.DefaultTips = "MemberEnterNotification"
	var n NotificationMsg
	n.SendID = req.OpUserID
	n.ContentType = constant.MemberEnterNotification
	n.SessionType = constant.GroupChatType
	n.MsgFrom = constant.SysMsgType
	n.OperationID = req.OperationID
	n.Content, _ = json.Marshal(tips)
	n.RecvID = req.GroupID
	Notification(&n)
}