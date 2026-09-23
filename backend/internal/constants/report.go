package constants

// ReportReason defines why a student reports an on-sale product.
const (
	ReportReasonFakeDescription = "fake_description" // 虚假描述
	ReportReasonProhibitedItem  = "prohibited_item"  // 违禁物品
)

// ReportReasons lists all valid report reasons shared with the frontend.
var ReportReasons = []string{
	ReportReasonFakeDescription, ReportReasonProhibitedItem,
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
	case ReportReasonFakeDescription:
		return "虚假描述"
	case ReportReasonProhibitedItem:
		return "违禁物品"
	default:
		return "未知"
	}
}

// ReportStatus defines report lifecycle states shared with the frontend.
const (
	ReportStatusPending   = "pending"    // 待处理
	ReportStatusTakenDown = "taken_down" // 已下架（举报成立）
	ReportStatusRejected  = "rejected"   // 已驳回（举报不成立）
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

// ReportAction defines the two ways an admin handles a pending report.
const (
	ReportActionTakeDown = "take_down" // 下架商品，举报成立
	ReportActionReject   = "reject"    // 保留商品，举报驳回
)

// ReportActions lists all valid admin handling actions.
var ReportActions = []string{
	ReportActionTakeDown, ReportActionReject,
}

// IsReportAction reports whether the given action is valid.
func IsReportAction(a string) bool {
	for _, v := range ReportActions {
		if v == a {
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

// ReportActionText returns the Chinese label of an admin handling action.
func ReportActionText(a string) string {
	switch a {
	case ReportActionTakeDown:
		return "下架"
	case ReportActionReject:
		return "驳回"
	default:
		return "未知"
	}
}
