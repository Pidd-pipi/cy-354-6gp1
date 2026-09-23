export const REPORT_REASONS = [
  { value: 'false_description', label: '虚假描述', tag: 'warning' },
  { value: 'prohibited_item', label: '违禁物品', tag: 'danger' },
  { value: 'fraud', label: '疑似诈骗', tag: 'danger' },
  { value: 'other', label: '其他', tag: 'info' },
] as const

export const REPORT_STATUSES = [
  { value: 'pending', label: '待处理', type: 'warning' },
  { value: 'taken_down', label: '已下架', type: 'success' },
  { value: 'rejected', label: '已驳回', type: 'info' },
] as const

export const REPORT_ACTIONS = [
  { value: 'take_down', label: '下架商品' },
  { value: 'reject', label: '驳回举报' },
] as const

export function reportReasonLabel(value: string): string {
  return REPORT_REASONS.find((r) => r.value === value)?.label ?? value
}

export function reportReasonTag(value: string): string {
  return REPORT_REASONS.find((r) => r.value === value)?.tag ?? 'info'
}

export function reportStatusLabel(value: string): string {
  return REPORT_STATUSES.find((s) => s.value === value)?.label ?? value
}

export function reportStatusType(value: string): string {
  return REPORT_STATUSES.find((s) => s.value === value)?.type ?? 'info'
}
