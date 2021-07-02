package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rustyx/etcdv3-browser/nodetree"
	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/etcdserver/api/v3rpc/rpctypes"
	"go.etcd.io/etcd/mvcc/mvccpb"
)

type apiServer struct {
	sync.Mutex // a lock for nodetree and revision updates
	etcd       *clientv3.Client
	root       *nodetree.Node
	broker     *Broker
	leases     map[int64]bool
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
	Lease   int64       `json:"lease,omitempty"`
}

func newServer(etcd *clientv3.Client, editable bool) *apiServer {
	server := apiServer{etcd: etcd, root: nodetree.NewNode("", 0), editable: editable, broker: NewBroker()}
	go server.initAndWatch()
	go server.broker.Start()
	go server.removeExpiredLoop()
	return &server
}

func (s *apiServer) handleList(w http.ResponseWriter, r *http.Request) {
	key := r.FormValue("k")
	switch r.Method {
	case "GET":
		s.listSubtree(w, r, key)
	default:
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (s *apiServer) handleOne(w http.ResponseWriter, r *http.Request) {
	key := r.FormValue("k")
	switch r.Method {
	case "GET":
		s.getOne(w, r, key)
	case "POST":
		s.updateOne(w, r, key)
	case "DELETE":
		s.deleteOne(w, r, key)
	default:
		w.WriteHeader(http.StatusBadRequest)
	}
}

type Entry struct {
	Key  string `json:"k"`
	Type int    `json:"t"` // bit field: 1 = has value, 2 = has children
}

type subtreeResponse struct {
	Rev      int64   `json:"rev"`
	Editable bool    `json:"editable,omitempty"`
	Keys     []Entry `json:"keys"`
}

func (s *apiServer) listSubtree(w http.ResponseWriter, r *http.Request, key string) {
	keys := s.getSubtreeKeys(key)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(*keys)
}

func (s *apiServer) getSubtreeKeys(prefix string) *subtreeResponse {
	s.Lock()
	defer s.Unlock()
	res := subtreeResponse{Rev: s.rev}
	if prefix == "" {
		res.Editable = s.editable
	}
	subtree := s.root.GetNode(prefix)
	if subtree != nil && subtree.Count() > 0 {
		res.Keys = make([]Entry, 0, subtree.Count())
		for k, v := range subtree.Children() {
			e := Entry{Key: k, Type: 0}
			if v.HasValue {
				e.Type |= 1
			}
			if v.Count() > 0 {
				e.Type |= 2
			}
			res.Keys = append(res.Keys, e)
		}
		sort.Slice(res.Keys[:], func(i, j int) bool {
			return res.Keys[i].Key < res.Keys[j].Key
		})
	}
	return &res
}

func (s *apiServer) getOne(w http.ResponseWriter, r *http.Request, key string) {
	resp, err := s.etcd.Get(r.Context(), key)
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

func (s *apiServer) getLeaseID(key string) clientv3.LeaseID {
	s.Lock()
	defer s.Unlock()
	res := s.root.GetNode(key)
	if res == nil {
		return 0
	}
	return clientv3.LeaseID(res.LeaseID)
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
	leaseID := s.getLeaseID(key)
	res, err := s.etcd.Put(r.Context(), key, string(body), clientv3.WithLease(leaseID))
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
		log.Print("Watching starting from rev ", rev)
		for resp := range s.etcd.Watch(context.Background(), "", clientv3.WithPrefix(), clientv3.WithRev(rev)) {
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
					s.broker.Publish(updateMsg{&key, &value, nil, ev.Kv.ModRevision, ev.Kv.Lease})
				case mvccpb.DELETE:
					s.broker.Publish(updateMsg{&key, nil, 1, ev.Kv.ModRevision, ev.Kv.Lease})
				}
			}
		}
		time.Sleep(time.Duration(delay) * time.Millisecond)
	}
	log.Fatal("Giving up.")
}

func (s *apiServer) loadExisting() {
	resp, err := s.etcd.Get(context.Background(), "", clientv3.WithPrefix())
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
		s.root.AddNode(string(ev.Key), ev.Lease)
	}
}

func (s *apiServer) loadUpdates(input chan interface{}) {
	for next := range input {
		msg := next.(updateMsg)
		s.Lock()
		if msg.Value != nil {
			s.root.AddNode(*msg.Key, msg.Lease)
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

func (s *apiServer) removeExpiredLoop() {
	for {
		now := time.Now().UTC().Unix()
		delay := 300 - now%300
		time.Sleep(time.Duration(delay) * time.Second)
		s.removeExpired()
	}
}

func (s *apiServer) removeExpired() {
	s.Lock()
	defer s.Unlock()
	s.leases = make(map[int64]bool)
	s.removeExpiredRecursive(s.root)
	s.leases = nil
}

func (s *apiServer) removeExpiredRecursive(node *nodetree.Node) {
	for k, sub := range node.Children() {
		if s.isExpired(sub.LeaseID) {
			node.DeleteNode(k)
			continue
		}
		s.removeExpiredRecursive(sub)
	}
}

func (s *apiServer) isExpired(leaseID int64) bool {
	if leaseID <= 0 {
		return false
	}
	expired, found := s.leases[leaseID]
	if !found {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		ttl, err := s.etcd.TimeToLive(ctx, clientv3.LeaseID(leaseID))
		cancel()
		if err != nil {
			log.Printf("TimeToLive %d: %v", leaseID, err)
			expired = false
		} else {
			expired = ttl.TTL <= 0
		}
		s.leases[leaseID] = expired
	}
	return expired
}
