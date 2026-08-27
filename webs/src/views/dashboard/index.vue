<template>
  <div class="p-4 md:p-6 lg:p-8 h-full flex flex-col gap-6">
    <!-- Top greeting bar (Tailwind Card) -->
    <div class="bg-white dark:bg-[#1a1d1b] rounded-xl shadow-[inset_0_0_0_1px_rgba(0,0,0,0.06)] dark:shadow-[inset_0_0_0_1px_rgba(255,255,255,0.08)] p-6">
      <el-row justify="space-between">
        <el-col :span="18" :xs="24">
          <div class="flex h-full items-center">
            <span class="w-12 h-12 rounded-full bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400 flex items-center justify-center text-xl font-bold mr-4">{{ (userStore.user.username || 'U').charAt(0).toUpperCase() }}</span>
            <div>
              <p class="text-lg font-medium text-gray-800 dark:text-gray-200">{{ greetings }}</p>
            </div>
          </div>
        </el-col>

        <el-col :span="6" :xs="24">
          <div class="flex h-full items-center justify-around">
            <el-statistic
              v-for="item in statisticData"
              :key="item.key"
              :value="item.value"
            >
              <template #title>
                <div class="flex items-center text-gray-500 dark:text-gray-400">
                  <svg-icon :icon-class="item.iconClass" size="18px" />
                  <span class="text-sm ml-1">{{ item.title }}</span>
                </div>
              </template>
            </el-statistic>
          </div>
        </el-col>
      </el-row>
    </div>

    <!-- Map & Ping Section -->
    <el-row :gutter="24" class="flex-1">
      <el-col :span="16" :xs="24" class="mb-4">
        <div class="bg-white dark:bg-[#1a1d1b] rounded-xl shadow-[inset_0_0_0_1px_rgba(0,0,0,0.06)] dark:shadow-[inset_0_0_0_1px_rgba(255,255,255,0.08)] p-6 h-full">
          <world-map />
        </div>
      </el-col>
      <el-col :span="8" :xs="24" class="mb-4">
        <div class="bg-white dark:bg-[#1a1d1b] rounded-xl shadow-[inset_0_0_0_1px_rgba(0,0,0,0.06)] dark:shadow-[inset_0_0_0_1px_rgba(255,255,255,0.08)] p-6 h-full">
          <node-ping />
        </div>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
defineOptions({
  name: "Dashboard",
  inheritAttrs: false,
});

import { useUserStore } from "@/store/modules/user";
import { getSubTotal,getNodeTotal } from "@/api/total";
const WorldMap = defineAsyncComponent(() => import("./components/WorldMap.vue"));
const NodePing = defineAsyncComponent(() => import("./components/NodePing.vue"));
const userStore = useUserStore();
const date: Date = new Date();
const subTotal = ref(0);
const nodeTotal = ref(0);
// 右上角数量
const statisticData = ref([
  {
    value: 0,
    iconClass: "message",
    title: "订阅",
    key: "message",
  },
  {
    value: 0,
    iconClass: "link",
    title: "节点",
    key: "upcoming",
  },
]);
const getsubtotal = async () => {
  const { data } = await getSubTotal();
  subTotal.value = data;
  statisticData.value[0].value = data;
};
const getnodetotal = async () => {
  const { data } = await getNodeTotal();
  nodeTotal.value = data;
  statisticData.value[1].value = data;
};
onMounted(() => {
  getsubtotal();
  getnodetotal();
});
const greetings = computed(() => {
  const hours = date.getHours();
  if (hours >= 6 && hours < 8) {
    return "晨起披衣出草堂，轩窗已自喜微凉🌅！";
  } else if (hours >= 8 && hours < 12) {
    return "上午好，" + userStore.user.nickname + "！";
  } else if (hours >= 12 && hours < 18) {
    return "下午好，" + userStore.user.nickname + "！";
  } else if (hours >= 18 && hours < 24) {
    return "晚上好，" + userStore.user.nickname + "！";
  } else {
    return "偷偷向银河要了一把碎星，只等你闭上眼睛撒入你的梦中，晚安🌛！";
  }
});




</script>

<style lang="scss" scoped>
.dashboard-container {
  position: relative;

  .data-box {
    display: flex;
    justify-content: space-between;
    padding: 20px;
    font-weight: bold;
    color: var(--el-text-color-regular);
    background: var(--el-bg-color-overlay);
    border-color: var(--el-border-color);
    box-shadow: var(--el-box-shadow-dark);
  }

  .svg-icon {
    fill: currentcolor !important;
  }
}
</style>
