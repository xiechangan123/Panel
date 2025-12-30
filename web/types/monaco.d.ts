// Monaco Editor 本地化模块声明
declare module 'monaco-editor/esm/nls.messages.cs.js'
declare module 'monaco-editor/esm/nls.messages.de.js'
declare module 'monaco-editor/esm/nls.messages.es.js'
declare module 'monaco-editor/esm/nls.messages.fr.js'
declare module 'monaco-editor/esm/nls.messages.it.js'
declare module 'monaco-editor/esm/nls.messages.ja.js'
declare module 'monaco-editor/esm/nls.messages.ko.js'
declare module 'monaco-editor/esm/nls.messages.pl.js'
declare module 'monaco-editor/esm/nls.messages.pt-br.js'
declare module 'monaco-editor/esm/nls.messages.ru.js'
declare module 'monaco-editor/esm/nls.messages.tr.js'
declare module 'monaco-editor/esm/nls.messages.zh-cn.js'
declare module 'monaco-editor/esm/nls.messages.zh-tw.js'

// Monaco Editor Worker 模块声明
declare module 'monaco-editor/esm/vs/editor/editor.worker?worker' {
  const EditorWorker: new () => Worker
  export default EditorWorker
}
declare module 'monaco-editor/esm/vs/language/json/json.worker?worker' {
  const JsonWorker: new () => Worker
  export default JsonWorker
}
declare module 'monaco-editor/esm/vs/language/css/css.worker?worker' {
  const CssWorker: new () => Worker
  export default CssWorker
}
declare module 'monaco-editor/esm/vs/language/html/html.worker?worker' {
  const HtmlWorker: new () => Worker
  export default HtmlWorker
}
declare module 'monaco-editor/esm/vs/language/typescript/ts.worker?worker' {
  const TsWorker: new () => Worker
  export default TsWorker
}
