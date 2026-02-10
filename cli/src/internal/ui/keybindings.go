package ui

import "github.com/charmbracelet/bubbletea"

// --- Key Constants ---

func isKey(msg tea.KeyMsg, keys ...string) bool {
	for _, k := range keys {
		if msg.String() == k {
			return true
		}
	}
	return false
}

func isQuit(msg tea.KeyMsg) bool {
	return isKey(msg, "q", "ctrl+c")
}

func isBack(msg tea.KeyMsg) bool {
	if msg.Type == tea.KeyEsc {
		return true
	}
	return isKey(msg, "esc", "escape", "ctrl+[")
}

func isUp(msg tea.KeyMsg) bool {
	return isKey(msg, "up")
}

func isDown(msg tea.KeyMsg) bool {
	return isKey(msg, "down")
}

func isEnter(msg tea.KeyMsg) bool {
	return isKey(msg, "enter", "return")
}

func isSpace(msg tea.KeyMsg) bool {
	return isKey(msg, " ")
}
