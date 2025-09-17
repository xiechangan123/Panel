export interface File {
  path: string
  keyword: string
  sub: boolean
}

export const useFileStore = defineStore('file', {
  state: (): File => {
    return {
      path: '/opt',
      keyword: '',
      sub: false
    }
  },
  actions: {
    set(info: File) {
      this.path = info.path
      this.keyword = info.keyword
      this.sub = info.sub
    }
  },
  persist: true
})
