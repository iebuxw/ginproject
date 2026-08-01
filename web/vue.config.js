module.exports = {
  devServer: {
    port: 3000,
    proxy: {
      '/api': { target: 'http://localhost:8000', changeOrigin: true }
    }
  },
  outputDir: 'dist',
  publicPath: '/'
}
