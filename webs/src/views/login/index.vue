<template>
  <div class="login-container">
    <!-- 顶部 -->
    <div class="absolute-lt flex-x-end p-3 w-full">
      <el-switch
        v-model="isDark"
        inline-prompt
        :active-icon="Moon"
        :inactive-icon="Sunny"
        @change="toggleTheme"
      />
      <lang-select class="ml-2 cursor-pointer" />
    </div>
    <!-- 登录表单 -->
    <el-card class="login-panel w-100 <sm:w-85">
      <div class="login-brand">
        <img :src="logo" alt="" />
        <div>
          <h2>{{ defaultSettings.title }}</h2>
          <p>节点与订阅控制台</p>
        </div>
        <el-tag effect="plain">v{{ version }}</el-tag>
      </div>

      <div class="login-intro">
        <span class="intro-dot"></span>
        <span>欢迎回来，请登录以继续管理服务</span>
      </div>

      <el-form
        ref="loginFormRef"
        :model="loginData"
        :rules="loginRules"
        class="login-form"
      >
        <!-- 用户名 -->
        <el-form-item prop="username">
          <div class="flex-y-center w-full">
            <svg-icon icon-class="user" class="mx-2" />
            <el-input
              ref="username"
              v-model="loginData.username"
              :placeholder="$t('login.username')"
              name="username"
              size="large"
              class="h-[48px]"
            />
          </div>
        </el-form-item>

        <!-- 密码 -->
        <el-tooltip
          :visible="isCapslock"
          :content="$t('login.capsLock')"
          placement="right"
        >
          <el-form-item prop="password">
            <div class="flex-y-center w-full">
              <svg-icon icon-class="lock" class="mx-2" />
              <el-input
                v-model="loginData.password"
                :placeholder="$t('login.password')"
                type="password"
                name="password"
                @keyup="checkCapslock"
                @keyup.enter="handleLogin"
                size="large"
                class="h-[48px] pr-2"
                show-password
              />
            </div>
          </el-form-item>
        </el-tooltip>

        <!-- 验证码 -->
        <el-form-item prop="captchaCode">
          <div class="flex-y-center w-full">
            <svg-icon icon-class="captcha" class="mx-2" />
            <el-input
              v-model="loginData.captchaCode"
              auto-complete="off"
              size="large"
              class="flex-1"
              :placeholder="$t('login.captchaCode')"
              @keyup.enter="handleLogin"
            />

            <el-image
              @click="getCaptcha"
              :src="captchaBase64"
              class="rounded-tr-md rounded-br-md cursor-pointer h-[48px]"
            />
          </div>
        </el-form-item>

        <!-- 登录按钮 -->
        <el-button
          :loading="loading"
          type="primary"
          size="large"
          class="w-full"
          @click.prevent="handleLogin"
          >{{ $t("login.login") }}
        </el-button>

      
      </el-form>
    </el-card>

  
  </div>
</template>

<script setup lang="ts">
import { useSettingsStore, useUserStore } from "@/store";
import { getCaptchaApi , GetVersion } from "@/api/auth";
import { LoginData } from "@/api/auth/types";
import { Sunny, Moon } from "@element-plus/icons-vue";
import { LocationQuery, LocationQueryValue, useRoute } from "vue-router";
import router from "@/router";
import defaultSettings from "@/settings";
import { ThemeEnum } from "@/enums/ThemeEnum";
const logo = new URL("../../assets/logo.png", import.meta.url).href;
// 获取版本号
const version = ref('')  
const fetchVersion = function(){
  GetVersion().then((res) => {
    console.log("Version fetched:", res.data); // 输出返回内容
    version.value = res.data;
  }).catch((error) => {
    console.error("Error fetching version:", error);
  });
}() 



// Stores
const userStore = useUserStore();
const settingsStore = useSettingsStore();

// Internationalization
const { t } = useI18n();

// Reactive states
const isDark = ref(settingsStore.theme === ThemeEnum.DARK);
const icpVisible = ref(true);
const loading = ref(false); // 按钮loading
const isCapslock = ref(false); // 是否大写锁定
const captchaBase64 = ref(); // 验证码图片Base64字符串
const loginFormRef = ref(ElForm); // 登录表单ref
const { height } = useWindowSize();

const loginData = ref<LoginData>({
  username: "",
  password: "",
});

const loginRules = computed(() => {
  return {
    username: [
      {
        required: true,
        trigger: "blur",
        message: t("login.message.username.required"),
      },
    ],
    password: [
      {
        required: true,
        trigger: "blur",
        message: t("login.message.password.required"),
      },
      {
        min: 6,
        message: t("login.message.password.min"),
        trigger: "blur",
      },
    ],
    captchaCode: [
      {
        required: true,
        trigger: "blur",
        message: t("login.message.captchaCode.required"),
      },
    ],
  };
});

/**
 * 获取验证码
 */
function getCaptcha() {
  getCaptchaApi().then(({ data }) => {
    loginData.value.captchaKey = data.captchaKey;
    captchaBase64.value = data.captchaBase64;
  });
}

/**
 * 登录
 */
const route = useRoute();
function handleLogin() {
  loginFormRef.value.validate((valid: boolean) => {
    if (valid) {
      loading.value = true;
      userStore
        .login(loginData.value)
        .then(() => {
          const query: LocationQuery = route.query;
          const redirect = (query.redirect as LocationQueryValue) ?? "/";
          const otherQueryParams = Object.keys(query).reduce(
            (acc: any, cur: string) => {
              if (cur !== "redirect") {
                acc[cur] = query[cur];
              }
              return acc;
            },
            {}
          );

          router.push({ path: redirect, query: otherQueryParams });
        })
        .catch(() => {
          getCaptcha();
        })
        .finally(() => {
          loading.value = false;
        });
    }
  });
}

/**
 * 主题切换
 */

const toggleTheme = () => {
  const newTheme =
    settingsStore.theme === ThemeEnum.DARK ? ThemeEnum.LIGHT : ThemeEnum.DARK;
  settingsStore.changeTheme(newTheme);
};
/**
 * 根据屏幕宽度切换设备模式
 */

watchEffect(() => {
  if (height.value < 600) {
    icpVisible.value = false;
  } else {
    icpVisible.value = true;
  }
});

/**
 * 检查输入大小写
 */
function checkCapslock(event: KeyboardEvent) {
  // 防止浏览器密码自动填充时报错
  if (event instanceof KeyboardEvent) {
    isCapslock.value = event.getModifierState("CapsLock");
  }
}

onMounted(() => {
  getCaptcha();
});
</script>

<style lang="scss" scoped>
html.dark .login-container {
  background:
    radial-gradient(circle at 18% 18%, rgba(93,211,199,.12), transparent 34%),
    radial-gradient(circle at 82% 80%, rgba(93,211,199,.07), transparent 30%),
    var(--ui-canvas);
}

.login-container {
  overflow-y: auto;
  position:relative;
  background:
    radial-gradient(circle at 16% 16%, rgba(15,118,110,.12), transparent 34%),
    radial-gradient(circle at 84% 84%, rgba(15,118,110,.07), transparent 30%),
    var(--ui-canvas);

  @apply wh-full flex-center;

  .login-form {
    padding: 20px 2px 2px;
  }
}

.login-container::before { position:absolute; inset:0; pointer-events:none; opacity:.38; background-image:linear-gradient(var(--ui-border) 1px,transparent 1px),linear-gradient(90deg,var(--ui-border) 1px,transparent 1px); background-size:32px 32px; mask-image:radial-gradient(circle at center,#000,transparent 72%); content:""; }
:deep(.login-panel) { position:relative; z-index:1; max-width:420px; border:1px solid var(--ui-border) !important; border-radius:16px !important; background:color-mix(in srgb,var(--ui-surface-strong) 92%,transparent) !important; box-shadow:0 24px 70px rgba(24,40,31,.14) !important; backdrop-filter:blur(16px); }
:deep(.login-panel .el-card__body) { padding:26px 28px 28px; }
.login-brand { display:grid; grid-template-columns:48px minmax(0,1fr) auto; align-items:center; gap:12px; }
.login-brand img { width:48px; height:48px; border-radius:13px; object-fit:cover; box-shadow:0 8px 20px var(--ui-accent-shadow); }
.login-brand h2 { margin:0; color:var(--ui-text); font-size:19px; line-height:1.2; }
.login-brand p { margin:4px 0 0; color:var(--ui-text-muted); font-size:11px; }
.login-intro { display:flex; align-items:center; gap:8px; margin-top:22px; padding:10px 12px; border:1px solid var(--ui-border); border-radius:9px; background:var(--ui-surface); color:var(--ui-text-secondary); font-size:12px; }
.intro-dot { width:7px; height:7px; border-radius:50%; background:#22a06b; box-shadow:0 0 0 3px rgba(34,160,107,.13); }

.el-form-item {
  margin-bottom:14px;
  padding:0 4px;
  background: var(--ui-surface-strong);
  border: 1px solid var(--ui-border);
  border-radius: 10px;
  transition:border-color 150ms ease,box-shadow 150ms ease,transform 150ms ease;
}
.el-form-item:focus-within { border-color:var(--ui-accent); box-shadow:0 0 0 3px var(--ui-focus-ring); transform:translateY(-1px); }
:deep(.el-button--primary) { height:44px; margin-top:4px; border-radius:9px !important; font-weight:700; box-shadow:0 8px 18px var(--ui-accent-shadow) !important; }

:deep(.el-input) {
  .el-input__wrapper {
    padding: 0;
    background-color: transparent;
    box-shadow: none;

    &.is-focus,
    &:hover {
      box-shadow: none !important;
    }

    input:-webkit-autofill {
      /* 通过延时渲染背景色变相去除背景颜色 */
      transition: background-color 1000s ease-in-out 0s;
    }
  }
}
</style>
