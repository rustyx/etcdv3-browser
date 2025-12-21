<template>
  <v-app>
    <v-app-bar app dense>
      <v-toolbar-title class="headline" @click="clearActiveItem" style="cursor: pointer;">
        <span>etcd browser</span>
      </v-toolbar-title>
      <v-spacer></v-spacer>
      <div class="app-bar-btn">
        <v-switch v-model="dark"></v-switch>
      </div>
    </v-app-bar>

    <v-main>
      <v-container fluid>
        <v-row>
          <v-col cols="12" md="6" class="etcd-tree">
            <tree-item class="the-tree" :item="treeRoot" :load-children="loadSubtree" @active="active" />
          </v-col>
          <v-col cols="12" md="6" class="right-sticky" :class="{ 'has-content': activeItemId }">
            <div class="details-wrapper">
              <div class="details-content">
                <div v-if="!activeItemId" class="title text-grey font-weight-light pt-3 pl-1">Select a key</div>
                <v-card v-else class="pt-3 pl-1 pr-1" flat>
                  <h4 class="mono mb-2">{{ activeItemId }}:</h4>
                  <pre class="mono mb-2">{{ activeItemValue }}</pre>
                </v-card>
              </div>
              <v-card-actions v-if="editable" class="details-actions">
                <v-btn variant="elevated" color="button-bg" @click.stop="btnAdd">Add</v-btn>
                <v-btn variant="elevated" color="button-bg" v-show="!!activeItemId" @click.stop="btnEdit">Edit</v-btn>
                <v-btn variant="elevated" color="button-bg" v-show="!!activeItemId" @click.stop="btnDelete">Delete</v-btn>
              </v-card-actions>
            </div>
          </v-col>
        </v-row>
      </v-container>
    </v-main>

    <v-dialog v-model="editDialogOpen" max-width="720" @keydown.esc="editDialogOpen = false" @keydown="handleEditDialogKeydown">
      <v-form v-model="editFormValid" @submit.prevent>
        <v-card>
          <v-container>
            <v-row>
              <v-col :cols="12">
                <v-text-field label="Key" v-model="editKey" mandatory :rules="[notBlank]" @keydown.enter.prevent="focusEditValue" ref="editKeyField" />
              </v-col>
              <v-col :cols="12">
                <v-textarea label="Value" v-model="editValue" rows="8" ref="editValueField"></v-textarea>
              </v-col>
            </v-row>
          </v-container>
          <v-card-actions>
            <div class="flex-grow-1"></div>
            <v-btn variant="elevated" color="button-bg" type="submit" @click.stop="btnSave()" :loading="saveInProgress">
              {{ editKey !== "" && editKey === activeItemId ? "Save" : "Add"}}</v-btn>
            <v-btn variant="elevated" color="button-bg" @click="editDialogOpen = false">Cancel</v-btn>
          </v-card-actions>
          <v-alert v-model="showSaveError" type="error" closable>{{ saveError }}</v-alert>
        </v-card>
      </v-form>
    </v-dialog>

    <v-dialog v-model="deleteDialogOpen" max-width="600" @keydown.esc="deleteDialogOpen = false" @keydown.enter="btnDoDelete">
      <v-form v-model="editFormValid" @submit.prevent>
        <v-card>
          <v-card-title>Delete {{editKey}}?</v-card-title>
          <v-card-actions>
            <div class="flex-grow-1"></div>
            <v-btn variant="elevated" color="button-bg" type="submit" @click.stop="btnDoDelete" :loading="saveInProgress">Delete</v-btn>
            <v-btn variant="elevated" color="button-bg" @click="deleteDialogOpen = false">Cancel</v-btn>
          </v-card-actions>
          <v-alert v-model="showSaveError" type="error" closable>{{ saveError }}</v-alert>
        </v-card>
      </v-form>
    </v-dialog>
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
      editable: false,
      editDialogOpen: false,
      editFormValid: false,
      deleteDialogOpen: false,
      saveInProgress: false,
      saveError: "",
      showSaveError: false,
      editKey: "",
      editValue: "",
      activeItemId: null,
      activeItemValue: null
    };
  },
  computed: {
    dark: {
      get: function() {
        return this.$vuetify.theme.global.name === 'dark';
      },
      set: function(v) {
        this.$vuetify.theme.global.name = v ? 'dark' : 'light';
        this.setCookie("dark", v, 3650);
      }
    }
  },
  mounted() {
    if (this.getCookie("dark")) {
      this.$vuetify.theme.global.name = 'dark';
    }
    this.wsconnect();
  },
  methods: {
    loadSubtree: async function(item) {
      // console.log(item.children);
      // await new Promise(resolve => setTimeout(resolve, 400));
      return fetch(process.env.VUE_APP_ROOT_API + "/api/list?k=" + encodeURIComponent(item.id))
        .then(res => res.json())
        .then(json => {
          lastRev = json.rev;
          if (item.id === "") {
            this.editable = !!json.editable;
          }
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
          console.warn(err); // eslint-disable-line no-console
          item.name = "error accessing etcd!";
        });
    },
    clearActiveItem: function() {
      this.activeItemId = null;
      this.activeItemValue = null;
    },
    active: function(item) {
      // console.log("active: ", item.id);
      this.activeItemValue = "";
      this.activeItemId = item.id;
      var vm = this;
      if (item.hasValue) {
        fetch(process.env.VUE_APP_ROOT_API + "/api/kv?k=" + encodeURIComponent(item.id))
          .then(res => res.text())
          .then(text => {
            vm.activeItemValue = text;
          })
          .catch(err => console.warn(err)); // eslint-disable-line no-console
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
        console.log("[ws] Connected"); // eslint-disable-line no-console
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
        console.log(`[ws] Disconnected, code=${event.code} reason=${event.reason}`); // eslint-disable-line no-console
        if (!event.wasClean) {
          setTimeout(vm.wsconnect, 2000);
        }
      };
      socket.onerror = function(error) {
        console.log(`[ws] ${error.message}`); // eslint-disable-line no-console
      };
    },
    btnAdd() {
      this.editKey = "";
      this.editValue = "";
      this.saveInProgress = false;
      this.saveError = "";
      this.showSaveError = false;
      this.editDialogOpen = true;
    },
    handleEditDialogKeydown(event) {
      if (event.key === 'Enter' && event.ctrlKey) {
        event.preventDefault();
        this.btnSave();
      }
    },
    focusEditValue() {
      this.$nextTick(() => {
        if (this.$refs.editValueField) {
          this.$refs.editValueField.focus();
        }
      });
    },
    btnEdit() {
      this.editKey = this.activeItemId;
      this.editValue = this.activeItemValue;
      this.saveInProgress = false;
      this.saveError = "";
      this.showSaveError = false;
      this.editDialogOpen = true;
    },
    notBlank(s) {
      return !!s || "This field is required";
    },
    btnDelete() {
      this.editKey = this.activeItemId;
      this.saveInProgress = false;
      this.saveError = "";
      this.showSaveError = false;
      this.deleteDialogOpen = true;
    },
    btnSave() {
      if (!this.editFormValid) return;
      this.saveError = "";
      this.showSaveError = false;
      fetch(process.env.VUE_APP_ROOT_API + "/api/kv?k=" + encodeURIComponent(this.editKey), {
        method: "POST",
        headers: {
          "Content-Type": "application/binary"
        },
        body: this.editValue
      })
        .then(res => {
          if (!res.ok) {
            this.saveError = res.status + " " + res.statusText;
            this.showSaveError = true;
            return;
          }
          this.editDialogOpen = false;
        })
        .catch(err => {
          this.saveError = err.message;
          this.showSaveError = true;
        })
        .then(() => {
          this.saveInProgress = false;
        });
    },
    btnDoDelete() {
      if (!this.editFormValid) return;
      this.saveError = "";
      this.showSaveError = false;
      fetch(process.env.VUE_APP_ROOT_API + "/api/kv?k=" + encodeURIComponent(this.editKey), {
        method: "DELETE"
      })
        .then(res => {
          if (!res.ok) {
            this.saveError = res.status + " " + res.statusText;
            this.showSaveError = true;
            return;
          }
          this.activeItemId = null;
          this.editDialogOpen = false;
        })
        .catch(err => {
          this.saveError = err.message;
          this.showSaveError = true;
        })
        .then(() => {
          this.deleteDialogOpen = false;
        });
    },
    setCookie(name, value, days) {
      var date = new Date();
      date.setTime(date.getTime() + days * 24 * 60 * 60 * 1000);
      document.cookie =
        name +
        "=" +
        encodeURIComponent(value || "") +
        "; expires=" +
        date.toUTCString() +
        "; path=/";
    },
    getCookie(name) {
      var nameEQ = name + "=";
      var ca = document.cookie.split(";");
      for (var i = 0; i < ca.length; i++) {
        var c = ca[i];
        while (c.charAt(0) == " ") {
          c = c.substring(1, c.length);
        }
        if (c.indexOf(nameEQ) == 0) {
          return decodeURIComponent(c.substring(nameEQ.length, c.length));
        }
      }
      return null;
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
/* workaround to align the theme switch in the app bar */
#app .app-bar-btn {
  padding: 24px 16px 0px 12px;
}

.right-sticky .v-card pre {
    max-height: calc(80vh - 140px);
    overflow-y: auto;
    border: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
    border-radius: 4px;
    padding: 8px;
    background: rgba(var(--v-theme-on-surface), 0.02);
}

@media (max-width: 960px) {
  .right-sticky {
    position: fixed;
    bottom: 0;
    left: 0;
    right: 0;
    width: 100%;
    min-height: 140px;
    height: 32vh;
    max-height: 600px;
    z-index: 10;
    background: rgb(var(--v-theme-surface));
    box-shadow: 0 -4px 12px rgba(0, 0, 0, 0.15);
    border-top-left-radius: 12px;
    border-top-right-radius: 12px;
    padding: 12px 16px;
    margin: 0;
    overflow-y: auto;
    transition: height 0.3s ease;
    display: block;
  }

  .right-sticky.has-content {
    height: 50vh;
  }

  .right-sticky.has-content .details-content {
    max-height: calc(50vh - 140px);
  }

  .right-sticky .v-card pre {
    max-height: 12vh;
  }

  .right-sticky .details-actions {
    display: flex;
    gap: 8px;
    padding: 12px 0;
  }

  .the-tree {
    padding-bottom: calc(50vh + 20px);
  }
}

@media (max-width: 600px) {
  .v-dialog {
    margin: 16px;
  }

  .v-dialog .v-card {
    width: 100%;
  }

  .v-dialog .v-container {
    padding: 12px;
  }

  .v-dialog .v-card-actions {
    padding: 8px 12px;
  }

  .v-dialog .v-card-actions .v-btn {
    flex: 1;
    min-width: 0;
  }

  div.etcd-tree {
    padding-top: 0;
  }

}
</style>
