<template>
  <div class="page">
    <h2>举报处理</h2>
    <el-radio-group v-model="statusFilter" class="filter" @change="load">
      <el-radio-button value="pending">待处理</el-radio-button>
      <el-radio-button value="taken_down">已下架</el-radio-button>
      <el-radio-button value="rejected">已驳回</el-radio-button>
    </el-radio-group>
    <el-table v-loading="loading" :data="reports" empty-text="暂无举报记录">
      <el-table-column prop="id" label="ID" width="70" />
      <el-table-column label="商品" min-width="150">
        <template #default="{ row }">
          <router-link to="/products">#{{ row.product_id }} {{ productTitle(row.product_id) }}</router-link>
        </template>
      </el-table-column>
      <el-table-column label="举报人" width="90">
        <template #default="{ row }">#{{ row.reporter_id }}</template>
      </el-table-column>
      <el-table-column label="原因" width="100">
        <template #default="{ row }">
          <el-tag size="small" type="warning">{{ reportReasonLabel(row.reason) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="description" label="情况说明" min-width="180" show-overflow-tooltip />
      <el-table-column label="状态" width="90">
        <template #default="{ row }">
          <el-tag size="small" :type="reportStatusType(row.status) as any">{{ reportStatusLabel(row.status) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="提交时间" width="150">
        <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="200" fixed="right">
        <template #default="{ row }">
          <template v-if="row.status === 'pending'">
            <el-button size="small" type="danger" @click="act(row, 'take_down')">下架</el-button>
            <el-button size="small" @click="act(row, 'reject')">驳回</el-button>
          </template>
          <template v-else>
            <span class="handled">
              处理人 #{{ row.handler_id }}<template v-if="row.handle_remark">：{{ row.handle_remark }}</template>
            </span>
          </template>
        </template>
      </el-table-column>
    </el-table>
    <el-pagination
      class="pager"
      layout="prev, pager, next, total"
      :total="total"
      :page-size="pageSize"
      :current-page="page"
      @current-change="onPageChange"
    />

    <el-dialog v-model="dialogVisible" :title="action === 'take_down' ? '下架商品' : '驳回举报'" width="420px">
      <el-alert
        :title="action === 'take_down' ? '商品已售出、已下架或已被其他管理员处理时，操作将被拒绝。' : '驳回后商品保留在售，举报原因将记录在案。'"
        type="info" :closable="false" class="tip" />
      <el-input v-model="remark" type="textarea" :rows="3" maxlength="500" show-word-limit
        :placeholder="action === 'take_down' ? '可填写下架说明（选填）' : '请填写驳回原因'" />
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button :type="action === 'take_down' ? 'danger' : 'primary'" :loading="submitting" @click="confirm">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { listReports, handleReport } from '../api/report'
import { listProducts } from '../api/product'
import { reportReasonLabel, reportStatusLabel, reportStatusType } from '../constants/report'
import { formatDateTime } from '../utils/dateFormat'
import type { Product, Report } from '../types'

const reports = ref<Report[]>([])
const productMap = ref<Record<number, Product>>({})
const total = ref(0)
const page = ref(1)
const pageSize = 10
const loading = ref(false)
const statusFilter = ref('pending')

function productTitle(id: number): string {
  return productMap.value[id]?.title ?? ''
}

const dialogVisible = ref(false)
const submitting = ref(false)
const action = ref<'take_down' | 'reject'>('take_down')
const current = ref<Report | null>(null)
const remark = ref('')

async function load() {
  loading.value = true
  try {
    const [reportRes, productRes] = await Promise.all([
      listReports({ status: statusFilter.value, page: page.value, page_size: pageSize }),
      listProducts({ page: 1, page_size: 100 }),
    ])
    reports.value = reportRes.data.items
    total.value = reportRes.data.total
    productMap.value = Object.fromEntries(productRes.data.items.map((p) => [p.id, p]))
  } finally {
    loading.value = false
  }
}

function onPageChange(p: number) {
  page.value = p
  load()
}

async function act(row: Report, a: 'take_down' | 'reject') {
  current.value = row
  action.value = a
  remark.value = ''
  if (a === 'take_down') {
    try {
      await ElMessageBox.confirm(`确定下架商品 #${row.product_id} 吗？下架后商品状态与举报结果将同时生效。`, '下架确认', {
        type: 'warning',
        confirmButtonText: '下架',
        cancelButtonText: '取消',
      })
    } catch {
      return
    }
  }
  dialogVisible.value = true
}

async function confirm() {
  if (!current.value) return
  if (action.value === 'reject' && !remark.value.trim()) {
    ElMessage.warning('请填写驳回原因')
    return
  }
  submitting.value = true
  try {
    await handleReport(current.value.id, { action: action.value, remark: remark.value })
    ElMessage.success(action.value === 'take_down' ? '已下架商品，举报成立' : '已驳回举报，商品保留在售')
    dialogVisible.value = false
    await load()
  } finally {
    submitting.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.filter {
  margin-bottom: 16px;
}
.pager {
  margin-top: 16px;
  justify-content: flex-end;
}
.handled {
  font-size: 12px;
  color: #909399;
}
.tip {
  margin-bottom: 12px;
}
</style>
