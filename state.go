package main

import (
	"bytes"
	"net/url"
	"sync"
	"time"

	_ "encoding/gob"

	"github.com/weaveworks/mesh"
)

type state struct {
	mtx           sync.RWMutex
	cert          map[mesh.PeerName]string
	self          mesh.PeerName
	rootCA        map[mesh.PeerName]RootCAPublicKey
	apiserverURLs map[mesh.PeerName][]url.URL
}

type RootCAPublicKey struct {
	Bytes     []byte
	NotBefore time.Time
}

// state implements GossipData.
var _ mesh.GossipData = &state{}

// Construct an empty state object, ready to receive updates.
// This is suitable to use at program start.
// Other peers will populate us with data.
func newState(self mesh.PeerName, certInfo *RootCAPublicKey) *state {
	st := &state{
		rootCA:        map[mesh.PeerName]RootCAPublicKey{},
		apiserverURLs: map[mesh.PeerName][]url.URL{},
		self:          self,
	}

	if certInfo != nil {
		st.rootCA[self] = *certInfo
	}

	return st
}

// Encode serializes our complete state to a slice of byte-slices.
// In this simple example, we use a single JSON-encoded buffer.
func (st *state) Encode() [][]byte {
	st.mtx.RLock()
	defer st.mtx.RUnlock()
	var buf bytes.Buffer
	return [][]byte{buf.Bytes()}
}

// Merge merges the other GossipData into this one,
// and returns our resulting, complete state.
func (st *state) Merge(other mesh.GossipData) (complete mesh.GossipData) {
	return nil
}

func (st *state) mergeReceived(set map[mesh.PeerName]int) (received mesh.GossipData) {
	st.mtx.Lock()
	defer st.mtx.Unlock()

	return nil
}

func (st *state) mergeDelta(set map[mesh.PeerName]int) (delta mesh.GossipData) {
	st.mtx.Lock()
	defer st.mtx.Unlock()

	return nil
}

func (st *state) mergeComplete(set map[mesh.PeerName]int) (complete mesh.GossipData) {
	st.mtx.Lock()
	defer st.mtx.Unlock()

	return nil
}
