export default {
  input: {
    include: ['**/*.js', '**/*.ts', '**/*.vue'],
    exclude: ['utils/gettext/**']
  },
  output: {
    path: './src/locales',
    potPath: './frontend.pot',
    locales: ['en', 'zh_CN', 'zh_TW'],
    linguas: false
  }
}
