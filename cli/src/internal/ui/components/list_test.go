package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestListNewList(t *testing.T) {
	list := NewList(10)
	assert.Equal(t, 10, list.PageSize)
	assert.Equal(t, 0, list.Cursor)
	assert.Equal(t, 0, list.Offset)
	assert.Nil(t, list.Items)
}

func TestListSetItems(t *testing.T) {
	list := NewList(5)
	items := []string{"a", "b", "c"}

	list.SetItems(items)

	assert.Equal(t, items, list.Items)
	assert.Equal(t, 0, list.Cursor)
	assert.Equal(t, 0, list.Offset)
}

func TestListDownMovement(t *testing.T) {
	list := NewList(3)
	list.SetItems([]string{"a", "b", "c", "d", "e"})

	// Start at 0
	assert.Equal(t, 0, list.Cursor)
	assert.Equal(t, 0, list.Offset)

	// Move down within page
	list.Down()
	assert.Equal(t, 1, list.Cursor)
	assert.Equal(t, 0, list.Offset)

	list.Down()
	assert.Equal(t, 2, list.Cursor)
	assert.Equal(t, 0, list.Offset)

	// Move down - should scroll
	list.Down()
	assert.Equal(t, 3, list.Cursor)
	assert.Equal(t, 1, list.Offset)

	// Continue to end
	list.Down()
	assert.Equal(t, 4, list.Cursor)
	assert.Equal(t, 2, list.Offset)

	// Try to go past end - should stay
	list.Down()
	assert.Equal(t, 4, list.Cursor)
	assert.Equal(t, 2, list.Offset)
}

func TestListUpMovement(t *testing.T) {
	list := NewList(3)
	list.SetItems([]string{"a", "b", "c", "d", "e"})

	// Move to end first
	list.Cursor = 4
	list.Offset = 2

	// Move up within page (cursor 4->3, both visible in offset 2)
	list.Up()
	assert.Equal(t, 3, list.Cursor)
	assert.Equal(t, 2, list.Offset)

	// Move up - cursor 3->2, still visible in offset 2 (page shows indices 2,3,4)
	list.Up()
	assert.Equal(t, 2, list.Cursor)
	assert.Equal(t, 2, list.Offset) // Stays at 2, cursor = offset so just at edge

	// Move up - cursor 2->1, now cursor < offset, so scroll
	list.Up()
	assert.Equal(t, 1, list.Cursor)
	assert.Equal(t, 1, list.Offset)

	// Move up - cursor 1->0, cursor < offset so scroll
	list.Up()
	assert.Equal(t, 0, list.Cursor)
	assert.Equal(t, 0, list.Offset)

	// Try to go before start - should stay
	list.Up()
	assert.Equal(t, 0, list.Cursor)
	assert.Equal(t, 0, list.Offset)
}

func TestListVisible(t *testing.T) {
	list := NewList(3)
	list.SetItems([]string{"a", "b", "c", "d", "e"})

	// Initial page
	visible := list.Visible()
	assert.Equal(t, []string{"a", "b", "c"}, visible)

	// Scroll down
	list.Offset = 1
	visible = list.Visible()
	assert.Equal(t, []string{"b", "c", "d"}, visible)

	// Last page (partial)
	list.Offset = 3
	visible = list.Visible()
	assert.Equal(t, []string{"d", "e"}, visible)
}

func TestListVisibleEmpty(t *testing.T) {
	list := NewList(5)
	list.SetItems([]string{})

	visible := list.Visible()
	assert.Nil(t, visible)
}

func TestListVisibleSmallerThanPage(t *testing.T) {
	list := NewList(10)
	list.SetItems([]string{"a", "b", "c"})

	visible := list.Visible()
	assert.Equal(t, []string{"a", "b", "c"}, visible)
}

func TestListSelected(t *testing.T) {
	list := NewList(5)
	list.SetItems([]string{"a", "b", "c"})

	assert.Equal(t, 0, list.Selected())

	list.Down()
	assert.Equal(t, 1, list.Selected())
}

func TestListIsSelected(t *testing.T) {
	list := NewList(5)
	list.SetItems([]string{"a", "b", "c"})
	list.Cursor = 1

	assert.False(t, list.IsSelected(0))
	assert.True(t, list.IsSelected(1))
	assert.False(t, list.IsSelected(2))
}

func TestListRelToAbs(t *testing.T) {
	list := NewList(3)
	list.SetItems([]string{"a", "b", "c", "d", "e"})
	list.Offset = 2

	// Visible items are ["c", "d", "e"]
	// Relative index 0 -> absolute index 2
	assert.Equal(t, 2, list.RelToAbs(0))
	assert.Equal(t, 3, list.RelToAbs(1))
	assert.Equal(t, 4, list.RelToAbs(2))
}

func TestListScrollingLargeList(t *testing.T) {
	list := NewList(5)
	items := make([]string, 20)
	for i := range items {
		items[i] = string(rune('a' + i))
	}
	list.SetItems(items)

	// Navigate to middle
	for i := 0; i < 10; i++ {
		list.Down()
	}

	assert.Equal(t, 10, list.Cursor)
	assert.Equal(t, 6, list.Offset) // Should show items 6-10

	visible := list.Visible()
	assert.Len(t, visible, 5)
	assert.Equal(t, "g", visible[0]) // 6th item (0-indexed)
}
