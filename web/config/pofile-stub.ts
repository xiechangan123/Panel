// vue3-gettext 的入口静态引入了 pofile，但它只被 CLI 提取命令用到
// pofile 依赖 node:fs 无法在浏览器运行，构建时用空实现替换，避免打入产物
export default {}
