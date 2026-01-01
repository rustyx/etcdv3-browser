module.exports = {
  chainWebpack: config => {
    // Increase performance limits to suppress bundle size warnings
    config.performance
      .maxEntrypointSize(2000000)
      .maxAssetSize(1500000);
    
    // Configure CSS minimizer to disable postcss-calc to prevent Vuetify max-content warnings
    config.optimization.minimizer('css').tap(args => {
      args[0].minimizerOptions = args[0].minimizerOptions || {};
      args[0].minimizerOptions.preset = [
        'default',
        {
          calc: false, // Disable calc optimization to prevent max-content warnings
        }
      ];
      return args;
    });
    
    if (process.env.NODE_ENV === 'test' || process.env.npm_lifecycle_event === 'test:unit') {
      const sassRule = config.module.rule('sass');
      sassRule.uses.clear();
      sassRule.use('null-loader').loader('null-loader');
      sassRule.oneOf('normal').uses.clear();
      sassRule.oneOf('normal').use('null-loader').loader('null-loader');
      config.merge({
        devtool: 'eval-cheap-module-source-map',
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