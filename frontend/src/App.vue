<template>
  <v-app>
    <v-toolbar app>
      <v-toolbar-title class="headline">
        <span>etcdv3-browser</span>
        <span class="font-weight-light"></span>
      </v-toolbar-title>
    </v-toolbar>

    <v-content>
      <Browser/>
    </v-content>
  </v-app>
</template>

<script>
import Browser from './components/Browser'

export default {
  name: 'App',
  components: {
    Browser
  },
  data () {
    return {
      //
    }
  }
}
</script>
