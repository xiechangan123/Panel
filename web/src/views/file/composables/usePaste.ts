import file from '@/api/panel/file'
import { useFileStore } from '@/store'
import { useGettext } from 'vue3-gettext'

export function usePaste() {
  const { $gettext } = useGettext()
  const fileStore = useFileStore()

  const handlePaste = (targetPath: string) => {
    const { marked, markedType } = fileStore.clipboard
    if (!marked.length) {
      window.$message.error($gettext('Please mark the files/folders to copy or move first'))
      return
    }

    const paths = marked.map((item) => ({
      name: item.name,
      source: item.source,
      target: targetPath + '/' + item.name,
      force: false
    }))
    const targets = paths.map((item) => item.target)

    useRequest(file.exist(targets)).onSuccess(({ data }) => {
      let hasConflict = false
      for (let i = 0; i < data.length; i++) {
        if (data[i]) {
          hasConflict = true
          const pathItem = paths[i]
          if (pathItem) pathItem.force = true
        }
      }

      const executePaste = () => {
        const request = markedType === 'copy' ? file.copy(paths) : file.move(paths)
        const successMsg =
          markedType === 'copy' ? $gettext('Copied successfully') : $gettext('Moved successfully')
        useRequest(request).onSuccess(() => {
          fileStore.clearClipboard()
          window.$bus.emit('file:refresh')
          window.$message.success(successMsg)
        })
      }

      if (hasConflict) {
        window.$dialog.warning({
          title: $gettext('Warning'),
          content: $gettext(
            'There are items with the same name %{ items } Do you want to overwrite?',
            {
              items: paths
                .filter((item) => item.force)
                .map((item) => item.name)
                .join(', ')
            }
          ),
          positiveText: $gettext('Overwrite'),
          negativeText: $gettext('Cancel'),
          onPositiveClick: executePaste,
          onNegativeClick: () => {
            window.$message.info($gettext('Canceled'))
          }
        })
      } else {
        executePaste()
      }
    })
  }

  return { handlePaste }
}
