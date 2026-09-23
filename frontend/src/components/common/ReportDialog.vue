<template>
  <el-dialog v-model="visible" title="举报商品" width="440px" @closed="handleClosed">
    <div class="report-product" v-if="product">
      <el-tag size="small">{{ productStatusLabel(product.status) }}</el-tag>
      <span class="report-product-title">{{ product.title }}</span>
    </div>
    <el-form ref="formRef" :model="form" :rules="rules" label-width="72px">
      <el-form-item label="举报原因" prop="reason">
        <el-select v-model="form.reason" placeholder="请选择原因" style="width: 100%">
          <el-option v-for="r in REPORT_REASONS" :key="r.value" :label="r.label" :value="r.value" />
        </el-select>
      </el-form-item>
      <el-form-item label="情况说明" prop="description">
        <el-input v-model="form.description" type="textarea" :rows="4" maxlength="500" show-word-limit
          placeholder="请描述具体情况，同一商品重复举报将返回原记录" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="danger" :loading="submitting" @click="submit">提交举报</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { ElMessage, type FormInstance } from 'element-plus'
import { REPORT_REASONS } from '../../constants/report'
import { productStatusLabel } from '../../constants/product'
import { createReport } from '../../api/report'
import type { Product, Report } from '../../types'

const props = defineProps<{ product: Product | null }>()
const emit = defineEmits<{ (e: 'submitted', report: Report): void; (e: 'close'): void }>()

const visible = ref(false)
const submitting = ref(false)
const formRef = ref<FormInstance>()
const form = reactive({ reason: '', description: '' })
const rules = {
  reason: [{ required: true, message: '请选择举报原因', trigger: 'change' }],
}

watch(() => props.product, (p) => {
  if (p) {
    visible.value = true
    form.reason = ''
    form.description = ''
    formRef.value?.clearValidate()
  }
})

function handleClosed() {
  emit('close')
}

async function submit() {
  if (!props.product || !formRef.value) return
  await formRef.value.validate()
  submitting.value = true
  try {
    const res = await createReport({
      product_id: props.product.id,
      reason: form.reason,
      description: form.description,
    })
    ElMessage.success('举报已提交，等待管理员处理')
    visible.value = false
    emit('submitted', res.data)
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
.report-product {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 16px;
}
.report-product-title {
  font-weight: 600;
  color: #303133;
}
</style>
