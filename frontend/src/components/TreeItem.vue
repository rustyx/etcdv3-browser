<template>
  <li class="item" :class="{folder: isFolder, err: item.isError}">
    <div v-if="!item.isRoot" @click="toggle">
      <span class="expand-icon">
        <v-icon v-if="isFolder" :class="{open: isOpen}">arrow_drop_down</v-icon>
      </span>
      <span class="name">{{ item.name }}</span>
    </div>
    <ul v-show="isOpen" v-if="isFolder">
      <span v-show="loading && !item.children.length">
        <v-icon class="loading-icon">loading</v-icon>
      </span>
      <span v-show="item.isRoot && !loading && !item.children.length">No entries found</span>
      <tree-item
        v-for="(child, index) in item.children"
        :key="index"
        :item="child"
        :load-children="loadChildren"
        :parent-open="isOpen"
        @active="$emit('active', $event)"
      ></tree-item>
    </ul>
  </li>
</template>

<script>
export default {
  name: "tree-item",
  props: {
    item: Object,
    loadChildren: Function,
    parentOpen: Boolean
  },
  data: () => ({
    isOpen: false,
    loading: false
  }),
  computed: {
    isFolder: function() {
      return this.item.children !== undefined;
    }
  },
  mounted() {
    if (this.item.isRoot) {
      this.toggle();
    }
  },
  watch: {
    parentOpen: function() {
      if (!this.parentOpen) {
        // recursively close
        this.isOpen = false;
      }
    }
  },
  methods: {
    toggle: async function() {
      this.$emit("active", this.item);
      if (this.isFolder) {
        this.item.children.length = 0;
        this.isOpen = !this.isOpen;
        if (this.isOpen) {
          this.loading = true;
          try {
            // this.item.children =
            await this.loadChildren(this.item);
            this.loading = false;
          } catch (e) {
            this.item.children.push({
              name: "error: " + e,
              isError: true
            });
            this.loading = false;
            throw e;
          }
        }
      }
    }
  }
};
</script>

<style>
.item {
  cursor: pointer;
  font-size: 17px;
}
.item span {
  vertical-align: top;
}
.expand-icon {
  min-width: 25px;
  display: inline-block;
}
.expand-icon .v-icon {
  transform: rotate(-90deg);
}
.expand-icon .v-icon.open {
  transform: rotate(0deg);
}
.loading-icon {
  margin-left: 20px;
  animation: progress-circular-rotate 1s linear infinite;
}
.err > div {
  color: red;
  cursor: default;
}
ul {
  padding-left: 1em;
  line-height: 1.5em;
  list-style-type: none;
}
</style>
