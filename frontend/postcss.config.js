module.exports = {
  plugins: {
    autoprefixer: {},
    // Disable postcss-calc to prevent warnings with Vuetify's max-content usage
    'postcss-calc': false
  }
}
