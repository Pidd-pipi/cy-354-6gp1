<template>
  <div class="page">
    <h2>🎓 毕业季专场</h2>
    <el-alert title="毕业季专场：学长学姐闲置好物集中放送" type="warning" :closable="false" class="banner" />
    <el-row :gutter="16">
      <el-col v-for="p in products" :key="p.id" :span="6" class="col">
        <ProductCard :product="p" show-report :current-user-id="authStore.user?.id || 0"
          @detail="showDetail" @buy="buy" @report="openReport" />
      </el-col>
    </el-row>
    <el-empty v-if="!loading && products.length === 0" description="专场暂无商品" />
    <el-dialog v-model="detailVisible" :title="current?.title" width="520px">
      <el-descriptions :column="2" border v-if="current">
        <el-descriptions-item label="分类">{{ categoryLabel(current.category) }}</el-descriptions-item>
        <el-descriptions-item label="成色">{{ current.condition }}</el-descriptions-item>
        <el-descriptions-item label="校区">{{ current.campus }}</el-descriptions-item>
        <el-descriptions-item label="价格">¥{{ current.price.toFixed(2) }}</el-descriptions-item>
        <el-descriptions-item label="描述" :span="2">{{ current.description }}</el-descriptions-item>
      </el-descriptions>
      <template v-if="current" #footer>
        <el-button type="primary" :disabled="current.status !== 'on_sale'" @click="buy(current)">购买</el-button>
        <el-button type="danger" plain
          :disabled="current.status !== 'on_sale' || current.seller_id === (authStore.user?.id || 0)"
          @click="openReport(current)">举报</el-button>
      </template>
    </el-dialog>
    <ReportDialog ref="reportDialogRef" @submitted="loadProducts" />
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import ProductCard from '../components/common/ProductCard.vue'
import ReportDialog from '../components/common/ReportDialog.vue'
import { listGraduation } from '../api/product'
import { createTradeOrder } from '../api/tradeOrder'
import { categoryLabel } from '../constants/product'
import type { Product } from '../types'
import { useAuthStore } from '../stores/authStore'
import { useRouter } from 'vue-router'

const products = ref<Product[]>([])
const loading = ref(false)
const detailVisible = ref(false)
const current = ref<Product | null>(null)
const reportDialogRef = ref<InstanceType<typeof ReportDialog> | null>(null)
const authStore = useAuthStore()
const router = useRouter()

function showDetail(p: Product) {
  current.value = p
  detailVisible.value = true
}

function openReport(p: Product) {
  if (!authStore.token) {
    ElMessage.warning('请先登录')
    router.push('/login')
    return
  }
  reportDialogRef.value?.open(p)
}

async function buy(p: Product) {
  if (!authStore.token) {
    ElMessage.warning('请先登录')
    router.push('/login')
    return
  }
  await createTradeOrder(p.id)
  ElMessage.success('已下单')
}

async function loadProducts() {
  loading.value = true
  try {
    const res = await listGraduation()
    products.value = res.data.items
  } finally {
    loading.value = false
  }
}

onMounted(loadProducts)
</script>

<style scoped>
.banner {
  margin-bottom: 16px;
}
.col {
  margin-bottom: 16px;
}
</style>
