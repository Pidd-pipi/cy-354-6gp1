<template>
  <el-dialog v-model="visible" title="举报商品" width="440px" @closed="onClosed">
    <el-alert title="如商品存在虚假描述、违禁物品等问题，请提交举报，平台管理员会尽快处理。" type="warning" :closable="false" class="report-tip" />
    <el-descriptions :column="1" border v-if="product" class="report-product">
      <el-descriptions-item label="商品">{{ product.title }}（#{{ product.id }}）</el-descriptions-item>
    </el-descriptions>
    <el-form ref="formRef" :model="form" :rules="rules" label-width="72px">
      <el-form-item label="举报原因" prop="reason">
        <el-select v-model="form.reason" placeholder="请选择举报原因" style="width: 100%">
          <el-option v-for="r in REPORT_REASONS" :key="r.value" :label="r.label" :value="r.value" />
        </el-select>
      </el-form-item>
      <el-form-item label="补充说明" prop="description">
        <el-input v-model="form.description" type="textarea" :rows="4" maxlength="500" show-word-limit
          placeholder="请描述具体问题，便于管理员核实" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="danger" :loading="submitting" @click="submit">提交举报</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import { ElMessage } from 'element-plus'
import { REPORT_REASONS } from '../../constants/report'
import { createProductReport } from '../../api/productReport'
import type { Product } from '../../types'

const emit = defineEmits<{ (e: 'submitted'): void }>()

const visible = ref(false)
const submitting = ref(false)
const product = ref<Product | null>(null)
const formRef = ref<FormInstance>()
const form = reactive({ reason: 'false_description', description: '' })

const rules: FormRules = {
  reason: [{ required: true, message: '请选择举报原因', trigger: 'change' }],
  description: [{ max: 500, message: '说明不能超过 500 字', trigger: 'blur' }],
}

function open(target: Product) {
  product.value = target
  form.reason = 'false_description'
  form.description = ''
  visible.value = true
}

function onClosed() {
  product.value = null
  formRef.value?.resetFields()
}

async function submit() {
  if (!product.value) return
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return
  submitting.value = true
  try {
    const res = await createProductReport({
      product_id: product.value.id,
      reason: form.reason,
      description: form.description,
    })
    ElMessage.success(res.message || '举报已提交')
    visible.value = false
    emit('submitted')
  } finally {
    submitting.value = false
  }
}

defineExpose({ open })
</script>

<style scoped>
.report-tip {
  margin: 0 0 12px;
}
.report-product {
  margin-bottom: 12px;
}
</style>
