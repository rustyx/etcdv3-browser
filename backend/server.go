package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/coreos/etcd/etcdserver/api/v3rpc/rpctypes"
	"github.com/rustyx/etcdv3-browser/nodetree"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/gorilla/websocket"
)

type apiServer struct {
	sync.Mutex // a lock for nodetree and revision updates
	etcd       *clientv3.Client
	root       *nodetree.Node
	broker     *Broker
	rev        int64
	editable   bool
}

type okResponse struct {
	Rev int64 `json:"rev"`
}

type updateMsg struct {
	Key     *string     `json:"key"`
	Value   interface{} `json:"value,omitempty"`   // undefined in case of omitted value
	Deleted interface{} `json:"deleted,omitempty"` // 1 if deleted, undefined otherwise
	Rev     int64       `json:"rev"`
}

func newServer(etcd *clientv3.Client, editable bool) *apiServer {
	server := apiServer{etcd: etcd, root: nodetree.NewNode(""), editable: editable, broker: NewBroker()}
	go server.initAndWatch()
	go server.broker.Start()
	return &server
}

func (s *apiServer) handleRequest(w http.ResponseWriter, r *http.Request, baseURI string) {
	prefix := r.URL.Path[len(baseURI):]
	switch r.Method {
	case "GET":
		if strings.HasSuffix(r.URL.Path, "/") {
			s.listSubtree(w, r, prefix)
		} else {
			s.getOne(w, r, prefix)
		}
	case "POST":
		s.updateOne(w, r, prefix)
	case "DELETE":
		s.deleteOne(w, r, prefix)
	default:
		w.WriteHeader(http.StatusBadRequest)
	}
}

type subtreeResponse struct {
	Rev  int64    `json:"rev"`
	Keys []string `json:"keys"`
}

func (s *apiServer) listSubtree(w http.ResponseWriter, r *http.Request, prefix string) {
	keys := s.getSubtreeKeys(prefix)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(*keys)
}

func (s *apiServer) getSubtreeKeys(prefix string) *subtreeResponse {
	s.Lock()
	defer s.Unlock()
	res := subtreeResponse{Rev: s.rev}
	subtree := s.root.GetNode(prefix)
	if subtree != nil && subtree.Count() > 0 {
		res.Keys = make([]string, 0, subtree.Count())
		for k, v := range subtree.Children() {
			if v.HasValue {
				res.Keys = append(res.Keys, k)
			}
			if v.Count() > 0 {
				res.Keys = append(res.Keys, k+"/")
			}
		}
		sort.Strings(res.Keys)
	}
	return &res
}

func (s *apiServer) getOne(w http.ResponseWriter, r *http.Request, prefix string) {
	resp, err := s.etcd.Get(r.Context(), prefix)
	if err != nil {
		log.Printf("Get: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain") // for ease of debugging, application/octet-stream otherwise
	if resp.Count == 0 {
		w.WriteHeader(http.StatusNotFound)
	}
	for _, ev := range resp.Kvs {
		if _, err = w.Write(ev.Value); err != nil {
			break
		}
	}
}

func (s *apiServer) updateOne(w http.ResponseWriter, r *http.Request, key string) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Print("ReadAll: ", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if !s.editable {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	res, err := s.etcd.Put(r.Context(), key, string(body))
	if err != nil {
		log.Printf("Put: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(&okResponse{Rev: res.Header.Revision})
}

func (s *apiServer) deleteOne(w http.ResponseWriter, r *http.Request, key string) {
	if !s.editable {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	res, err := s.etcd.Delete(r.Context(), key)
	if err != nil {
		log.Printf("Delete: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(&okResponse{Rev: res.Header.Revision})
}

func (s *apiServer) initAndWatch() {
	s.loadExisting()
	rev := s.rev
	go s.loadUpdates(s.broker.Subscribe())
	for delay := 20; delay < 10000; delay *= 2 {
		opts := []clientv3.OpOption{
			clientv3.WithPrefix(),
			clientv3.WithRev(rev),
		}
		log.Print("Watching starting from rev ", rev)
		for resp := range s.etcd.Watch(context.Background(), "", opts...) {
			if err := resp.Err(); err != nil {
				if err == rpctypes.ErrCompacted {
					rev = resp.CompactRevision
				}
				log.Print("watch failed: ", err)
				break
			}
			for _, ev := range resp.Events {
				key := string(ev.Kv.Key)
				if rev < ev.Kv.ModRevision {
					rev = ev.Kv.ModRevision
				}
				switch ev.Type {
				case mvccpb.PUT:
					value := string(ev.Kv.Value)
					s.broker.Publish(updateMsg{&key, &value, nil, ev.Kv.ModRevision})
				case mvccpb.DELETE:
					s.broker.Publish(updateMsg{&key, nil, 1, ev.Kv.ModRevision})
				}
			}
		}
		time.Sleep(time.Duration(delay) * time.Millisecond)
	}
	log.Fatal("Giving up.")
}

func (s *apiServer) loadExisting() {
	opts := []clientv3.OpOption{clientv3.WithPrefix()}
	resp, err := s.etcd.Get(context.Background(), "", opts...)
	if err != nil {
		log.Fatal("loadExisting: ", err)
		return
	}
	s.Lock()
	defer s.Unlock()
	for _, ev := range resp.Kvs {
		if ev.ModRevision > s.rev {
			s.rev = ev.ModRevision
		}
		s.root.AddNode(string(ev.Key))
	}
}

func (s *apiServer) loadUpdates(input chan interface{}) {
	for next := range input {
		msg := next.(updateMsg)
		s.Lock()
		if msg.Value != nil {
			s.root.AddNode(*msg.Key)
		} else {
			s.root.DeleteNode(*msg.Key)
		}
		if msg.Rev > s.rev {
			s.rev = msg.Rev
		}
		s.Unlock()
	}
	log.Print("loadUpdates exited")
}

const (
	pingPeriod   = 240 * time.Second
	writeTimeout = 10 * time.Second
	readTimeout  = writeTimeout + pingPeriod
)

func readPump(conn *websocket.Conn, keychan chan string) {
	defer func() { conn.Close(); close(keychan) }()
	conn.SetReadLimit(1024)
	if err := conn.SetReadDeadline(time.Now().Add(readTimeout)); err != nil {
		log.Print("SetReadDeadline: ", err)
	}
	conn.SetPongHandler(func(string) error { return conn.SetReadDeadline(time.Now().Add(readTimeout)) })
	for {
		_, msgb, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Print("ReadMessage: ", err)
			}
			break
		}
		var msg updateMsg
		err = json.Unmarshal(msgb, &msg)
		if err != nil || msg.Key == nil {
			log.Print("ReadMessage Unmarshal: ", err)
			continue
		}
		keychan <- *msg.Key
	}
}

func (s *apiServer) handleWebsocket(w http.ResponseWriter, r *http.Request) {
	var upgrader = websocket.Upgrader{
		ReadBufferSize:    1024,
		WriteBufferSize:   4096,
		EnableCompression: true,
		CheckOrigin: func(r *http.Request) bool {
			return true // TODO implement cors checking
		},
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("WS Upgrade: ", err)
		return
	}
	defer conn.Close()
	keychan := make(chan string, 64)
	rev := r.URL.Query()["rev"]
	if len(rev) > 0 && rev[0] != "0" {
		i, _ := strconv.ParseInt(rev[0], 10, 64)
		if i != s.rev {
			//TODO implement rev
			log.Printf("todo: handleWebsocket requesting rev %v - ignored, will continue from %d", rev[0], s.rev)
		}
	}
	go readPump(conn, keychan)
	input := s.broker.Subscribe()
	defer s.broker.Unsubscribe(input)
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	key := ""
loop:
	for {
		var err error
		select {
		case newkey, ok := <-keychan:
			if !ok {
				break loop
			}
			key = newkey
		case next, ok := <-input:
			if !ok {
				break loop
			}
			msg := next.(updateMsg)
			if msg.Value == nil {
				msg.Deleted = 1
			}
			if key != *msg.Key {
				msg.Value = nil
			}
			msgb, err2 := json.Marshal(msg)
			if err2 != nil {
				log.Print("json.Marshal: ", err2)
				break loop
			}
			if err = conn.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
				log.Print("SetWriteDeadline: ", err)
				break loop
			}
			err = conn.WriteMessage(websocket.TextMessage, msgb)
		case <-ticker.C:
			if err = conn.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
				log.Print("SetWriteDeadline: ", err)
				break loop
			}
			err = conn.WriteMessage(websocket.PingMessage, nil)
		}
		if err != nil {
			if err != websocket.ErrCloseSent {
				log.Print("WriteMessage: ", err)
			}
			break
		}
	}
}
