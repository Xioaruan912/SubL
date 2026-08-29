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
                <el-button type="success" plain size="small" @click="router.push('/template/rules')">从规则中心添加</el-button>
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

      <!-- Loon 可视化表单模式 -->
      <template v-else>
        <el-row :gutter="16">
          <el-col :span="14" :xs="24">
            <el-form label-width="130px">
              <el-form-item label="文件名" required>
                <el-input v-model="loonForm.filename" placeholder="例如 loon.conf" />
              </el-form-item>

              <el-collapse>
                <el-collapse-item title="General 基础配置" name="loon-general">
                  <el-form-item label="IP 模式">
                    <el-select v-model="loonForm.general.ip_mode" class="full">
                      <el-option label="仅 IPv4 (v4-only)" value="v4-only" />
                      <el-option label="双栈 (dual)" value="dual" />
                      <el-option label="优先 IPv4" value="ipv4-preferred" />
                      <el-option label="优先 IPv6" value="ipv6-preferred" />
                    </el-select>
                  </el-form-item>
                  <el-form-item label="DNS 服务器">
                    <el-input v-model="loonForm.general.dns_server" placeholder="system,1.1.1.1" />
                  </el-form-item>
                  <el-form-item label="SNI 嗅探">
                    <el-switch v-model="loonForm.general.sni_sniffing" />
                  </el-form-item>
                  <el-form-item label="禁用 STUN">
                    <el-switch v-model="loonForm.general.disable_stun" />
                  </el-form-item>
                  <el-form-item label="局域网访问">
                    <el-switch v-model="loonForm.general.allow_wifi_access" />
                  </el-form-item>
                  <el-row :gutter="12">
                    <el-col :span="12"><el-form-item label="HTTP 端口"><el-input-number v-model="loonForm.general.wifi_http_port" :min="1" :max="65535" class="full" /></el-form-item></el-col>
                    <el-col :span="12"><el-form-item label="SOCKS5 端口"><el-input-number v-model="loonForm.general.wifi_socks_port" :min="1" :max="65535" class="full" /></el-form-item></el-col>
                  </el-row>
                  <el-form-item label="测速超时(秒)">
                    <el-input-number v-model="loonForm.general.test_timeout" :min="1" :max="60" class="full" />
                  </el-form-item>
                  <el-form-item label="网络测试 URL">
                    <el-input v-model="loonForm.general.internet_test_url" />
                  </el-form-item>
                  <el-form-item label="节点测速 URL">
                    <el-input v-model="loonForm.general.proxy_test_url" />
                  </el-form-item>
                  <el-form-item label="资源解析器">
                    <el-input v-model="loonForm.general.resource_parser" />
                  </el-form-item>
                  <el-form-item label="GeoIP URL">
                    <el-input v-model="loonForm.general.geoip_url" />
                  </el-form-item>
                  <el-form-item label="ASN URL">
                    <el-input v-model="loonForm.general.ipasn_url" />
                  </el-form-item>
                  <el-form-item label="UDP 兜底">
                    <el-select v-model="loonForm.general.udp_fallback_mode" class="full">
                      <el-option label="DIRECT" value="DIRECT" />
                      <el-option label="REJECT" value="REJECT" />
                    </el-select>
                  </el-form-item>
                  <el-form-item label="域名拒绝阶段">
                    <el-select v-model="loonForm.general.domain_reject_mode" class="full">
                      <el-option label="DNS" value="DNS" />
                      <el-option label="Request" value="Request" />
                    </el-select>
                  </el-form-item>
                  <el-form-item label="DNS 拒绝方式">
                    <el-select v-model="loonForm.general.dns_reject_mode" class="full">
                      <el-option label="回环 IP" value="LoopbackIP" />
                      <el-option label="空响应" value="NOANSWER" />
                      <el-option label="NXDomain" value="NXDOMAIN" />
                    </el-select>
                  </el-form-item>
                </el-collapse-item>

                <el-collapse-item title="节点筛选 (Remote Filter)" name="loon-filter">
                  <div v-for="(f, i) in loonForm.filters" :key="i" class="loon-row">
                    <el-input v-model="f.name" placeholder="筛选名（如 香港）" class="loon-name" />
                    <el-input v-model="f.regex" placeholder="NameRegex 正则" class="loon-regex" />
                    <el-button link type="danger" @click="loonForm.filters.splice(i, 1)">删除</el-button>
                  </div>
                  <el-button size="small" @click="loonForm.filters.push({ name: '', regex: '' })">添加筛选</el-button>
                </el-collapse-item>

                <el-collapse-item title="策略组 (Proxy Group)" name="loon-group">
                  <div v-for="(gr, i) in loonForm.groups" :key="i" class="loon-group-row">
                    <el-input v-model="gr.name" placeholder="组名（如 US）" class="loon-name" />
                    <el-select v-model="gr.type" class="loon-type">
                      <el-option label="select" value="select" />
                      <el-option label="url-test" value="url-test" />
                      <el-option label="fallback" value="fallback" />
                    </el-select>
                    <el-input v-model="gr.policies" placeholder="策略列表（逗号分隔）" class="loon-regex" />
                    <el-button link type="danger" @click="loonForm.groups.splice(i, 1)">删除</el-button>
                  </div>
                  <el-button size="small" @click="loonForm.groups.push({ name: '', type: 'select', policies: '', url: '', img_url: '' })">添加组</el-button>
                </el-collapse-item>

                <el-collapse-item title="远程规则 (Remote Rule)" name="loon-rule">
                  <el-input v-model="loonForm.remote_rules" type="textarea" :rows="8" placeholder="每行一条：https://xxx, policy=组名, tag=标签, enabled=true" />
                </el-collapse-item>

                <el-collapse-item title="插件 (Plugin)" name="loon-plugin">
                  <el-input v-model="loonForm.plugins" type="textarea" :rows="6" placeholder="每行一条：https://xxx.lpx, enabled=true" />
                </el-collapse-item>

                <el-collapse-item title="本地规则 (Rule)" name="loon-local-rule">
                  <el-input v-model="loonForm.rules" type="textarea" :rows="4" placeholder="本地规则，自动追加 DOMAIN-SUFFIX,local,DIRECT / GEOIP,CN,DIRECT / FINAL,FINAL" />
                </el-collapse-item>
              </el-collapse>
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
                :model-value="loonPreview"
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
import { useRouter } from "vue-router";
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
const router = useRouter();

// Loon 表单默认值（基础配置）
const defaultGeneral = () => ({
  ip_mode: "v4-only",
  dns_server: "system",
  sni_sniffing: true,
  disable_stun: true,
  allow_wifi_access: false,
  wifi_http_port: 7222,
  wifi_socks_port: 7221,
  test_timeout: 5,
  switch_node_after_failure: 3,
  internet_test_url: "https://www.youtube.com",
  proxy_test_url: "https://www.youtube.com",
  resource_parser: "https://github.com/sub-store-org/Sub-Store/releases/latest/download/sub-store-parser.loon.min.js",
  geoip_url: "https://github.com/Masaiki/GeoIP2-CN/raw/release/Country.mmdb",
  ipasn_url: "https://raw.githubusercontent.com/P3TERX/GeoLite.mmdb/download/GeoLite2-ASN.mmdb",
  udp_fallback_mode: "REJECT",
  domain_reject_mode: "DNS",
  dns_reject_mode: "LoopbackIP",
  bypass_tun: "10.0.0.0/8,100.64.0.0/10,127.0.0.0/8,169.254.0.0/16,172.16.0.0/12,192.0.0.0/24,192.0.2.0/24,192.88.99.0/24,192.168.0.0/16,198.51.100.0/24,203.0.113.0/24,224.0.0.0/4,255.255.255.255/32",
  skip_proxy: "192.168.0.0/16,10.0.0.0/8,172.16.0.0/12,localhost,*.local,e.crashlynatics.com",
});
const defaultFilters = () => [
  { name: "美国", regex: "(?i)(美国|波特兰|达拉斯|俄勒冈|凤凰城|费利蒙|硅谷|拉斯维加斯|洛杉矶|圣何塞|圣克拉拉|西雅图|芝加哥|🇺🇸|US|USA)" },
  { name: "香港", regex: "(?i)(香港|hong|HK|HKG|🇭🇰)" },
  { name: "日本", regex: "(?i)(日本|东京|大阪|泉日|埼玉|🇯🇵|JP|Japan)" },
  { name: "新加坡", regex: "(?i)(狮城|新加坡|🇸🇬|SG|Singapore)" },
  { name: "台湾", regex: "(?i)(台湾|台灣|🇹🇼|TW|Taiwan)" },
];
const defaultGroups = () => {
  const testURL = "http://cp.cloudflare.com/generate_204";
  const icon = "https://raw.githubusercontent.com/erdongchanyo/icon/main/";
  return [
    { name: "Global", type: "select", policies: "US, HK, JP, SG, DIRECT", url: testURL, img_url: icon + "Policy-Filter/Outside.png" },
    { name: "US", type: "select", policies: "美国", url: testURL, img_url: icon + "Policy-Country/US.png" },
    { name: "HK", type: "select", policies: "香港", url: testURL, img_url: icon + "Policy-Country/HK02.png" },
    { name: "JP", type: "select", policies: "日本", url: testURL, img_url: icon + "Policy-Country/JP.png" },
    { name: "SG", type: "select", policies: "新加坡", url: testURL, img_url: icon + "Policy-Country/SG.png" },
    { name: "TW", type: "select", policies: "台湾", url: testURL, img_url: icon + "Policy-Country/TW.png" },
    { name: "Netflix", type: "select", policies: "Global,TW, US, HK, JP, SG, DIRECT", url: testURL, img_url: icon + "Policy-Filter/Netflix.png" },
    { name: "Youtube", type: "select", policies: "Global, TW,HK, US, JP, SG, DIRECT", url: testURL, img_url: icon + "Policy-Filter/Youtube.png" },
    { name: "Mainland", type: "select", policies: "DIRECT", url: testURL, img_url: icon + "Policy-Filter/Mainland.png" },
    { name: "Advertising", type: "select", policies: "REJECT", url: testURL, img_url: icon + "Policy-Filter/AdBlock.png" },
    { name: "FINAL", type: "select", policies: "Global, DIRECT", url: testURL, img_url: icon + "Policy-Filter/Final01.png" },
  ];
};

const loonForm = ref({
  filename: "loon.conf",
  general: defaultGeneral(),
  filters: defaultFilters(),
  groups: defaultGroups(),
  remote_rules: "",
  plugins: "",
  rules: "",
});

// 前端生成 Loon 预览（与后端 buildLoonConfig 一致）
const loonPreview = computed(() => {
  const g = loonForm.value.general;
  const lines: string[] = [];
  lines.push("[General]", `ip-mode = ${g.ip_mode}`, `dns-server = ${g.dns_server}`, `sni-sniffing = ${g.sni_sniffing}`, `disable-stun = ${g.disable_stun}`, `udp-fallback-mode = ${g.udp_fallback_mode}`, `domain-reject-mode = ${g.domain_reject_mode}`, `dns-reject-mode = ${g.dns_reject_mode}`, `wifi-access-http-port = ${g.wifi_http_port}`, `wifi-access-socks5-port = ${g.wifi_socks_port}`, `allow-wifi-access = ${g.allow_wifi_access}`, "interface-mode = auto", `test-timeout = ${g.test_timeout}`, `switch-node-after-failure-times = ${g.switch_node_after_failure}`, `internet-test-url = ${g.internet_test_url}`, `proxy-test-url = ${g.proxy_test_url}`, `resource-parser = ${g.resource_parser}`, `geoip-url = ${g.geoip_url}`, `ipasn-url = ${g.ipasn_url}`, `skip-proxy = ${g.skip_proxy}`, `bypass-tun = ${g.bypass_tun}`, "", "[Proxy]", "", "[Remote Proxy]", "", "[Remote Filter]", "# 远程节点订阅正则筛选");
  loonForm.value.filters.forEach((f: any) => f.name && lines.push(`${f.name} = NameRegex, FilterKey = "${f.regex}"`));
  lines.push("", "[Proxy Group]", "");
  loonForm.value.groups.forEach((gr: any) => {
    if (!gr.name) return;
    let line = `${gr.name} = ${gr.type}, ${gr.policies}`;
    if (gr.url) line += `, url = ${gr.url}`;
    if (gr.img_url) line += `, img-url = ${gr.img_url}`;
    lines.push(line);
  });
  lines.push("", "[Remote Rule]", loonForm.value.remote_rules, "", "[Proxy Chain]", "[Rule]", loonForm.value.rules, "DOMAIN-SUFFIX,local,DIRECT", "GEOIP,CN,DIRECT", "FINAL,FINAL", "", "[Host]", "", "[Rewrite]", "", "[Script]", "", "[Plugin]", loonForm.value.plugins, "", "[Mitm]", "hostname =", "skip-server-cert-verify = false");
  return lines.join("\n");
});

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
        loonForm.value.filename = fname;
        // 简单解析已有文本：尽量填入表单（完整解析复杂，保留文本到 rules/remote_rules）
        loonForm.value.remote_rules = "";
        loonForm.value.plugins = "";
        loonForm.value.rules = "";
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
  const fname = target.value === "loon" ? loonForm.value.filename : form.filename;
  if (!fname.trim()) {
    ElMessage.warning("请填写文件名");
    return;
  }
  loading.value = true;
  try {
    let payload: any;
    if (target.value === "loon") {
      payload = {
        target: "loon",
        filename: loonForm.value.filename.trim(),
        general: loonForm.value.general,
        filters: loonForm.value.filters,
        groups: loonForm.value.groups,
        remote_rules: loonForm.value.remote_rules,
        plugins: loonForm.value.plugins,
        rules: loonForm.value.rules,
      };
      if (editingOldname.value) payload.edit_oldname = editingOldname.value;
    } else {
      payload = {
        ...form,
        filename: form.filename.trim(),
        target: "clash",
        rules: rulesText.value.split("\n").filter((l) => l.trim() && !l.trim().startsWith("#")).map((l) => l.trim()),
      };
      if (templateSource.value === "default") payload.source = "default";
      if (editingOldname.value) payload.edit_oldname = editingOldname.value;
    }
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

.loon-row,
.loon-group-row {
  display: flex;
  gap: 8px;
  align-items: center;
  margin-bottom: 8px;
}
.loon-name { width: 140px; flex-shrink: 0; }
.loon-regex { flex: 1; }
.loon-type { width: 110px; flex-shrink: 0; }
.full { width: 100%; }
</style>