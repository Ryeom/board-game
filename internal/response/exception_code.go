package response

const (
	ErrorCodeDefaultInternalServerError = "DEFAULT_INTERNAL_SERVER_ERROR"
)

// Authentication Error Codes
const (
	ErrorCodeAuthBind                      = "ERROR_AUTH_BIND"
	ErrorCodeAuthValidation                = "ERROR_AUTH_VALIDATION"
	ErrorCodeAuthEmailDuplicate            = "ERROR_AUTH_EMAIL_DUPLICATE"
	ErrorCodeAuthUserLookupFailed          = "ERROR_AUTH_USER_LOOKUP_FAILED"
	ErrorCodeAuthPasswordHashingFailed     = "ERROR_AUTH_PASSWORD_HASHING_FAILED"
	ErrorCodeAuthCreateUserFailed          = "ERROR_AUTH_CREATE_USER_FAILED"
	ErrorCodeAuthInvalidCredentials        = "ERROR_AUTH_INVALID_CREDENTIALS"
	ErrorCodeAuthJwtGenerationFailed       = "ERROR_AUTH_JWT_GENERATION_FAILED"
	ErrorCodeAuthInvalidToken              = "ERROR_AUTH_INVALID_TOKEN"
	ErrorCodeAuthLogoutFailed              = "ERROR_AUTH_LOGOUT_FAILED"
	ErrorCodeAuthTokenBlacklistCheckFailed = "ERROR_AUTH_TOKEN_BLACKLIST_CHECK_FAILED"
	ErrorCodeAuthTokenBlacklisted          = "ERROR_AUTH_TOKEN_BLACKLISTED"
	ErrorCodeAuthInvalidRequest            = "ERROR_AUTH_INVALID_REQUEST"
	ErrorCodeWSExpectedIdentify            = "ERROR_WS_EXPECTED_IDENTIFICATION"
	ErrorCodeWSInitialSessionSaveFailed    = "ERROR_WS_INITIAL_SESSION_SAVE_FAILED"
	ErrorCodeWSInvalidMessageFormat        = "ERROR_WS_INVALID_MESSAGE_FORMAT"
	ErrorCodeChatNotInRoom                 = "ERROR_CHAT_NOT_IN_ROOM"
	ErrorCodeChatEmptyMessage              = "ERROR_CHAT_EMPTY_MESSAGE"
	ErrorCodeChatSendFailed                = "ERROR_CHAT_SEND_FAILED"
	ErrorCodeChatHistoryFetchFailed        = "ERROR_CHAT_HISTORY_FAILURE"
)

// User Error Codes
const (
	ErrorCodeUserUnauthorized            = "ERROR_USER_UNAUTHORIZED"
	ErrorCodeUserProfileFetchFailed      = "ERROR_USER_PROFILE_FETCH_FAILED"
	ErrorCodeUserNotFound                = "ERROR_USER_NOT_FOUND"
	ErrorCodeUserProfileUpdateFailed     = "ERROR_USER_PROFILE_UPDATE_FAILED"
	ErrorCodeUserCurrentPasswordMismatch = "ERROR_USER_CURRENT_PASSWORD_MISMATCH"
	ErrorCodeUserPasswordChangeFailed    = "ERROR_USER_PASSWORD_CHANGE_FAILED"
)

// Room Error Codes
const (
	ErrorCodeRoomInvalidRequest        = "ERROR_ROOM_INVALID_REQUEST"
	ErrorCodeRoomNotFound              = "ERROR_ROOM_NOT_FOUND"
	ErrorCodeRoomUnsupportedGameMode   = "ERROR_ROOM_UNSUPPORTED_GAME_MODE"
	ErrorCodeRoomPasswordHashingFailed = "ERROR_ROOM_PASSWORD_HASHING_FAILED"
	ErrorCodeRoomNotInRoom             = "ERROR_ROOM_NOT_IN_ROOM"
	ErrorCodeRoomFull                  = "ERROR_ROOM_FULL"
	ErrorCodeRoomWrongPassword         = "ERROR_ROOM_WRONG_PASSWORD"
	ErrorCodeRoomJoinFailed            = "ERROR_ROOM_JOIN_FAILED"
	ErrorCodeRoomCreationFailed        = "ERROR_ROOM_CREATION_FAILED"
	ErrorCodeRoomDeleteFailed          = "ERROR_ROOM_DELETE_FAILED"
	ErrorCodeRoomAlreadyJoined         = "ERROR_ROOM_ALREADY_JOINED"
	ErrorCodeRoomLeaveFailed           = "ERROR_ROOM_LEAVE_FAILED"
	ErrorCodeRoomNotHost               = "ERROR_ROOM_NOT_HOST"
	ErrorCodeRoomUpdateFailed          = "ERROR_ROOM_UPDATE_FAILED"
	ErrorCodeRoomUserNotInRoom         = "ERROR_ROOM_USER_NOT_IN_ROOM"
	ErrorCodeRoomKickFailed            = "ERROR_ROOM_KICK_FAILED"
	ErrorCodeUserNoUpdates             = "ERROR_USER_NO_UPDATES"
	ErrorCodeUserInvalidRequest        = "ERROR_USER_INVALID_REQUEST"
)

const (
	ErrorCodeWSUnknownEvent = "ERROR_WS_UNKNOWN_EVENT"
)
