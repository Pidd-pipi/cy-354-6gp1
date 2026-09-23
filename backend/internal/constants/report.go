package constants

// ReportReason defines product report reason enum values shared with the frontend.
const (
	ReportReasonFalseDescription = "false_description"
	ReportReasonProhibitedItem   = "prohibited_item"
	ReportReasonFraud            = "fraud"
	ReportReasonOther            = "other"
)

// ReportReasons lists all valid product report reasons.
var ReportReasons = []string{
	ReportReasonFalseDescription, ReportReasonProhibitedItem, ReportReasonFraud, ReportReasonOther,
}

// IsReportReason reports whether the given reason is valid.
func IsReportReason(r string) bool {
	for _, v := range ReportReasons {
		if v == r {
			return true
		}
	}
	return false
}

// ReportReasonText returns the Chinese label of a report reason.
func ReportReasonText(r string) string {
	switch r {
	case ReportReasonFalseDescription:
		return "虚假描述"
	case ReportReasonProhibitedItem:
		return "违禁物品"
	case ReportReasonFraud:
		return "疑似诈骗"
	case ReportReasonOther:
		return "其他"
	default:
		return "未知"
	}
}

// ReportStatus defines report workflow states shared with the frontend.
const (
	ReportStatusPending   = "pending"
	ReportStatusTakenDown = "taken_down"
	ReportStatusRejected  = "rejected"
)

// ReportStatuses lists all valid report statuses.
var ReportStatuses = []string{
	ReportStatusPending, ReportStatusTakenDown, ReportStatusRejected,
}

// IsReportStatus reports whether the given status is valid.
func IsReportStatus(s string) bool {
	for _, v := range ReportStatuses {
		if v == s {
			return true
		}
	}
	return false
}

// ReportStatusText returns the Chinese label of a report status.
func ReportStatusText(s string) string {
	switch s {
	case ReportStatusPending:
		return "待处理"
	case ReportStatusTakenDown:
		return "已下架"
	case ReportStatusRejected:
		return "已驳回"
	default:
		return "未知"
	}
}

// ReportAction defines admin handling actions shared with the frontend.
const (
	ReportActionTakeDown = "take_down"
	ReportActionReject   = "reject"
)

// ReportActions lists all valid admin report actions.
var ReportActions = []string{ReportActionTakeDown, ReportActionReject}

// IsReportAction reports whether the given action is valid.
func IsReportAction(a string) bool {
	for _, v := range ReportActions {
		if v == a {
			return true
		}
	}
	return false
}

// ReportActionText returns the Chinese label of an admin report action.
func ReportActionText(a string) string {
	switch a {
	case ReportActionTakeDown:
		return "下架商品"
	case ReportActionReject:
		return "驳回举报"
	default:
		return "未知"
	}
}
