import request from '../utils/request'
import type { ProductReport } from '../types'

export interface CreateProductReportPayload {
  product_id: number
  reason: string
  description?: string
}

export interface HandleProductReportPayload {
  action: 'take_down' | 'reject'
  remark?: string
}

export interface ReportResponse {
  code: number
  message: string
  data: ProductReport
}

export function createProductReport(data: CreateProductReportPayload) {
  return request.post<never, ReportResponse>('/reports/products', data)
}

export function listMyProductReports() {
  return request.get<never, { code: number; message: string; data: ProductReport[] }>('/reports/products/me')
}

export function listPendingProductReports() {
  return request.get<never, { code: number; message: string; data: ProductReport[] }>('/admin/reports/products')
}

export function handleProductReport(id: number, data: HandleProductReportPayload) {
  return request.post<never, ReportResponse>(`/admin/reports/products/${id}/handle`, data)
}
