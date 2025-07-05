package response

const (
	SuccessCodeSystemOK           = "SUCCESS_SYSTEM_OK"
	SuccessCodeUserIdentify       = "SUCCESS_USER_IDENTIFY"
	SuccessCodeUserUpdate         = "SUCCESS_USER_UPDATE"
	SuccessCodeUserStatusFetch    = "SUCCESS_USER_STATUS_FETCH"
	SuccessCodeUserSignUp         = "SUCCESS_USER_SIGNUP"
	SuccessCodeUserLogin          = "SUCCESS_USER_LOGIN"
	SuccessCodeUserLogout         = "SUCCESS_USER_LOGOUT"
	SuccessCodeUserPasswordChange = "SUCCESS_USER_PASSWORD_CHANGE"
	SuccessCodeUserProfileFetch   = "SUCCESS_USER_PROFILE_FETCH"
	SuccessCodeUserProfileUpdate  = "SUCCESS_USER_PROFILE_UPDATE"
	SuccessCodeUserProfileGet     = "SUCCESS_USER_PROFILE_GET"

	SuccessCodeRoomCreate    = "SUCCESS_ROOM_CREATE"
	SuccessCodeRoomJoin      = "SUCCESS_ROOM_JOIN"
	SuccessCodeRoomLeave     = "SUCCESS_ROOM_LEAVE"
	SuccessCodeRoomListFetch = "SUCCESS_ROOM_LIST_FETCH"
	SuccessCodeRoomUpdate    = "SUCCESS_ROOM_UPDATE"
	SuccessCodeRoomKick      = "SUCCESS_ROOM_KICK"
	SuccessCodeRoomDelete    = "SUCCESS_ROOM_DELETE"
	SuccessCodeRoomNoChanges = "SUCCESS_ROOM_NO_CHANGES" // 변경 사항 없을 때

	SuccessCodeChatSend         = "SUCCESS_CHAT_SEND"
	SuccessCodeChatHistoryFetch = "SUCCESS_CHAT_HISTORY_FETCH"

	SuccessCodeSystemErrorReceived = "SUCCESS_SYSTEM_ERROR_RECEIVED"

	SuccessCodeGameStart  = "SUCCESS_GAME_START"
	SuccessCodeGameEnd    = "SUCCESS_GAME_END"
	SuccessCodeGameAction = "SUCCESS_GAME_ACTION"
	SuccessCodeGameSync   = "SUCCESS_GAME_SYNC"
)
