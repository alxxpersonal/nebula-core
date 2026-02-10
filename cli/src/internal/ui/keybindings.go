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

func isTab(msg tea.KeyMsg, n int) bool {
	switch n {
	case 1:
		return isKey(msg, "1")
	case 2:
		return isKey(msg, "2")
	case 3:
		return isKey(msg, "3")
	case 4:
		return isKey(msg, "4")
	case 5:
		return isKey(msg, "5")
	case 6:
		return isKey(msg, "6")
	case 7:
		return isKey(msg, "7")
	case 8:
		return isKey(msg, "8")
	}
	return false
}
