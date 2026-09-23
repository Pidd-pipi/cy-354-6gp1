<template>
  <div class="page">
    <h2>🛡️ 举报处理</h2>
    <el-alert title="仅展示待处理举报。下架会同时将商品置为已下架并反馈结果；商品已售出/已下架或已被其他管理员处理时，操作会被拒绝。"
      type="info" :closable="false" class="banner" />
    <el-table :data="reports" v-loading="loading" empty-text="暂无待处理举报">
      <el-table-column prop="id" label="ID" width="70" />
      <el-table-column label="商品" min-width="180">
        <template #default="{ row }">
          <div>{{ row.product_title || '商品已删除' }}</div>
          <div class="sub">#{{ row.product_id }} · 当前状态：{{ productStatusLabel(row.product_status) }}</div>
        </template>
      </el-table-column>
      <el-table-column label="举报人" width="120">
        <template #default="{ row }">{{ row.reporter_name }}（#{{ row.reporter_id }}）</template>
      </el-table-column>
      <el-table-column label="原因" width="100">
        <template #default="{ row }">
          <el-tag size="small" :type="reportReasonTag(row.reason) as any">{{ reportReasonLabel(row.reason) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="description" label="说明" min-width="160" />
      <el-table-column label="提交时间" width="150">
        <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="180" fixed="right">
        <template #default="{ row }">
          <el-button size="small" type="danger" @click="takeDown(row)">下架</el-button>
          <el-button size="small" @click="openReject(row)">驳回</el-button>
        </template>
      </el-table-column>
    </el-table>
    <el-dialog v-model="rejectVisible" title="驳回举报" width="440px">
      <el-alert title="驳回后商品保持在售，请填写驳回原因。" type="warning" :closable="false" class="banner" />
      <el-input v-model="rejectRemark" type="textarea" :rows="4" maxlength="500" show-word-limit
        placeholder="请填写驳回原因（学生可在个人中心查看）" />
      <template #footer>
        <el-button @click="rejectVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="confirmReject">确认驳回</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { listPendingProductReports, handleProductReport } from '../api/productReport'
import { reportReasonLabel, reportReasonTag } from '../constants/report'
import { productStatusLabel } from '../constants/product'
import { formatDateTime } from '../utils/dateFormat'
import type { ProductReport } from '../types'

const reports = ref<ProductReport[]>([])
const loading = ref(false)
const submitting = ref(false)
const rejectVisible = ref(false)
const rejectRemark = ref('')
const current = ref<ProductReport | null>(null)

async function load() {
  loading.value = true
  try {
    const res = await listPendingProductReports()
    reports.value = res.data
  } finally {
    loading.value = false
  }
}

async function takeDown(row: ProductReport) {
  try {
    await ElMessageBox.confirm(`确认下架商品「${row.product_title}」并反馈举报结果？`, '下架确认', {
      type: 'warning',
      confirmButtonText: '下架商品',
      cancelButtonText: '取消',
    })
  } catch {
    return
  }
  submitting.value = true
  try {
    const res = await handleProductReport(row.id, { action: 'take_down' })
    ElMessage.success(res.message || '已下架')
    await load()
  } finally {
    submitting.value = false
  }
}

function openReject(row: ProductReport) {
  current.value = row
  rejectRemark.value = ''
  rejectVisible.value = true
}

async function confirmReject() {
  if (!current.value) return
  if (!rejectRemark.value.trim()) {
    ElMessage.warning('请填写驳回原因')
    return
  }
  submitting.value = true
  try {
    const res = await handleProductReport(current.value.id, { action: 'reject', remark: rejectRemark.value.trim() })
    ElMessage.success(res.message || '已驳回')
    rejectVisible.value = false
    await load()
  } finally {
    submitting.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.banner {
  margin-bottom: 16px;
}
.sub {
  font-size: 12px;
  color: #909399;
}
</style>
