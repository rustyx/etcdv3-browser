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

	"github.com/rustyx/etcdv3-browser/nodetree"

	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/gorilla/websocket"
	"go.etcd.io/etcd/clientv3"
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
	Key   *string `json:"key"`
	Value *string `json:"value"` // nil (null) in case of a delete
	Rev   int64   `json:"rev"`
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
	json.NewEncoder(w).Encode(*keys)
}

func (s *apiServer) getSubtreeKeys(prefix string) *subtreeResponse {
	s.Lock()
	defer s.Unlock()
	res := subtreeResponse{Rev: s.rev}
	subtree := s.root.GetNode(prefix)
	if subtree != nil && subtree.Count() > 0 {
		res.Keys = make([]string, 0, subtree.Count())
		for k, v := range subtree.Children() {
			if v.Count() == 0 {
				res.Keys = append(res.Keys, k)
			} else {
				res.Keys = append(res.Keys, k+"/")
			}
		}
		sort.Strings(res.Keys)
	}
	return &res
}

func (s *apiServer) getOne(w http.ResponseWriter, r *http.Request, prefix string) {
	resp, err := s.etcd.Get(r.Context(), prefix)
	if errCheck(w, err) {
		return
	}
	w.Header().Set("Content-Type", "text/plain") // for ease of debugging, application/octet-stream otherwise
	if resp.Count == 0 {
		w.WriteHeader(http.StatusNotFound)
	}
	for _, ev := range resp.Kvs {
		w.Write(ev.Value)
	}
}

func (s *apiServer) updateOne(w http.ResponseWriter, r *http.Request, key string) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if !s.editable {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	res, err := s.etcd.Put(r.Context(), key, string(body))
	if errCheck(w, err) {
		return
	}
	json.NewEncoder(w).Encode(&okResponse{Rev: res.Header.Revision})
}

func (s *apiServer) deleteOne(w http.ResponseWriter, r *http.Request, key string) {
	if !s.editable {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	res, err := s.etcd.Delete(r.Context(), key)
	if errCheck(w, err) {
		return
	}
	json.NewEncoder(w).Encode(&okResponse{Rev: res.Header.Revision})
}

func errCheck(w http.ResponseWriter, err error) bool {
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return true
	}
	return false
}

func (s *apiServer) initAndWatch() {
	s.loadExisting()
	log.Println("Watching starting from rev", s.rev)
	opts := []clientv3.OpOption{
		clientv3.WithPrefix(),
		clientv3.WithRev(s.rev),
	}
	go s.loadUpdates(s.broker.Subscribe())
	for resp := range s.etcd.Watch(context.Background(), "", opts...) {
		if err := resp.Err(); err != nil {
			log.Fatal(err)
			break
		}
		for _, ev := range resp.Events {
			key := string(ev.Kv.Key)
			switch ev.Type {
			case mvccpb.PUT:
				value := string(ev.Kv.Value)
				s.broker.Publish(updateMsg{&key, &value, ev.Kv.ModRevision})
			case mvccpb.DELETE:
				s.broker.Publish(updateMsg{&key, nil, ev.Kv.ModRevision})
			}
		}
	}
	log.Fatal("Watch ended")
}

func (s *apiServer) loadExisting() {
	opts := []clientv3.OpOption{clientv3.WithPrefix()}
	resp, err := s.etcd.Get(context.Background(), "", opts...)
	if err != nil {
		log.Fatal(err)
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
	log.Println("loadUpdates exited")
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
		log.Println(err)
		return
	}
	rev := r.URL.Query()["rev"]
	if len(rev) > 0 && rev[0] != "0" {
		i, _ := strconv.ParseInt(rev[0], 10, 64)
		if i != s.rev {
			//TODO implement rev
			log.Printf("todo: handleWebsocket requesting rev %v - ignored, will continue from %d", rev[0], s.rev)
		}
	}
	input := s.broker.Subscribe()
	defer s.broker.Unsubscribe(input)
	for next := range input {
		msg := next.(updateMsg)
		binmsg, err := json.Marshal(msg)
		if errCheck(w, err) {
			return
		}
		err = conn.WriteMessage(websocket.BinaryMessage, binmsg)
		if errCheck(w, err) {
			return
		}
	}
}
