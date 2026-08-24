<template>
  <div class="app-container">
    <el-row :gutter="16">
      <!-- 左侧表单 -->
      <el-col :span="14" :xs="24">
        <el-card shadow="never">
          <template #header>
            <div class="card-header">构建 Clash 模板</div>
          </template>

          <el-form label-width="110px">
            <el-form-item label="文件名" required>
              <el-input v-model="form.filename" placeholder="例如 my_clash.yaml" />
            </el-form-item>

            <el-row :gutter="12">
              <el-col :span="8">
                <el-form-item label="HTTP 端口">
                  <el-input-number v-model="form.port" :min="1" :max="65535" style="width: 100%" />
                </el-form-item>
              </el-col>
              <el-col :span="8">
                <el-form-item label="SOCKS 端口">
                  <el-input-number v-model="form.socks_port" :min="1" :max="65535" style="width: 100%" />
                </el-form-item>
              </el-col>
              <el-col :span="8">
                <el-form-item label="模式">
                  <el-select v-model="form.mode" style="width: 100%">
                    <el-option label="Rule（规则）" value="Rule" />
                    <el-option label="Global（全局）" value="Global" />
                    <el-option label="Direct（直连）" value="Direct" />
                  </el-select>
                </el-form-item>
              </el-col>
            </el-row>

            <el-form-item label="允许局域网">
              <el-switch v-model="form.allow_lan" />
            </el-form-item>

            <!-- 分组 -->
            <el-form-item label="策略分组">
              <div class="group-list">
                <div v-for="(g, i) in form.groups" :key="i" class="group-row">
                  <el-input v-model="g.name" placeholder="分组名（如 AI）" class="g-name" />
                  <el-select v-model="g.type" class="g-type">
                    <el-option label="select 手动选择" value="select" />
                    <el-option label="url-test 自动测速" value="url-test" />
                    <el-option label="fallback 故障转移" value="fallback" />
                  </el-select>
                  <el-input v-model="g.filter" placeholder="filter 正则（如 (?i)US|美国）" class="g-filter" />
                  <el-checkbox v-model="g.include_all_providers" title="匹配全部节点">全部</el-checkbox>
                  <el-button link type="danger" @click="removeGroup(i)">
                    <i-ep-delete />
                  </el-button>
                </div>
                <el-button type="primary" plain size="small" @click="addGroup">
                  + 添加分组
                </el-button>
              </div>
              <div class="group-hint">
                类型说明：<b>select</b> = 手动选择节点；<b>url-test</b> = 自动测速选最快节点；<b>fallback</b> = 按顺序优先可用节点（自动故障转移）。filter 为正则，匹配的节点自动填入该分组。
              </div>
            </el-form-item>

            <el-row :gutter="12">
              <el-col :span="14">
                <el-form-item label="测速 URL">
                  <el-input v-model="form.test_url" placeholder="http://www.gstatic.com/generate_204" />
                </el-form-item>
              </el-col>
              <el-col :span="10">
                <el-form-item label="测速间隔(s)">
                  <el-input-number v-model="form.interval" :min="60" :max="86400" style="width: 100%" />
                </el-form-item>
              </el-col>
            </el-row>

            <el-form-item>
              <el-button type="primary" :loading="loading" @click="build">
                生成并保存
              </el-button>
            </el-form-item>
          </el-form>
        </el-card>
      </el-col>

      <!-- 右侧预览 -->
      <el-col :span="10" :xs="24">
        <el-card shadow="never">
          <template #header>
            <div class="card-header">
              生成预览
              <el-button v-if="yamlPreview" text size="small" @click="copyYaml">复制</el-button>
            </div>
          </template>
          <el-input
            v-model="yamlPreview"
            type="textarea"
            :rows="30"
            readonly
            placeholder="生成后在此显示 YAML 内容"
            class="yaml-preview"
          />
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from "vue";
import { BuildClashTemplate } from "@/api/template/build";

defineOptions({
  name: "TemplateBuilder",
});

interface GroupItem {
  name: string;
  type: string;
  filter: string;
  include_all_providers: boolean;
}

const loading = ref(false);
const yamlPreview = ref("");
const savedFilename = ref("");

const form = reactive({
  filename: "my_clash.yaml",
  port: 7890,
  socks_port: 7891,
  allow_lan: true,
  mode: "Rule",
  test_url: "http://www.gstatic.com/generate_204",
  interval: 300,
  groups: [
    { name: "🔰 节点选择", type: "select", filter: "", include_all_providers: false },
  ] as GroupItem[],
});

const addGroup = () => {
  form.groups.push({ name: "", type: "select", filter: "", include_all_providers: false });
};

const removeGroup = (i: number) => {
  form.groups.splice(i, 1);
};

const build = async () => {
  if (!form.filename.trim()) {
    ElMessage.warning("请填写文件名");
    return;
  }
  if (form.groups.length === 0) {
    ElMessage.warning("请至少添加一个分组");
    return;
  }
  loading.value = true;
  try {
    const { data } = await BuildClashTemplate({ ...form, filename: form.filename.trim() });
    yamlPreview.value = data?.yaml || "";
    savedFilename.value = data?.filename || "";
    ElMessage.success(`模板已保存：${savedFilename.value}`);
  } catch (e: any) {
    ElMessage.error(e?.message || "生成失败");
  } finally {
    loading.value = false;
  }
};

const copyYaml = async () => {
  try {
    await navigator.clipboard.writeText(yamlPreview.value);
    ElMessage.success("已复制");
  } catch {
    ElMessage.warning("复制失败");
  }
};
</script>

<style lang="scss" scoped>
.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 15px;
  font-weight: 600;
}

.group-list {
  width: 100%;

  .group-row {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 8px;

    .g-name {
      width: 140px;
    }

    .g-type {
      width: 160px;
    }

    .g-filter {
      flex: 1;
    }
  }
}

.group-hint {
  margin-top: 6px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.5;
}

.yaml-preview {
  font-family: monospace;
  font-size: 12px;
}
</style>