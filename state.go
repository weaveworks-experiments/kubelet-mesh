package main

import (
	"bytes"
	"log"
	"sync"
	"time"

	"encoding/gob"

	"github.com/weaveworks/mesh"
)

type RootCAPublicKey struct {
	Bytes     []byte
	NotBefore time.Time
}

type ClusterInfo struct {
	RootCA *RootCAPublicKey
	// TODO ApiserverURLs []url.URL
	ApiserverURLs []string
}

type state struct {
	mtx  sync.RWMutex
	self mesh.PeerName
	set  map[mesh.PeerName]ClusterInfo
}

var logger *log.Logger

// state implements GossipData.
var _ mesh.GossipData = &state{}

// Construct an empty state object, ready to receive updates.
// This is suitable to use at program start.
// Other peers will populate us with data.
func newState(self mesh.PeerName, certInfo *RootCAPublicKey, log_ptr *log.Logger) *state {
	logger = log_ptr
	st := &state{
		set:  map[mesh.PeerName]ClusterInfo{},
		self: self,
	}

	st.set[self] = ClusterInfo{RootCA: certInfo}

	logger.Printf("I have root CA which is not valid before %v", st.set[self].RootCA.NotBefore)

	return st
}

func (st *state) copy() *state {
	st.mtx.RLock()
	defer st.mtx.RUnlock()
	return &state{
		set: st.set,
	}
}

// Encode serializes our complete state to a slice of byte-slices.
// In this simple example, we use a single JSON-encoded buffer.
func (st *state) Encode() [][]byte {
	st.mtx.RLock()
	defer st.mtx.RUnlock()
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(st.set); err != nil {
		panic(err)
	}
	return [][]byte{buf.Bytes()}
}

// Merge merges the other GossipData into this one,
// and returns our resulting, complete state.
func (st *state) Merge(other mesh.GossipData) (complete mesh.GossipData) {
	return st.mergeComplete(other.(*state).copy().set)
}

func mergedClusterInfo(peerInfo, ourInfo ClusterInfo) ClusterInfo {
	logger.Println("mergedClusterInfo(", peerInfo, ",", ourInfo, ")")
	cl := ClusterInfo{}
	// keep the root CA we're given if it exists and our current state says
	// it doesn't or if it's newer than the one we know about in our
	// current state
	if ourInfo.RootCA == nil || peerInfo.RootCA.NotBefore.UnixNano() > ourInfo.RootCA.NotBefore.UnixNano() {
		cl.RootCA = peerInfo.RootCA
	}
	// now take the union of the URLs we're given and the URLs we know
	// about
	urls := map[string]bool{}
	for _, url := range ourInfo.ApiserverURLs {
		urls[url] = true
	}
	for _, url := range peerInfo.ApiserverURLs {
		urls[url] = true
	}
	newURLs := []string{}
	for url, _ := range urls {
		newURLs = append(newURLs, url)
	}
	cl.ApiserverURLs = newURLs
	return cl
}

func (st *state) mergeReceived(set map[mesh.PeerName]ClusterInfo) (received mesh.GossipData) {
	st.mtx.Lock()
	defer st.mtx.Unlock()
	for peer, v := range set {
		cl := mergedClusterInfo(v, st.set[peer])
		st.set[peer] = cl
	}
	return &state{
		set: set,
	}
}

func (st *state) mergeDelta(set map[mesh.PeerName]ClusterInfo) (delta mesh.GossipData) {
	st.mtx.Lock()
	defer st.mtx.Unlock()

	for peer, v := range set {
		cl := mergedClusterInfo(v, st.set[peer])
		st.set[peer] = cl
	}
	if len(set) <= 0 {
		// TODO maybe we needed to mutate 'set' here, but we didn't.
		return nil
	}
	return &state{
		set: set,
	}
}

func (st *state) mergeComplete(set map[mesh.PeerName]ClusterInfo) (complete mesh.GossipData) {
	st.mtx.Lock()
	defer st.mtx.Unlock()

	for peer, v := range set {
		cl := mergedClusterInfo(v, st.set[peer])
		st.set[peer] = cl
	}
	return &state{
		set: st.set,
	}
}
