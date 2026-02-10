package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestIsQuit(t *testing.T) {
	assert.True(t, isQuit(tea.KeyMsg{Type: tea.KeyCtrlC}))
	assert.True(t, isQuit(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}))
	assert.False(t, isQuit(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}))
}

func TestIsEnter(t *testing.T) {
	assert.True(t, isEnter(tea.KeyMsg{Type: tea.KeyEnter}))
	assert.False(t, isEnter(tea.KeyMsg{Type: tea.KeySpace}))
}

func TestIsSpace(t *testing.T) {
	assert.True(t, isSpace(tea.KeyMsg{Type: tea.KeySpace}))
	assert.False(t, isSpace(tea.KeyMsg{Type: tea.KeyEnter}))
}

func TestIsBack(t *testing.T) {
	assert.True(t, isBack(tea.KeyMsg{Type: tea.KeyEsc}))
	assert.False(t, isBack(tea.KeyMsg{Type: tea.KeyEnter}))
}

func TestIsDown(t *testing.T) {
	assert.True(t, isDown(tea.KeyMsg{Type: tea.KeyDown}))
	assert.False(t, isDown(tea.KeyMsg{Type: tea.KeyUp}))
	assert.False(t, isDown(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}))
}

func TestIsUp(t *testing.T) {
	assert.True(t, isUp(tea.KeyMsg{Type: tea.KeyUp}))
	assert.False(t, isUp(tea.KeyMsg{Type: tea.KeyDown}))
	assert.False(t, isUp(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}))
}

func TestIsTab(t *testing.T) {
	assert.True(t, isTab(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}}, 1))
	assert.True(t, isTab(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}}, 2))
	assert.True(t, isTab(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'5'}}, 5))
	assert.True(t, isTab(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'6'}}, 6))
	assert.True(t, isTab(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'7'}}, 7))
	assert.True(t, isTab(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'8'}}, 8))
	assert.False(t, isTab(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}}, 2))
}

func TestIsKey(t *testing.T) {
	assert.True(t, isKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}, "s"))
	assert.True(t, isKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}, "a"))
	assert.True(t, isKey(tea.KeyMsg{Type: tea.KeyBackspace}, "backspace"))
	assert.True(t, isKey(tea.KeyMsg{Type: tea.KeyLeft}, "left"))
	assert.True(t, isKey(tea.KeyMsg{Type: tea.KeyRight}, "right"))
	assert.False(t, isKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}, "a"))
	assert.False(t, isKey(tea.KeyMsg{Type: tea.KeyLeft}, "right"))
}
