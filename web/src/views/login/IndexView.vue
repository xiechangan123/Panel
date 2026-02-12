<script lang="ts" setup>
import user from '@/api/panel/user'
import bgImg from '@/assets/images/login_bg.webp'
import logoImg from '@/assets/images/logo.svg'
import { addDynamicRoutes } from '@/router'
import { useThemeStore, useUserStore } from '@/store'
import { getLocal, removeLocal, setLocal } from '@/utils'
import { rsaEncrypt } from '@/utils/encrypt'
import { useGettext } from 'vue3-gettext'

const { $gettext } = useGettext()
const router = useRouter()
const route = useRoute()
const query = route.query
const { data: key, loading: isLoading } = useRequest(user.key, { initialData: '' })
const { data: isLogin } = useRequest(user.isLogin, { initialData: false })

interface LoginInfo {
  username: string
  password: string
  safe_login: boolean
  pass_code: string
  captcha_code: string
}

const loginInfo = ref<LoginInfo>({
  username: '',
  password: '',
  safe_login: true,
  pass_code: '',
  captcha_code: ''
})

const localLoginInfo = getLocal('loginInfo') as LoginInfo
if (localLoginInfo) {
  loginInfo.value.username = localLoginInfo.username || ''
  loginInfo.value.password = localLoginInfo.password || ''
}

const userStore = useUserStore()
const themeStore = useThemeStore()
const logining = ref(false)
const isRemember = useStorage('isRemember', false)
const showTwoFA = ref(false)
const captchaRequired = ref(false)
const captchaImage = ref('')

const logo = computed(() => themeStore.logo || logoImg)

// 刷新验证码
const refreshCaptcha = () => {
  useRequest(user.captcha())
    .onSuccess(({ data }) => {
      captchaRequired.value = data.required
      captchaImage.value = 'data:image/png;base64,' + data.image || ''
      loginInfo.value.captcha_code = ''
    })
    .onError(() => {
      captchaRequired.value = false
      captchaImage.value = ''
    })
}

async function handleLogin() {
  const { username, password, pass_code, safe_login, captcha_code } = loginInfo.value
  if (!username || !password) {
    window.$message.warning($gettext('Please enter username and password'))
    return
  }
  const trimmedCaptcha = captcha_code?.trim() || ''
  if (captchaRequired.value && !trimmedCaptcha) {
    window.$message.warning($gettext('Please enter captcha code'))
    return
  }
  logining.value = true
  // 等待公钥加载完成（密码管理器可能在公钥就绪前自动提交）
  if (isLoading.value) {
    await new Promise<void>((resolve) => {
      const stop = watch(isLoading, (v) => {
        if (!v) {
          stop()
          resolve()
        }
      })
    })
  }
  if (!key.value) {
    logining.value = false
    window.$message.warning(
      $gettext('Failed to get encryption public key, please refresh the page and try again')
    )
    return
  }
  useRequest(
    user.login(
      rsaEncrypt(username, String(unref(key))),
      rsaEncrypt(password, String(unref(key))),
      pass_code,
      safe_login,
      trimmedCaptcha
    )
  )
    .onSuccess(async () => {
      window.$notification?.success({ title: $gettext('Login successful!'), duration: 2500 })
      if (isRemember.value) {
        setLocal('loginInfo', { username, password })
      } else {
        removeLocal('loginInfo')
      }

      await addDynamicRoutes()
      useRequest(user.info()).onSuccess(({ data }) => {
        userStore.set(data as any)
      })
      if (query.redirect) {
        const path = query.redirect as string
        Reflect.deleteProperty(query, 'redirect')
        await router.push({ path, query })
      } else {
        await router.push('/')
      }
    })
    .onError(() => {
      // 登录失败后刷新验证码状态
      refreshCaptcha()
    })
    .onComplete(() => {
      logining.value = false
    })
}

const isTwoFA = () => {
  const { username } = loginInfo.value
  if (!username) {
    return
  }
  useRequest(user.isTwoFA(username))
    .onSuccess(({ data }) => {
      showTwoFA.value = Boolean(data)
    })
    .onError(() => {
      showTwoFA.value = false
    })
}

watch(isLogin, async () => {
  if (isLogin) {
    await addDynamicRoutes()
    useRequest(user.info()).onSuccess(({ data }) => {
      userStore.set(data as any)
    })
    if (query.redirect) {
      const path = query.redirect as string
      Reflect.deleteProperty(query, 'redirect')
      await router.push({ path, query })
    } else {
      await router.push('/')
    }
  }
})

onMounted(() => {
  refreshCaptcha()
})
</script>

<template>
  <AppPage :show-footer="true" :style="{ backgroundImage: `url(${bgImg})` }" bg-cover>
    <div m-auto flex flex-col items-center>
      <!-- Logo -->
      <n-image :src="logo" preview-disabled mb-22 h-80 w-80 object-contain />

      <!-- 登录卡片 -->
      <div px-28 py-32 rounded-lg bg-white min-w-380 card-shadow class="dark:bg-dark">
        <h2 text-32 font-600 mb-28 text-center>{{ themeStore.name }}</h2>

        <n-input
          v-model:value="loginInfo.username"
          :maxlength="32"
          :placeholder="$gettext('Username')"
          autofocus
          class="text-15 h-48 items-center"
          :on-blur="isTwoFA"
        />

        <n-input
          v-model:value="loginInfo.password"
          :maxlength="32"
          :placeholder="$gettext('Password')"
          class="text-15 mt-20 h-48 items-center"
          type="password"
          show-password-on="click"
          @keydown.enter="handleLogin"
        />

        <n-input
          v-if="showTwoFA"
          v-model:value="loginInfo.pass_code"
          :maxlength="6"
          :placeholder="$gettext('2FA Code')"
          class="text-15 mt-20 h-48 items-center"
          type="text"
          @keydown.enter="handleLogin"
        />

        <n-flex v-if="captchaRequired" align="center" class="mt-20">
          <n-input
            v-model:value="loginInfo.captcha_code"
            :maxlength="4"
            :placeholder="$gettext('Captcha Code')"
            class="text-15 h-48 items-center"
            style="flex: 1"
            type="text"
            @keydown.enter="handleLogin"
          />
          <n-image
            :src="captchaImage"
            preview-disabled
            class="rounded h-48 cursor-pointer"
            @click="refreshCaptcha"
          />
        </n-flex>

        <n-flex class="mt-20">
          <n-checkbox v-model:checked="loginInfo.safe_login" :label="$gettext('Safe Login')" />
          <n-checkbox v-model:checked="isRemember" :label="$gettext('Remember Me')" />
        </n-flex>

        <n-button
          :loading="isLoading || logining"
          :disabled="isLoading || logining"
          class="text-16 mt-24 h-48 w-full"
          type="primary"
          @click="handleLogin"
        >
          {{ $gettext('Login') }}
        </n-button>
      </div>
    </div>
  </AppPage>
</template>
