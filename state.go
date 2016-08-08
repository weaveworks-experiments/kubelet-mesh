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

func shouldUseTheirRootCA(ours, theirs ClusterInfo) bool {
	if ours.RootCA == nil {
		// Our certificate doesn't exist yet, so take whatever they give us
		return true
	}
	if theirs.RootCA.NotBefore.After(ours.RootCA.NotBefore) {
		// Their certificate is appears to have new validity period, we
		// should take that and use it
		return true
	}
	if theirs.RootCA.NotBefore.Equal(ours.RootCA.NotBefore) {
		// Both certificate has the same starting date of the validity
		// period, we should always pick the same one
		return bytes.Compare(theirs.RootCA.Signature, ours.RootCA.Signature) > 0
	}
	// Stick to what we have
	return false
}

func mergeClusterInfo(ours, theirs ClusterInfo) (result, delta ClusterInfo) {

	if theirs.RootCA != nil {
		if theirs.RootCA.Signature != nil {
			logger.Println("theirs.RootCA.Signature:", theirs.RootCA.Signature[:8])
		} else {
			logger.Println("theirs.RootCA.Signature: nil")
		}
	} else {
		logger.Println("theirs.RootCA: nil")
	}
	if ours.RootCA != nil {
		if ours.RootCA.Signature != nil {
			logger.Println("ours.RootCA.Signature:", ours.RootCA.Signature[:8])
		} else {
			logger.Println("ours.RootCA.Signature: nil")
		}
	} else {
		logger.Println("ours.RootCA: nil")
	}

	result = ours

	if theirs.RootCA != nil {
		if shouldUseTheirRootCA(ours, theirs) {
			result.RootCA = theirs.RootCA
			delta.RootCA = theirs.RootCA
		}
	}

	existing := map[string]struct{}{}
	incoming := map[string]struct{}{}
	for _, url := range ours.ApiserverURLs {
		existing[url] = struct{}{}
	}
	for _, url := range theirs.ApiserverURLs {
		incoming[url] = struct{}{}
	}
	for url := range incoming {
		if _, ok := existing[url]; !ok {
			// Don't have, do want; merge in.
			result.ApiserverURLs = append(result.ApiserverURLs, url)
			delta.ApiserverURLs = append(delta.ApiserverURLs, url)
		}
	}

	return result, delta
}

func (st *state) mergeReceived(set ClusterInfo) (received mesh.GossipData) {
	st.mtx.Lock()
	defer st.mtx.Unlock()
	cl, _ := mergeClusterInfo(set, st.set)
	st.set = cl
	return &state{
		set: set,
	}
}

func (st *state) mergeDelta(set ClusterInfo) (delta mesh.GossipData) {
	st.mtx.Lock()
	defer st.mtx.Unlock()

	cl, d := mergeClusterInfo(set, st.set)
	st.set = cl

	if len(set.ApiserverURLs) <= 0 && set.RootCA == nil {
		return nil
	}

	return &state{
		set: d,
	}
}

func (st *state) mergeComplete(set ClusterInfo) (complete mesh.GossipData) {
	st.mtx.Lock()
	defer st.mtx.Unlock()

	cl, _ := mergeClusterInfo(set, st.set)
	st.set = cl
	return &state{
		set: st.set,
	}
}
