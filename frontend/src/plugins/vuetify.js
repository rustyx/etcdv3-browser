import Vue from 'vue';
import 'material-design-icons-iconfont/dist/material-design-icons.css';
import Vuetify from 'vuetify/lib';

Vue.use(Vuetify, {
  iconfont: 'md',
});

export default new Vuetify({
  theme: {
    dark: false,
    iconfont: 'md',
  }
});
