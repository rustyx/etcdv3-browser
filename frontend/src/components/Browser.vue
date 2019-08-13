<template>
  <v-layout text-md wrap>
    <v-flex xs6>
      <v-treeview
        :active.sync="active"
        :items="items"
        :load-children="loadSubtree"
        :open.sync="open"
        activatable
        active-class="primary--text"
        open-on-click
        class="pt-1"
      ></v-treeview>
    </v-flex>
    <v-flex d-flex class="right-sticky">
      <div
        v-if="!active.length"
        class="title grey--text text--lighten-1 font-weight-light pt-3 pl-1"
      >Select a key</div>
      <v-card v-else class="pt-3 pl-1 text-xs-left" flat>
        <h4 class="mono mb-2">{{ active[0] }}:</h4>
        <pre class="mono mb-2">{{ selected }}</pre>
      </v-card>
    </v-flex>
  </v-layout>
</template>

<script>
//const pause = ms => new Promise(resolve => setTimeout(resolve, ms))
var wsuri = process.env.VUE_APP_ROOT_WS;
if (wsuri == "") {
  var loc = window.location;
  wsuri = (loc.protocol === "http:" ? "ws:" : "wss:") + "//" + loc.host;
}
wsuri += "/api/kvws?rev=";
var lastRev = 0; // TODO: implement lastRev on server side
var wsConnectRetry = 0;
var socket;
export default {
  data: () => ({
    active: [],
    open: [""],
    items: [
      {
        id: "",
        name: "<etcd root>",
        loaded: false,
        children: [],
        childrenMap: new Map()
      }
    ],
    selected: null
  }),
  computed: {},
  mounted() {
    this.wsconnect();
  },
  watch: {
    active: async function(items) {
      this.selected = "";
      if (items.length) {
        fetch(
          process.env.VUE_APP_ROOT_API +
            "/api/kv?k=" +
            encodeURIComponent(items[0])
        )
          .then(res => res.text())
          .then(text => {
            this.selected = text;
          })
          .catch(err => console.warn(err));
        if (socket) {
          socket.send(JSON.stringify({ key: items[0] }));
        }
      }
    }
  },
  methods: {
    async loadSubtree(item) {
      return fetch(
        process.env.VUE_APP_ROOT_API +
          "/api/list?k=" +
          encodeURIComponent(item.id)
      )
        .then(res => res.json())
        .then(json => {
          lastRev = json.rev;
          json.keys.forEach(s => {
            var el = { name: s.k, id: item.id + s.k, loaded: false };
            if (s.t & 2) {
              el.children = [];
              el.childrenMap = new Map();
            }
            item.children.push(el);
            item.childrenMap.set(s.k, el);
          });
          item.loaded = true;
        })
        .catch(err => {
          console.warn(err);
          item.name = "error accessing etcd!";
        });
    },
    wsconnect() {
      var self = this;
      if (++wsConnectRetry > 20) {
        return;
      }
      socket = new WebSocket(wsuri + lastRev);
      socket.onopen = function() {
        console.log("[ws] Connected");
        wsConnectRetry = 0;
      };
      socket.onmessage = function(event) {
        var msg = JSON.parse(event.data);
        if (!msg.rev || !msg.key) {
          return;
        }
        lastRev = msg.rev;
        var path = msg.key.split(/([^/]*\/)/).filter(x => x);
        var root = self.items[0];
        var item = root;
        var lastId = "";
        var depth = 0;
        path.every(function(s) {
          // descend down the tree, matching subelements
          depth++;
          root = item;
          if (root !== undefined) {
            if (root.childrenMap === undefined) {
              root.children = [];
              root.childrenMap = new Map();
            }
            item = root.childrenMap.get(s);
          }
          lastId = s;
          return item !== undefined;
        });
        if (msg.key === self.active[0]) {
          self.selected = msg.value;
        }
        if (msg.deleted) {
          if (item !== undefined) {
            root.children.splice(
              root.children.findIndex(el => el.name === item.name),
              1
            );
            root.childrenMap.delete(item.name);
          }
        } else {
          if (item === undefined && root !== undefined && root.loaded) {
            var el = { name: lastId, id: root.id + lastId, loaded: false };
            if (depth < path.length) {
              el.children = [];
              el.childrenMap = new Map();
            }
            root.children.push(el);
            root.childrenMap.set(lastId, el);
          }
        }
      };
      socket.onclose = function(event) {
        console.log(
          `[ws] Disconnected, code=${event.code} reason=${event.reason}`
        );
        if (!event.wasClean) {
          setTimeout(self.wsconnect, 2000);
        }
      };
      socket.onerror = function(error) {
        console.log(`[ws] ${error.message}`);
      };
    }
  }
};
</script>

<style>
.mono {
  font-family: "Courier New", Courier, monospace;
  overflow-x: auto;
  white-space: pre-wrap;
  word-wrap: break-word;
}
.right-sticky {
  width: 40%;
  position: fixed;
  right: 10px;
}
</style>
