package torrent

import (
	"sync"

	pp "github.com/anacrolix/torrent/peer_protocol"
)

// Piece priority describes the importance of obtaining a particular piece.

type piecePriority byte

const (
	PiecePriorityNone      piecePriority = iota // Not wanted.
	PiecePriorityNormal                         // Wanted.
	PiecePriorityReadahead                      // May be required soon.
	PiecePriorityNext                           // Succeeds a piece where a read occurred.
	PiecePriorityNow                            // A read occurred in this piece.
)

type piece struct {
	// The completed piece SHA1 hash, from the metainfo "pieces" field.
	Hash pieceSum
	// Chunks dirtied since the last piece hash. The offset and length can be
	// determined by the request chunkSize in use.
	DirtyChunks      []bool
	Hashing          bool
	QueuedForHash    bool
	EverHashed       bool
	PublicPieceState PieceState

	pendingWritesMutex sync.Mutex
	pendingWrites      int
	noPendingWrites    sync.Cond
}

func (p *piece) pendingChunkIndex(index int) bool {
	if p.Hashing || p.QueuedForHash {
		return false
	}
	if index >= len(p.DirtyChunks) {
		return true
	}
	return !p.DirtyChunks[index]
}

func (p *piece) pendingChunk(cs chunkSpec, chunkSize pp.Integer) bool {
	return p.pendingChunkIndex(chunkIndex(cs, chunkSize))
}

func (p *piece) numDirtyChunks() (num int) {
	for _, dirty := range p.DirtyChunks {
		if dirty {
			num++
		}
	}
	return
}

func (p *piece) unpendChunkIndex(i int) {
	for len(p.DirtyChunks) <= i {
		p.DirtyChunks = append(p.DirtyChunks, false)
	}
	p.DirtyChunks[i] = true
}

func chunkIndexSpec(index int, pieceLength, chunkSize pp.Integer) chunkSpec {
	ret := chunkSpec{pp.Integer(index) * chunkSize, chunkSize}
	if ret.Begin+ret.Length > pieceLength {
		ret.Length = pieceLength - ret.Begin
	}
	return ret
}

func (p *piece) numPendingChunks(t *torrent, piece int) int {
	return t.numPendingChunks(piece)
}

func (p *piece) waitForPendingWrites() {
	p.pendingWritesMutex.Lock()
	for p.pendingWrites != 0 {
		p.noPendingWrites.Wait()
	}
	p.pendingWritesMutex.Unlock()
}
