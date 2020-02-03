module.exports = {
  chainWebpack: config => {
    config.performance
      .maxEntrypointSize(1000000)
      .maxAssetSize(800000);
    if (process.env.NODE_ENV === 'test' || process.env.npm_lifecycle_event === 'test:unit') {
      const sassRule = config.module.rule('sass');
      sassRule.uses.clear();
      sassRule.use('null-loader').loader('null-loader');
      sassRule.oneOf('normal').uses.clear();
      sassRule.oneOf('normal').use('null-loader').loader('null-loader');
      config.merge({
        devtool: 'cheap-module-eval-source-map',
      });
      /** To debug test:unit, use the following launch configuration:
        {
            "type": "node",
            "request": "launch",
            "name": "etcdv3 test:unit debug",
            "cwd": "${workspaceFolder}/frontend",
            "program": "${workspaceFolder}/frontend/node_modules/@vue/cli-service/bin/vue-cli-service.js",
            "args": ["test:unit", "--inspect-brk", "--watch", "--timeout", "900000"],
            "port": 9229
        }
      */
    }
  }
}