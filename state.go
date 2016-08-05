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
	Signature []byte
}

type ClusterInfo struct {
	RootCA *RootCAPublicKey
	// TODO ApiserverURLs []url.URL
	ApiserverURLs []string
}

type state struct {
	mtx  sync.RWMutex
	self mesh.PeerName
	// TODO rename 'set' to 'info'
	set ClusterInfo
}

var logger *log.Logger

// state implements GossipData.
var _ mesh.GossipData = &state{}

// Construct an empty state object, ready to receive updates.
// This is suitable to use at program start.
// Other peers will populate us with data.
func newState(self mesh.PeerName, certInfo *RootCAPublicKey, apiservers []string, log_ptr *log.Logger) *state {
	logger = log_ptr
	st := &state{
		set:  ClusterInfo{},
		self: self,
	}

	st.set = ClusterInfo{RootCA: certInfo, ApiserverURLs: apiservers}

	logger.Printf("I have root CA which is not valid before %v", st.set.RootCA.NotBefore)

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

func mergedClusterInfo(peerInfo, ourInfo ClusterInfo) (result ClusterInfo, delta ClusterInfo) {
	if peerInfo.RootCA != nil {
		if peerInfo.RootCA.Signature != nil {
			logger.Println("peerInfo.RootCA:", peerInfo.RootCA.Signature[:8])
		} else {
			logger.Println("peerInfo.RootCA.Signature: nil")
		}
	} else {
		logger.Println("peerInfo.RootCA: nil")
	}
	if ourInfo.RootCA != nil {
		if ourInfo.RootCA.Signature != nil {
			logger.Println("ourInfo.RootCA:", ourInfo.RootCA.Signature[:8])
		} else {
			logger.Println("ourInfo.RootCA.Signature: nil")
		}
	} else {
		logger.Println("ourInfo.RootCA: nil")
	}

	result = ClusterInfo{}
	delta = ClusterInfo{}
	// keep the root CA we're given if it exists and our current state says
	// it doesn't or if it's newer than the one we know about in our
	// current state
	result.RootCA = ourInfo.RootCA
	if peerInfo.RootCA != nil || peerInfo.RootCA.NotBefore.UnixNano() > ourInfo.RootCA.NotBefore.UnixNano() {
		if peerInfo.RootCA != nil {
			result.RootCA = peerInfo.RootCA
			delta.RootCA = peerInfo.RootCA
		}
	}
	// now take the union of the URLs we're given and the URLs we know
	// about
	resultURLs := map[string]bool{}
	deltaURLs := map[string]bool{}
	for _, url := range ourInfo.ApiserverURLs {
		resultURLs[url] = true
	}
	for _, url := range peerInfo.ApiserverURLs {
		resultURLs[url] = true
		deltaURLs[url] = true
	}
	newResultURLs := []string{}
	for url, _ := range resultURLs {
		newResultURLs = append(newResultURLs, url)
	}
	newDeltaURLs := []string{}
	for url, _ := range deltaURLs {
		newDeltaURLs = append(newDeltaURLs, url)
	}
	result.ApiserverURLs = newResultURLs
	delta.ApiserverURLs = newDeltaURLs
	return
}

func (st *state) mergeReceived(set ClusterInfo) (received mesh.GossipData) {
	st.mtx.Lock()
	defer st.mtx.Unlock()
	cl, _ := mergedClusterInfo(set, st.set)
	st.set = cl
	return &state{
		set: set,
	}
}

func (st *state) mergeDelta(set ClusterInfo) (delta mesh.GossipData) {
	st.mtx.Lock()
	defer st.mtx.Unlock()

	cl, delta := mergedClusterInfo(set, st.set)
	st.set = cl

	if len(set) <= 0 {
		return nil
	}

	return &state{
		set: delta,
	}
}

func (st *state) mergeComplete(set ClusterInfo) (complete mesh.GossipData) {
	st.mtx.Lock()
	defer st.mtx.Unlock()

	cl, _ := mergedClusterInfo(v, st.set)
	st.set = cl
	return &state{
		set: st.set,
	}
}
