<template>
  <div class="app-container">
    <el-card shadow="never">
      <template #header>
        <div class="card-header">
          <span>操作模板</span>
          <div class="header-actions">
            <el-select
              v-model="target"
              placeholder="目标类型"
              class="source-select"
              style="width: 130px"
            >
              <el-option label="Clash/Mihomo" value="clash" />
              <el-option label="Loon" value="loon" />
            </el-select>
            <el-select
              v-model="templateSource"
              placeholder="选择模板来源"
              class="source-select"
              @change="onSourceChange"
            >
              <el-option label="内置默认 mihomo 配置" value="default" />
              <el-option label="新建空白" value="blank" />
              <el-option
                v-for="t in templateFiles"
                :key="t.file"
                :label="'编辑已有：' + t.file"
                :value="'file:' + t.file"
              />
            </el-select>
            <el-button :loading="loading" type="primary" @click="build">
              生成并保存
            </el-button>
          </div>
        </div>
      </template>

      <template v-if="target !== 'loon'">
      <el-row :gutter="16">
        <el-col :span="14" :xs="24">
          <el-form label-width="120px">
            <!-- 文件名 -->
            <el-form-item label="文件名" required>
              <el-input v-model="form.filename" placeholder="例如 my_clash.yaml" />
            </el-form-item>

            <!-- 端口区块 -->
            <el-collapse>
              <el-collapse-item title="端口配置" name="ports">
                <el-row :gutter="12">
                  <el-col :span="8"><el-form-item label="HTTP 端口"><el-input-number v-model="form.port" :min="1" :max="65535" style="width:100%" /></el-form-item></el-col>
                  <el-col :span="8"><el-form-item label="SOCKS 端口"><el-input-number v-model="form.socks_port" :min="1" :max="65535" style="width:100%" /></el-form-item></el-col>
                  <el-col :span="8"><el-form-item label="Mixed 端口"><el-input-number v-model="form.mixed_port" :min="1" :max="65535" style="width:100%" /></el-form-item></el-col>
                </el-row>
                <el-row :gutter="12">
                  <el-col :span="8"><el-form-item label="Redir 端口"><el-input-number v-model="form.redir_port" :min="1" :max="65535" style="width:100%" /></el-form-item></el-col>
                  <el-col :span="8"><el-form-item label="TProxy 端口"><el-input-number v-model="form.tproxy_port" :min="1" :max="65535" style="width:100%" /></el-form-item></el-col>
                  <el-col :span="8"><el-form-item label="DNS 端口"><el-input-number v-model="form.dns_listen_port" :min="1" :max="65535" style="width:100%" /></el-form-item></el-col>
                </el-row>
                <el-form-item label="外部控制">
                  <el-input v-model="form.external_controller" placeholder="0.0.0.0:9090" style="width:100%" />
                </el-form-item>
                <el-form-item label="端口偏移联动">
                  <el-switch v-model="form.port_offset" />
                  <span class="field-hint">开启后修改主端口，mixed/redir/tproxy/socks 端口自动偏移</span>
                </el-form-item>
              </el-collapse-item>

              <el-collapse-item title="基础配置" name="base">
                <el-row :gutter="12">
                  <el-col :span="8"><el-form-item label="允许局域网"><el-switch v-model="form.allow_lan" /></el-form-item></el-col>
                  <el-col :span="8"><el-form-item label="IPv6"><el-switch v-model="form.ipv6" /></el-form-item></el-col>
                  <el-col :span="8"><el-form-item label="UDP"><el-switch v-model="form.udp" /></el-form-item></el-col>
                </el-row>
                <el-row :gutter="12">
                  <el-col :span="8">
                    <el-form-item label="模式">
                      <el-select v-model="form.mode" style="width:100%">
                        <el-option label="Rule（规则）" value="rule" />
                        <el-option label="Global（全局）" value="global" />
                        <el-option label="Direct（直连）" value="direct" />
                      </el-select>
                    </el-form-item>
                  </el-col>
                  <el-col :span="8"><el-form-item label="日志级别"><el-select v-model="form.log_level" style="width:100%"><el-option label="info" value="info" /><el-option label="debug" value="debug" /><el-option label="warning" value="warning" /><el-option label="error" value="error" /></el-select></el-form-item></el-col>
                  <el-col :span="8"><el-form-item label="客户端指纹"><el-input v-model="form.global_client_fingerprint" placeholder="chrome" style="width:100%" /></el-form-item></el-col>
                </el-row>
              </el-collapse-item>

              <el-collapse-item title="高级配置（geodata / tun / sniffer / dns）" name="advanced">
                <el-row :gutter="12">
                  <el-col :span="8"><el-form-item label="统一延迟"><el-switch v-model="form.unified_delay" /></el-form-item></el-col>
                  <el-col :span="8"><el-form-item label="Geodata 模式"><el-switch v-model="form.geodata_mode" /></el-form-item></el-col>
                  <el-col :span="8"><el-form-item label="Geo 自动更新"><el-switch v-model="form.geo_auto_update" /></el-form-item></el-col>
                </el-row>
                <el-row :gutter="12">
                  <el-col :span="8"><el-form-item label="更新间隔(h)"><el-input-number v-model="form.geo_update_interval" :min="1" :max="168" style="width:100%" /></el-form-item></el-col>
                  <el-col :span="8"><el-form-item label="TCP 并发"><el-switch v-model="form.tcp_concurrent" /></el-form-item></el-col>
                  <el-col :span="8"><el-form-item label="进程模式"><el-input v-model="form.find_process_mode" placeholder="strict" style="width:100%" /></el-form-item></el-col>
                </el-row>
                <el-row :gutter="12">
                  <el-col :span="8"><el-form-item label="Sniffer"><el-switch v-model="form.sniffer_enable" /></el-form-item></el-col>
                  <el-col :span="8"><el-form-item label="TUN"><el-switch v-model="form.tun_enable" /></el-form-item></el-col>
                  <el-col :span="8"><el-form-item label="TUN 栈"><el-select v-model="form.tun_stack" style="width:100%"><el-option label="system" value="system" /><el-option label="gvisor" value="gvisor" /><el-option label="mixed" value="mixed" /></el-select></el-form-item></el-col>
                </el-row>
                <el-row :gutter="12">
                  <el-col :span="8"><el-form-item label="DNS"><el-switch v-model="form.dns_enable" /></el-form-item></el-col>
                  <el-col :span="8">
                    <el-form-item label="DNS 模式">
                      <el-select v-model="form.dns_enhanced_mode" style="width:100%">
                        <el-option label="fake-ip" value="fake-ip" />
                        <el-option label="redir-host" value="redir-host" />
                        <el-option label="normal" value="normal" />
                      </el-select>
                    </el-form-item>
                  </el-col>
                </el-row>
              </el-collapse-item>
            </el-collapse>

            <!-- 策略分组 -->
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
                  <el-button link type="danger" @click="removeGroup(i)"><i-ep-delete /></el-button>
                </div>
                <el-button type="primary" plain size="small" @click="addGroup">+ 添加分组</el-button>
              </div>
              <div class="group-hint">类型说明：<b>select</b>=手动选择；<b>url-test</b>=自动测速；<b>fallback</b>=故障转移。filter 为正则，匹配节点自动填入。</div>
            </el-form-item>

            <!-- 规则集 -->
            <el-form-item label="规则集 (rule-providers)">
              <div class="group-list">
                <div v-for="(rp, i) in form.rule_providers" :key="i" class="rp-row">
                  <el-input v-model="rp.name" placeholder="名称（如 AI）" class="rp-name" />
                  <el-input v-model="rp.url" placeholder="规则集 URL" class="rp-url" />
                  <el-select v-model="rp.behavior" class="rp-behavior">
                    <el-option label="classical" value="classical" />
                    <el-option label="domain" value="domain" />
                    <el-option label="ipcidr" value="ipcidr" />
                  </el-select>
                  <el-button link type="danger" @click="removeRP(i)"><i-ep-delete /></el-button>
                </div>
                <el-button type="primary" plain size="small" @click="addRP">+ 添加规则集</el-button>
              </div>
            </el-form-item>

            <!-- 规则文本 -->
            <el-form-item label="规则 (rules)">
              <el-input
                v-model="rulesText"
                type="textarea"
                :rows="8"
                placeholder="每行一条规则，如：RULE-SET,AI,AI / GEOIP,CN,DIRECT / MATCH,日常使用"
              />
            </el-form-item>
          </el-form>
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
              :rows="40"
              readonly
              placeholder="生成后在此显示 YAML 内容"
              class="yaml-preview"
            />
          </el-card>
        </el-col>
      </el-row>
      </template>

      <!-- Loon 文本模板模式 -->
      <template v-else>
        <el-row :gutter="16">
          <el-col :span="14" :xs="24">
            <el-form label-width="120px">
              <el-form-item label="文件名" required>
                <el-input v-model="form.filename" placeholder="例如 loon.conf" />
              </el-form-item>
              <el-form-item label="Loon 配置">
                <el-input
                  v-model="loonText"
                  type="textarea"
                  :rows="34"
                  placeholder="粘贴 Loon 配置（含 [Proxy] 段，订阅时自动填充节点）"
                />
              </el-form-item>
            </el-form>
          </el-col>
          <el-col :span="10" :xs="24">
            <el-card shadow="never" class="preview-card">
              <template #header>
                <div class="preview-header">
                  <span>配置预览</span>
                  <el-button link type="primary" size="small" @click="copyYaml">复制</el-button>
                </div>
              </template>
              <el-input
                :model-value="loonText"
                type="textarea"
                :rows="40"
                readonly
                placeholder="Loon 配置预览"
                class="yaml-preview"
              />
            </el-card>
          </el-col>
        </el-row>
      </template>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed } from "vue";
import { BuildClashTemplate, GetDefaultTemplate } from "@/api/template/build";
import { getTemp } from "@/api/template/temp";

defineOptions({
  name: "TemplateBuilder",
});

interface GroupItem {
  name: string;
  type: string;
  filter: string;
  include_all_providers: boolean;
}
interface RuleProviderItem {
  name: string;
  url: string;
  behavior: string;
  type: string;
  interval: number;
  format: string;
  proxy: string;
  path: string;
}

const loading = ref(false);
const yamlPreview = ref("");
const savedFilename = ref("");
const templateSource = ref("default");
const templateFiles = ref<any[]>([]);
const editingOldname = ref("");
const target = ref("clash"); // clash / loon
const loonText = ref("");

const form = reactive({
  filename: "mihomo_default.yaml",
  port: 7890,
  socks_port: 7891,
  mixed_port: 7892,
  redir_port: 7893,
  tproxy_port: 7894,
  dns_listen_port: 1053,
  external_controller: "0.0.0.0:9090",
  port_offset: true,
  allow_lan: true,
  ipv6: true,
  udp: true,
  mode: "rule",
  log_level: "info",
  global_client_fingerprint: "chrome",
  unified_delay: true,
  geodata_mode: false,
  geo_auto_update: true,
  geo_update_interval: 24,
  tcp_concurrent: true,
  find_process_mode: "strict",
  sniffer_enable: true,
  tun_enable: true,
  tun_stack: "system",
  dns_enable: true,
  dns_enhanced_mode: "fake-ip",
  test_url: "http://www.gstatic.com/generate_204",
  interval: 300,
  groups: [] as GroupItem[],
  rule_providers: [] as RuleProviderItem[],
});

const rulesText = ref("");

const loadDefault = async () => {
  try {
    const { data } = await GetDefaultTemplate();
    if (data) {
      Object.assign(form, data, { filename: "mihomo_default.yaml" });
      rulesText.value = (data.rules || []).join("\n");
    }
  } catch {
    // ignore
  }
};

const onSourceChange = async (val: string) => {
  editingOldname.value = "";
  if (val === "default") {
    target.value = "clash";
    await loadDefault();
  } else if (val === "blank") {
    form.filename = "my_clash.yaml";
    form.groups = [{ name: "🔰 节点选择", type: "select", filter: "", include_all_providers: false }];
    form.rule_providers = [];
    rulesText.value = "";
  } else if (val.startsWith("file:")) {
    const fname = val.slice(5);
    const item = templateFiles.value.find((t) => t.file === fname);
    if (item) {
      editingOldname.value = fname;
      form.filename = fname;
      // 判断是否为 Loon 配置（含 [Proxy]/[General] 段 或 .conf 后缀）
      const isLoon = fname.endsWith(".conf") || /\[(General|Proxy|Remote Filter|Plugin)\]/.test(item.text || "");
      if (isLoon) {
        target.value = "loon";
        loonText.value = item.text || "";
      } else {
        target.value = "clash";
        rulesText.value = item.text || "";
      }
    }
  }
};

const addGroup = () => {
  form.groups.push({ name: "", type: "select", filter: "", include_all_providers: false });
};
const removeGroup = (i: number) => form.groups.splice(i, 1);

const addRP = () => {
  form.rule_providers.push({ name: "", url: "", behavior: "classical", type: "http", interval: 3600, format: "yaml", proxy: "DIRECT", path: "./rules/" });
};
const removeRP = (i: number) => form.rule_providers.splice(i, 1);

const build = async () => {
  if (!form.filename.trim()) {
    ElMessage.warning("请填写文件名");
    return;
  }
  loading.value = true;
  try {
    const payload: any = {
      ...form,
      filename: form.filename.trim(),
      target: target.value,
      loon_text: target.value === "loon" ? loonText.value : "",
      rules: rulesText.value.split("\n").filter((l) => l.trim() && !l.trim().startsWith("#")).map((l) => l.trim()),
    };
    if (templateSource.value === "default") payload.source = "default";
    if (editingOldname.value) payload.edit_oldname = editingOldname.value;
    const { data } = await BuildClashTemplate(payload);
    yamlPreview.value = data?.yaml || "";
    savedFilename.value = data?.filename || "";
    ElMessage.success(`模板已保存：${savedFilename.value}`);
    loadTemplates();
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

const loadTemplates = async () => {
  try {
    const { data } = await getTemp();
    templateFiles.value = data || [];
  } catch {
    templateFiles.value = [];
  }
};

onMounted(async () => {
  await loadDefault();
  await loadTemplates();
});
</script>

<style lang="scss" scoped>
.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 15px;
  font-weight: 600;

  .header-actions {
    display: flex;
    gap: 8px;
    align-items: center;
  }

  .source-select {
    width: 240px;
  }
}

.group-list {
  width: 100%;

  .group-row, .rp-row {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 8px;

    .g-name { width: 120px; }
    .g-type { width: 150px; }
    .g-filter { flex: 1; }
    .rp-name { width: 110px; }
    .rp-url { flex: 1; }
    .rp-behavior { width: 110px; }
  }
}

.group-hint {
  margin-top: 6px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.5;
}

.field-hint {
  margin-left: 8px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.yaml-preview {
  font-family: monospace;
  font-size: 12px;
}
</style>