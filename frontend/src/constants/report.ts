export const REPORT_REASONS = [
  { value: 'fake_description', label: '虚假描述' },
  { value: 'prohibited_item', label: '违禁物品' },
] as const

export const REPORT_STATUSES = [
  { value: 'pending', label: '待处理', type: 'warning' },
  { value: 'taken_down', label: '已下架', type: 'danger' },
  { value: 'rejected', label: '已驳回', type: 'info' },
] as const

export const REPORT_ACTIONS = [
  { value: 'take_down', label: '下架商品', type: 'danger' },
  { value: 'reject', label: '驳回举报', type: 'info' },
] as const

export function reportReasonLabel(value: string): string {
  return REPORT_REASONS.find((r) => r.value === value)?.label ?? value
}

export function reportStatusLabel(value: string): string {
  return REPORT_STATUSES.find((s) => s.value === value)?.label ?? value
}

export function reportStatusType(value: string): string {
  return REPORT_STATUSES.find((s) => s.value === value)?.type ?? 'info'
}
