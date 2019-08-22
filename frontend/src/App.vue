<template>
  <v-app>
    <v-toolbar app dense>
      <v-toolbar-title class="headline">
        <span>etcd browser</span>
        <span class="font-weight-light"></span>
      </v-toolbar-title>
    </v-toolbar>

    <v-content>
      <v-layout text-md wrap>
        <v-flex xs6>
          <tree-item
            class="the-tree"
            :item="treeRoot"
            :load-children="loadSubtree"
            @active="active"
          />
        </v-flex>
        <v-flex d-flex class="right-sticky">
          <div
            v-if="!activeItemId"
            class="title grey--text text--lighten-1 font-weight-light pt-3 pl-1"
          >Select a key</div>
          <v-card v-else class="pt-3 pl-1 text-xs-left" flat>
            <h4 class="mono mb-2">{{ activeItemId }}:</h4>
            <pre class="mono mb-2">{{ activeItemValue }}</pre>
          </v-card>
        </v-flex>
      </v-layout>
    </v-content>
  </v-app>
</template>

<script>
import treeItem from "./components/TreeItem";

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
  name: "App",
  components: {
    treeItem
  },
  data() {
    return {
      treeRoot: {
        id: "",
        name: "<root>",
        isRoot: true,
        hasValue: false,
        children: [],
        childrenMap: new Map()
      },
      activeItemId: null,
      activeItemValue: null
    };
  },
  mounted() {
    this.wsconnect();
  },
  methods: {
    loadSubtree: async function(item) {
      // console.log(item.children);
      // await new Promise(resolve => setTimeout(resolve, 400));
      return fetch(
        process.env.VUE_APP_ROOT_API +
          "/api/list?k=" +
          encodeURIComponent(item.id)
      )
        .then(res => res.json())
        .then(json => {
          lastRev = json.rev;
          json.keys.forEach(s => {
            var el = { name: s.k, id: item.id + s.k, hasValue: !!(s.t & 1) };
            if (s.t & 2) {
              el.children = [];
              el.childrenMap = new Map();
            }
            item.children.push(el);
            item.childrenMap.set(s.k, el);
          });
        })
        .catch(err => {
          console.warn(err);
          item.name = "error accessing etcd!";
        });
    },
    active: function(item) {
      // console.log("active: ", item.id);
      this.activeItemValue = "";
      this.activeItemId = item.id;
      var vm = this;
      if (item.hasValue) {
        fetch(
          process.env.VUE_APP_ROOT_API +
            "/api/kv?k=" +
            encodeURIComponent(item.id)
        )
          .then(res => res.text())
          .then(text => {
            vm.activeItemValue = text;
          })
          .catch(err => console.warn(err));
        if (socket) {
          socket.send(JSON.stringify({ key: item.id }));
        }
      }
    },
    wsconnect() {
      // return;
      var vm = this;
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
        var path = msg.key.split(/((?:^\/+[^/]+|[^/]*)\/)/).filter(x => x);
        var root = vm.treeRoot;
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
        if (msg.key === vm.activeItemId) {
          vm.activeItemValue = msg.value;
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
          if (item === undefined && root !== undefined) {
            var el = {
              name: lastId,
              id: root.id + lastId,
              hasValue: true
            };
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
          setTimeout(vm.wsconnect, 2000);
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
  width: 45%;
  position: fixed;
  right: 10px;
}
.right-sticky pre {
  white-space: pre-wrap;
  word-wrap: anywhere;
}
.the-tree {
  margin: 10px 0 0 -10px;
}
</style>
