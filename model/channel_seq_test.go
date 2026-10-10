package model

import (
	"fmt"
	"testing"

	"github.com/axzcnzxis/myapi/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// resetChannelTables clears the channels table so each sequence test starts from
// a known, empty state. The channels table has no soft-delete column, so a plain
// delete removes the rows for good.
func resetChannelTables(t *testing.T) {
	t.Helper()
	require.NoError(t, DB.AutoMigrate(&Channel{}))
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&Channel{}).Error)
	// Other tests insert channels with large explicit ids, which leaves SQLite's
	// AUTOINCREMENT counter far ahead of the rows deleted above. Clearing it keeps
	// the "a new channel appends at the end" expectations exact.
	DB.Exec("DELETE FROM sqlite_sequence WHERE name = 'channels'")
	t.Cleanup(func() {
		require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&Channel{}).Error)
	})
}

// createChannelWithId inserts a channel with an explicit id.
func createChannelWithId(t *testing.T, id int) {
	t.Helper()
	require.NoError(t, DB.Create(&Channel{
		Id:          id,
		Name:        fmt.Sprintf("channel-%d", id),
		Key:         fmt.Sprintf("sk-%d", id),
		Status:      common.ChannelStatusEnabled,
		Models:      "gpt-4o",
		Group:       "default",
		CreatedTime: common.GetTimestamp(),
	}).Error)
}

// channelSeqById maps every listed channel to the sequence number the console
// shows in its first column.
func channelSeqById(channels []*Channel) map[int]int {
	seqs := make(map[int]int, len(channels))
	for _, channel := range channels {
		seqs[channel.Id] = channel.Seq
	}
	return seqs
}

// attachSeqToListing mirrors what the console list handler does: load the
// channels and fill in the sequence column.
func attachSeqToListing(t *testing.T) []*Channel {
	t.Helper()
	channels, err := GetAllChannels(0, 10, true, false)
	require.NoError(t, err)
	require.NoError(t, AttachChannelSeq(DB, channels))
	return channels
}

// ---------------------------------------------------------------------------
// AttachChannelSeq — the value behind the console's first column
// ---------------------------------------------------------------------------

func TestAttachChannelSeq_NumbersRowsByAscendingId(t *testing.T) {
	resetChannelTables(t)
	for _, id := range []int{1, 2, 3} {
		createChannelWithId(t, id)
	}

	assert.Equal(t, map[int]int{1: 1, 2: 2, 3: 3}, channelSeqById(attachSeqToListing(t)))
}

func TestAttachChannelSeq_DefaultListingIsOldestFirst(t *testing.T) {
	resetChannelTables(t)
	for _, id := range []int{1, 2, 3} {
		createChannelWithId(t, id)
	}

	// Without any explicit sort the console must list channels oldest-first, so
	// the sequence column reads 1, 2, 3 from top to bottom.
	channels := attachSeqToListing(t)
	require.Len(t, channels, 3)
	for i, channel := range channels {
		assert.Equal(t, i+1, channel.Id, "rows must come back oldest first")
		assert.Equal(t, i+1, channel.Seq)
	}
}

func TestAttachChannelSeq_DeletingTheSecondChannelRenumbersTheRest(t *testing.T) {
	resetChannelTables(t)
	for _, id := range []int{1, 2, 3} {
		createChannelWithId(t, id)
	}

	// The channel is hard-deleted, so it drops out of the listing and the former
	// #3 must move up to occupy #2.
	require.NoError(t, DB.Delete(&Channel{}, 2).Error)

	assert.Equal(t, map[int]int{1: 1, 3: 2}, channelSeqById(attachSeqToListing(t)))
}

func TestAttachChannelSeq_NewChannelAppendsAtTheEnd(t *testing.T) {
	resetChannelTables(t)
	for _, id := range []int{1, 2, 3} {
		createChannelWithId(t, id)
	}
	require.NoError(t, DB.Delete(&Channel{}, 2).Error)

	// A new channel takes the next database sequence value rather than the freed
	// number, so the sequence only ever grows at the end.
	created := &Channel{
		Name:        "appended",
		Key:         "sk-appended",
		Status:      common.ChannelStatusEnabled,
		Models:      "gpt-4o",
		Group:       "default",
		CreatedTime: common.GetTimestamp(),
	}
	require.NoError(t, DB.Create(created).Error)
	assert.Equal(t, 4, created.Id, "the freed #2 must not be reused")

	assert.Equal(t, map[int]int{1: 1, 3: 2, 4: 3}, channelSeqById(attachSeqToListing(t)))
}

func TestAttachChannelSeq_LeavesEmptyWithoutChannels(t *testing.T) {
	require.NoError(t, AttachChannelSeq(DB, nil))
}

func TestAttachChannelSeq_DoesNotCountChannelsOutsideThePage(t *testing.T) {
	resetChannelTables(t)
	for _, id := range []int{1, 2, 3, 4, 5} {
		createChannelWithId(t, id)
	}

	// A page holding only the last two rows still numbers them by their global
	// position (4, 5), not by their position within the page.
	page, err := GetAllChannels(3, 2, false, false)
	require.NoError(t, err)
	require.NoError(t, AttachChannelSeq(DB, page))
	require.Len(t, page, 2)
	assert.Equal(t, map[int]int{4: 4, 5: 5}, channelSeqById(page))
}

func TestSearchChannels_ReturnsSeq(t *testing.T) {
	resetChannelTables(t)
	createChannelWithId(t, 1)
	createChannelWithId(t, 2)

	channels, err := SearchChannels("channel-2", "", "", false, NewChannelSortOptions("", "", false))
	require.NoError(t, err)
	require.Len(t, channels, 1)
	require.NoError(t, AttachChannelSeq(DB, channels))
	assert.Equal(t, 2, channels[0].Seq, "the sequence is the global position, not the position within the result page")
}
