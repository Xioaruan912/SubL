<template>
  <div class="app-container">
    <el-card shadow="never">
      <template #header>
        <div class="head">
          <span>多订阅合集</span>
          <div class="head-actions">
            <el-select v-model="client" size="small" style="width:140px">
              <el-option v-for="c in clients" :key="c" :label="c" :value="c" />
            </el-select>
            <el-button type="primary" size="small" @click="openEdit()">新建合集</el-button>
            <el-button size="small" @click="load">刷新</el-button>
          </div>
        </div>
      </template>

      <el-table :data="items" v-loading="loading" size="small">
        <el-table-column prop="name" label="名称" min-width="160" />
        <el-table-column label="成员订阅" width="110">
          <template #default="{ row }">{{ memberIds(row).length }} 个</template>
        </el-table-column>
        <el-table-column label="去重" width="80">
          <template #default="{ row }"><el-tag size="small" :type="row.dedupe ? 'success' : 'info'">{{ row.dedupe ? '开' : '关' }}</el-tag></template>
        </el-table-column>
        <el-table-column label="到期" width="160">
          <template #default="{ row }">{{ row.expiresAt ? new Date(row.expiresAt).toLocaleString() : '永久' }}</template>
        </el-table-column>
        <el-table-column prop="note" label="备注" min-width="120" show-overflow-tooltip />
        <el-table-column label="操作" width="260" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click="copyLink(row)">复制链接</el-button>
            <el-button link type="primary" size="small" @click="openEdit(row)">编辑</el-button>
            <el-button link type="warning" size="small" @click="resetToken(row)">重置令牌</el-button>
            <el-button link type="danger" size="small" @click="remove(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="dialog" :title="form.id ? '编辑合集' : '新建合集'" width="620px">
      <el-form label-position="top">
        <el-form-item label="名称"><el-input v-model="form.name" placeholder="例如 主力合集" /></el-form-item>
        <el-form-item label="成员订阅">
          <el-select v-model="form.memberList" multiple filterable style="width:100%" placeholder="选择要合并的订阅">
            <el-option v-for="s in subs" :key="s.ID" :label="s.Name" :value="s.ID" />
          </el-select>
        </el-form-item>
        <el-form-item label="节点去重"><el-switch v-model="form.dedupe" /></el-form-item>
        <el-form-item label="到期时间（留空=永不过期）">
          <el-date-picker v-model="form.expires" type="datetime" style="width:100%" placeholder="选择到期时间" />
        </el-form-item>
        <el-form-item label="合集处理链（可选，JSON）">
          <el-input v-model="form.pipeline" type="textarea" :rows="4" placeholder='{"excludePlaceholder":true,"emoji":true,"dedupe":true}' />
        </el-form-item>
        <el-form-item label="备注"><el-input v-model="form.note" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialog=false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue";
import { getCollections, addCollection, updateCollection, deleteCollection, resetCollectionToken } from "@/api/subcription/collection";
import { getSubs } from "@/api/subcription/subs";

defineOptions({ name: "Collections" });

const clients = ["clash", "surge", "loon", "v2ray", "singbox", "qx", "shadowrocket"];
const client = ref("clash");
const items = ref<any[]>([]);
const subs = ref<any[]>([]);
const loading = ref(false);
const dialog = ref(false);
const saving = ref(false);
const form = ref<any>({ id: 0, name: "", memberList: [] as number[], dedupe: true, expires: null, pipeline: "", note: "" });

const memberIds = (row: any): number[] => {
  try { const v = JSON.parse(row.memberIds || "[]"); return Array.isArray(v) ? v : []; } catch { return []; }
};

const load = async () => {
  loading.value = true;
  try {
    const [c, s] = await Promise.all([getCollections(), getSubs()]);
    items.value = c.data || [];
    subs.value = s.data || [];
  } finally {
    loading.value = false;
  }
};

const openEdit = (row?: any) => {
  if (row) {
    form.value = {
      id: row.id, name: row.name, memberList: memberIds(row), dedupe: !!row.dedupe,
      expires: row.expiresAt ? new Date(row.expiresAt) : null, pipeline: row.pipeline || "", note: row.note || "",
    };
  } else {
    form.value = { id: 0, name: "", memberList: [], dedupe: true, expires: null, pipeline: "", note: "" };
  }
  dialog.value = true;
};

const save = async () => {
  if (!form.value.name.trim()) { ElMessage.warning("名称不能为空"); return; }
  saving.value = true;
  try {
    const payload = {
      id: form.value.id, name: form.value.name, dedupe: form.value.dedupe,
      memberIds: JSON.stringify(form.value.memberList || []),
      pipeline: form.value.pipeline || "", note: form.value.note || "",
      expiresAt: form.value.expires ? new Date(form.value.expires).toISOString() : null,
    };
    if (form.value.id) await updateCollection(payload); else await addCollection(payload);
    ElMessage.success("已保存");
    dialog.value = false;
    await load();
  } finally {
    saving.value = false;
  }
};

const copyLink = async (row: any) => {
  const link = `${location.origin}/cc/?token=${row.token}&client=${client.value}`;
  try { await navigator.clipboard.writeText(link); ElMessage.success("已复制：" + link); } catch { ElMessage.warning(link); }
};

const resetToken = async (row: any) => {
  await ElMessageBox.confirm(`重置「${row.name}」的合集令牌？旧链接将立即失效。`, "重置令牌", { type: "warning" });
  await resetCollectionToken(row.id);
  ElMessage.success("已重置");
  await load();
};

const remove = async (row: any) => {
  await ElMessageBox.confirm(`删除合集「${row.name}」？`, "删除", { type: "warning" });
  await deleteCollection(row.id);
  ElMessage.success("已删除");
  await load();
};

onMounted(load);
</script>

<style scoped>
.app-container { padding: 10px; }
.head { display: flex; justify-content: space-between; align-items: center; }
.head-actions { display: flex; gap: 8px; align-items: center; }
</style>
