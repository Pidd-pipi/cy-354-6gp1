import request from '../utils/request'
import type { PageResult, Report } from '../types'

export function createReport(data: { product_id: number; reason: string; description?: string }) {
  return request.post<never, { code: number; message: string; data: Report }>('/reports', data)
}

export function listMyReports() {
  return request.get<never, { code: number; message: string; data: Report[] }>('/reports/me')
}

export function listReports(params: { status?: string; page?: number; page_size?: number }) {
  return request.get<never, { code: number; message: string; data: PageResult<Report> }>('/admin/reports', { params })
}

export function handleReport(id: number, data: { action: string; remark?: string }) {
  return request.post<never, { code: number; message: string; data: Report }>(`/admin/reports/${id}/handle`, data)
}
