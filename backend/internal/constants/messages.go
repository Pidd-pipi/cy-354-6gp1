package constants

// Messages centralizes user-facing prompts, backend responses and log wording.
const (
	MsgOK                   = "ok"
	MsgValidationFailed     = "参数校验失败"
	MsgUnauthorized         = "未登录或登录已过期"
	MsgForbidden            = "没有操作权限"
	MsgNotFound             = "资源不存在"
	MsgConflict             = "资源状态冲突"
	MsgRateLimited          = "请求过于频繁，请稍后再试"
	MsgInternalError        = "服务器内部错误"
	MsgPhoneAlreadyUsed     = "该手机号已被注册"
	MsgPhoneOrPassword      = "手机号或密码错误"
	MsgProductNotOnSale     = "商品当前不在售"
	MsgProductSold          = "商品已售出"
	MsgTradeStatusInvalid   = "当前交易状态不可操作"
	MsgNotParticipant       = "仅买家和卖家可操作该订单"
	MsgAlreadyReviewed      = "该交易已评价"
	MsgExchangeClosed       = "该交换已关闭"
	MsgNoMatch              = "暂未找到匹配的书籍交换"
	MsgReportTarget         = "举报对象无效"
	MsgReportSelf           = "不能举报自己发布的商品"
	MsgReportReasonInvalid  = "举报原因不合法"
	MsgReportActionInvalid  = "举报处理动作不合法"
	MsgReportRemarkRequired = "驳回举报必须填写原因"
	MsgReportHandled        = "该举报已被处理"
	MsgReportProductGone    = "商品已售出或已下架，无法下架处理"
)
