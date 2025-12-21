import 'vuetify/styles';
import { createVuetify } from 'vuetify';
import * as components from 'vuetify/components';
import * as directives from 'vuetify/directives';
import { md } from 'vuetify/iconsets/md';

export default createVuetify({
  components,
  directives,
  icons: {
    defaultSet: 'md',
    sets: {
      md,
    },
  },
  theme: {
    defaultTheme: 'light',
    themes: {
      light: {
        colors: {
          primary: '#1976D2',
          secondary: '#424242',
          accent: '#82B1FF',
          'button-bg': '#EEEEEE',
        },
      },
      dark: {
        colors: {
          primary: '#2196F3',
          secondary: '#424242',
          accent: '#FF4081',
          'button-bg': '#424242',
        },
      },
    },
  },
});
