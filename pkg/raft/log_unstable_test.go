// This code has been modified from its original form by Cockroach Labs, Inc.
// All modifications are Copyright 2024 Cockroach Labs, Inc.
//
// Copyright 2015 The etcd Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package raft

import (
	"fmt"
	"testing"

	pb "github.com/cockroachdb/cockroach/pkg/raft/raftpb"
	"github.com/stretchr/testify/require"
)

func (u *unstable) checkInvariants(t testing.TB) {
	t.Helper()
	require.GreaterOrEqual(t, u.offsetInProgress, u.offset)
	require.LessOrEqual(t, u.offsetInProgress-u.offset, uint64(len(u.entries)))
	if u.snapshot != nil {
		require.Equal(t, u.snapshot.Metadata.Index+1, u.offset)
	} else {
		require.False(t, u.snapshotInProgress)
	}
	if len(u.entries) != 0 {
		require.Equal(t, u.entries[0].Index, u.offset)
	}
	if u.offsetInProgress > u.offset && u.snapshot != nil {
		require.True(t, u.snapshotInProgress)
	}
}

func TestUnstableMaybeFirstIndex(t *testing.T) {
	tests := []struct {
		entries []pb.Entry
		offset  uint64
		snap    *pb.Snapshot

		wok    bool
		windex uint64
	}{
		// no snapshot
		{
			index(5).terms(1), 5, nil,
			false, 0,
		},
		{
			[]pb.Entry{}, 0, nil,
			false, 0,
		},
		// has snapshot
		{
			index(5).terms(1), 5, &pb.Snapshot{Metadata: pb.SnapshotMetadata{Index: 4, Term: 1}},
			true, 5,
		},
		{
			[]pb.Entry{}, 5, &pb.Snapshot{Metadata: pb.SnapshotMetadata{Index: 4, Term: 1}},
			true, 5,
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			u := newUnstable(tt.offset, raftLogger)
			u.snapshot = tt.snap
			u.entries = tt.entries
			u.checkInvariants(t)

			index, ok := u.maybeFirstIndex()
			require.Equal(t, tt.wok, ok)
			require.Equal(t, tt.windex, index)
		})
	}
}

func TestMaybeLastIndex(t *testing.T) {
	tests := []struct {
		entries []pb.Entry
		offset  uint64
		snap    *pb.Snapshot

		wok    bool
		windex uint64
	}{
		// last in entries
		{
			index(5).terms(1), 5, nil,
			true, 5,
		},
		{
			index(5).terms(1), 5, &pb.Snapshot{Metadata: pb.SnapshotMetadata{Index: 4, Term: 1}},
			true, 5,
		},
		// last in snapshot
		{
			[]pb.Entry{}, 5, &pb.Snapshot{Metadata: pb.SnapshotMetadata{Index: 4, Term: 1}},
			true, 4,
		},
		// empty unstable
		{
			[]pb.Entry{}, 0, nil,
			false, 0,
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			u := newUnstable(tt.offset, raftLogger)
			u.snapshot = tt.snap
			u.entries = tt.entries
			u.checkInvariants(t)

			index, ok := u.maybeLastIndex()
			require.Equal(t, tt.wok, ok)
			require.Equal(t, tt.windex, index)
		})
	}
}

func TestUnstableMaybeTerm(t *testing.T) {
	tests := []struct {
		entries []pb.Entry
		offset  uint64
		snap    *pb.Snapshot
		index   uint64

		wok   bool
		wterm uint64
	}{
		// term from entries
		{
			index(5).terms(1), 5, nil,
			5,
			true, 1,
		},
		{
			index(5).terms(1), 5, nil,
			6,
			false, 0,
		},
		{
			index(5).terms(1), 5, nil,
			4,
			false, 0,
		},
		{
			index(5).terms(1), 5, &pb.Snapshot{Metadata: pb.SnapshotMetadata{Index: 4, Term: 1}},
			5,
			true, 1,
		},
		{
			index(5).terms(1), 5, &pb.Snapshot{Metadata: pb.SnapshotMetadata{Index: 4, Term: 1}},
			6,
			false, 0,
		},
		// term from snapshot
		{
			index(5).terms(1), 5, &pb.Snapshot{Metadata: pb.SnapshotMetadata{Index: 4, Term: 1}},
			4,
			true, 1,
		},
		{
			index(5).terms(1), 5, &pb.Snapshot{Metadata: pb.SnapshotMetadata{Index: 4, Term: 1}},
			3,
			false, 0,
		},
		{
			[]pb.Entry{}, 5, &pb.Snapshot{Metadata: pb.SnapshotMetadata{Index: 4, Term: 1}},
			5,
			false, 0,
		},
		{
			[]pb.Entry{}, 5, &pb.Snapshot{Metadata: pb.SnapshotMetadata{Index: 4, Term: 1}},
			4,
			true, 1,
		},
		{
			[]pb.Entry{}, 0, nil,
			5,
			false, 0,
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			u := newUnstable(tt.offset, raftLogger)
			u.snapshot = tt.snap
			u.entries = tt.entries
			u.checkInvariants(t)

			term, ok := u.maybeTerm(tt.index)
			require.Equal(t, tt.wok, ok)
			require.Equal(t, tt.wterm, term)
		})
	}
}

func TestUnstableRestore(t *testing.T) {
	u := unstable{
		entries:            index(5).terms(1),
		offset:             5,
		offsetInProgress:   6,
		snapshot:           &pb.Snapshot{Metadata: pb.SnapshotMetadata{Index: 4, Term: 1}},
		snapshotInProgress: true,
		logger:             raftLogger,
	}
	u.checkInvariants(t)

	s := snapshot{
		term: 2,
		snap: pb.Snapshot{Metadata: pb.SnapshotMetadata{Index: 6, Term: 2}},
	}
	u.restore(s)
	u.checkInvariants(t)

	require.Equal(t, s.lastIndex()+1, u.offset)
	require.Equal(t, s.lastIndex()+1, u.offsetInProgress)
	require.Zero(t, len(u.entries))
	require.Equal(t, &s.snap, u.snapshot)
	require.False(t, u.snapshotInProgress)
}

func TestUnstableNextEntries(t *testing.T) {
	tests := []struct {
		entries          []pb.Entry
		offset           uint64
		offsetInProgress uint64

		wentries []pb.Entry
	}{
		// nothing in progress
		{
			index(5).terms(1, 1), 5, 5,
			index(5).terms(1, 1),
		},
		// partially in progress
		{
			index(5).terms(1, 1), 5, 6,
			index(6).terms(1),
		},
		// everything in progress
		{
			index(5).terms(1, 1), 5, 7,
			nil, // nil, not empty slice
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			u := newUnstable(tt.offset, raftLogger)
			u.entries = tt.entries
			u.offsetInProgress = tt.offsetInProgress
			u.checkInvariants(t)
			require.Equal(t, tt.wentries, u.nextEntries())
		})
	}
}

func TestUnstableNextSnapshot(t *testing.T) {
	s := &pb.Snapshot{Metadata: pb.SnapshotMetadata{Index: 4, Term: 1}}
	tests := []struct {
		offset             uint64
		snapshot           *pb.Snapshot
		snapshotInProgress bool

		wsnapshot *pb.Snapshot
	}{
		// snapshot not unstable
		{
			0, nil, false,
			nil,
		},
		// snapshot not in progress
		{
			5, s, false,
			s,
		},
		// snapshot in progress
		{
			5, s, true,
			nil,
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			u := newUnstable(tt.offset, raftLogger)
			u.snapshot = tt.snapshot
			u.snapshotInProgress = tt.snapshotInProgress
			u.checkInvariants(t)
			require.Equal(t, tt.wsnapshot, u.nextSnapshot())
		})
	}
}

func TestUnstableAcceptInProgress(t *testing.T) {
	tests := []struct {
		entries            []pb.Entry
		snapshot           *pb.Snapshot
		offset             uint64
		offsetInProgress   uint64
		snapshotInProgress bool

		woffsetInProgress   uint64
		wsnapshotInProgress bool
	}{
		{
			[]pb.Entry{}, nil,
			5, 5, // no entries
			false, // snapshot not already in progress
			5, false,
		},
		{
			index(5).terms(1), nil,
			5, 5, // entries not in progress
			false, // snapshot not already in progress
			6, false,
		},
		{
			index(5).terms(1, 1), nil,
			5, 5, // entries not in progress
			false, // snapshot not already in progress
			7, false,
		},
		{
			index(5).terms(1, 1), nil,
			5, 6, // in-progress to the first entry
			false, // snapshot not already in progress
			7, false,
		},
		{
			index(5).terms(1, 1), nil,
			5, 7, // in-progress to the second entry
			false, // snapshot not already in progress
			7, false,
		},
		// with snapshot
		{
			[]pb.Entry{}, &pb.Snapshot{Metadata: pb.SnapshotMetadata{Index: 4, Term: 1}},
			5, 5, // no entries
			false, // snapshot not already in progress
			5, true,
		},
		{
			index(5).terms(1), &pb.Snapshot{Metadata: pb.SnapshotMetadata{Index: 4, Term: 1}},
			5, 5, // entries not in progress
			false, // snapshot not already in progress
			6, true,
		},
		{
			index(5).terms(1, 1), &pb.Snapshot{Metadata: pb.SnapshotMetadata{Index: 4, Term: 1}},
			5, 5, // entries not in progress
			false, // snapshot not already in progress
			7, true,
		},
		{
			[]pb.Entry{}, &pb.Snapshot{Metadata: pb.SnapshotMetadata{Index: 4, Term: 1}},
			5, 5, // entries not in progress
			true, // snapshot already in progress
			5, true,
		},
		{
			index(5).terms(1), &pb.Snapshot{Metadata: pb.SnapshotMetadata{Index: 4, Term: 1}},
			5, 5, // entries not in progress
			true, // snapshot already in progress
			6, true,
		},
		{
			index(5).terms(1, 1), &pb.Snapshot{Metadata: pb.SnapshotMetadata{Index: 4, Term: 1}},
			5, 5, // entries not in progress
			true, // snapshot already in progress
			7, true,
		},
		{
			index(5).terms(1, 1), &pb.Snapshot{Metadata: pb.SnapshotMetadata{Index: 4, Term: 1}},
			5, 6, // in-progress to the first entry
			true, // snapshot already in progress
			7, true,
		},
		{
			index(5).terms(1, 1), &pb.Snapshot{Metadata: pb.SnapshotMetadata{Index: 4, Term: 1}},
			5, 7, // in-progress to the second entry
			true, // snapshot already in progress
			7, true,
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			u := newUnstable(tt.offset, raftLogger)
			u.snapshot = tt.snapshot
			u.entries = tt.entries
			u.snapshotInProgress = tt.snapshotInProgress
			u.offsetInProgress = tt.offsetInProgress
			u.checkInvariants(t)

			u.acceptInProgress()
			u.checkInvariants(t)
			require.Equal(t, tt.woffsetInProgress, u.offsetInProgress)
			require.Equal(t, tt.wsnapshotInProgress, u.snapshotInProgress)
		})
	}
}

func TestUnstableStableTo(t *testing.T) {
	tests := []struct {
		entries          []pb.Entry
		offset           uint64
		offsetInProgress uint64
		snap             *pb.Snapshot
		index, term      uint64

		woffset           uint64
		woffsetInProgress uint64
		wlen              int
	}{
		{
			[]pb.Entry{}, 0, 0, nil,
			5, 1,
			0, 0, 0,
		},
		{
			index(5).terms(1), 5, 6, nil,
			5, 1, // stable to the first entry
			6, 6, 0,
		},
		{
			index(5).terms(1, 1), 5, 6, nil,
			5, 1, // stable to the first entry
			6, 6, 1,
		},
		{
			index(5).terms(1, 1), 5, 7, nil,
			5, 1, // stable to the first entry and in-progress ahead
			6, 7, 1,
		},
		{
			index(6).terms(2), 6, 7, nil,
			6, 1, // stable to the first entry and term mismatch
			6, 7, 1,
		},
		{
			index(5).terms(1), 5, 6, nil,
			4, 1, // stable to old entry
			5, 6, 1,
		},
		{
			index(5).terms(1), 5, 6, nil,
			4, 2, // stable to old entry
			5, 6, 1,
		},
		// with snapshot
		{
			index(5).terms(1), 5, 6, &pb.Snapshot{Metadata: pb.SnapshotMetadata{Index: 4, Term: 1}},
			5, 1, // stable to the first entry
			6, 6, 0,
		},
		{
			index(5).terms(1, 1), 5, 6, &pb.Snapshot{Metadata: pb.SnapshotMetadata{Index: 4, Term: 1}},
			5, 1, // stable to the first entry
			6, 6, 1,
		},
		{
			index(5).terms(1, 1), 5, 7, &pb.Snapshot{Metadata: pb.SnapshotMetadata{Index: 4, Term: 1}},
			5, 1, // stable to the first entry and in-progress ahead
			6, 7, 1,
		},
		{
			index(6).terms(2), 6, 7, &pb.Snapshot{Metadata: pb.SnapshotMetadata{Index: 5, Term: 1}},
			6, 1, // stable to the first entry and term mismatch
			6, 7, 1,
		},
		{
			index(5).terms(1), 5, 6, &pb.Snapshot{Metadata: pb.SnapshotMetadata{Index: 4, Term: 1}},
			4, 1, // stable to snapshot
			5, 6, 1,
		},
		{
			index(5).terms(2), 5, 6, &pb.Snapshot{Metadata: pb.SnapshotMetadata{Index: 4, Term: 2}},
			4, 1, // stable to old entry
			5, 6, 1,
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			u := newUnstable(tt.offset, raftLogger)
			u.snapshot = tt.snap
			u.entries = tt.entries
			u.offsetInProgress = tt.offsetInProgress
			u.snapshotInProgress = u.snapshot != nil && u.offsetInProgress > u.offset
			u.checkInvariants(t)

			if u.snapshotInProgress {
				u.stableSnapTo(u.snapshot.Metadata.Index)
			}
			u.checkInvariants(t)
			u.stableTo(entryID{term: tt.term, index: tt.index})
			u.checkInvariants(t)
			require.Equal(t, tt.woffset, u.offset)
			require.Equal(t, tt.woffsetInProgress, u.offsetInProgress)
			require.Equal(t, tt.wlen, len(u.entries))
		})
	}
}

func TestUnstableTruncateAndAppend(t *testing.T) {
	tests := []struct {
		entries          []pb.Entry
		offset           uint64
		offsetInProgress uint64
		snap             *pb.Snapshot
		toappend         []pb.Entry

		woffset           uint64
		woffsetInProgress uint64
		wentries          []pb.Entry
	}{
		// append to the end
		{
			index(5).terms(1), 5, 5, nil,
			index(6).terms(1, 1),
			5, 5, index(5).terms(1, 1, 1),
		},
		{
			index(5).terms(1), 5, 6, nil,
			index(6).terms(1, 1),
			5, 6, index(5).terms(1, 1, 1),
		},
		// replace the unstable entries
		{
			index(5).terms(1), 5, 5, nil,
			index(5).terms(2, 2),
			5, 5, index(5).terms(2, 2),
		},
		{
			index(5).terms(1), 5, 5, nil,
			index(4).terms(2, 2, 2),
			4, 4, index(4).terms(2, 2, 2),
		},
		{
			index(5).terms(1), 5, 6, nil,
			index(5).terms(2, 2),
			5, 5, index(5).terms(2, 2),
		},
		// truncate the existing entries and append
		{
			index(5).terms(1, 1, 1), 5, 5, nil,
			index(6).terms(2),
			5, 5, index(5).terms(1, 2),
		},
		{
			index(5).terms(1, 1, 1), 5, 5, nil,
			index(7).terms(2, 2),
			5, 5, index(5).terms(1, 1, 2, 2),
		},
		{
			index(5).terms(1, 1, 1), 5, 6, nil,
			index(6).terms(2),
			5, 6, index(5).terms(1, 2),
		},
		{
			index(5).terms(1, 1, 1), 5, 7, nil,
			index(6).terms(2),
			5, 6, index(5).terms(1, 2),
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			u := newUnstable(tt.offset, raftLogger)
			u.snapshot = tt.snap
			u.entries = tt.entries
			u.offsetInProgress = tt.offsetInProgress
			u.snapshotInProgress = u.snapshot != nil && u.offsetInProgress > u.offset
			u.checkInvariants(t)

			u.truncateAndAppend(tt.toappend)
			u.checkInvariants(t)
			require.Equal(t, tt.woffset, u.offset)
			require.Equal(t, tt.woffsetInProgress, u.offsetInProgress)
			require.Equal(t, tt.wentries, u.entries)
		})
	}
}
