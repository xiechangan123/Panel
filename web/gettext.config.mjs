export default {
  input: {
    include: ['**/*.js', '**/*.ts', '**/*.vue'],
    exclude: ['utils/gettext/**'],
    jsExtractorOpts: [
      {
        keyword: '__', // $gettext
        options: {
          content: {
            replaceNewLines: '\n'
          },
          arguments: {
            text: 0
          }
        }
      },
      {
        keyword: '_n', // $ngettext
        options: {
          content: {
            replaceNewLines: '\n'
          },
          arguments: {
            text: 0,
            textPlural: 1
          }
        }
      },
      {
        keyword: '_x', // $pgettext
        options: {
          content: {
            replaceNewLines: '\n'
          },
          arguments: {
            context: 0,
            text: 1
          }
        }
      },
      {
        keyword: '_nx', // $npgettext
        options: {
          content: {
            replaceNewLines: '\n'
          },
          arguments: {
            context: 0,
            text: 1,
            textPlural: 2
          }
        }
      }
    ]
  },
  output: {
    path: './src/locales',
    potPath: './frontend.pot',
    locales: ['en', 'zh_CN', 'zh_TW'],
    linguas: false
  }
}
