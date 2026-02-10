package components

// List is a simple scrollable list with cursor.
type List struct {
	Items    []string
	Cursor   int
	Offset   int
	PageSize int
}

// NewList creates a list with the given page size.
func NewList(pageSize int) *List {
	return &List{PageSize: pageSize}
}

// SetItems replaces items and resets cursor.
func (l *List) SetItems(items []string) {
	l.Items = items
	l.Cursor = 0
	l.Offset = 0
}

// Down moves the cursor down.
func (l *List) Down() {
	if l.Cursor < len(l.Items)-1 {
		l.Cursor++
		if l.Cursor >= l.Offset+l.PageSize {
			l.Offset++
		}
	}
}

// Up moves the cursor up.
func (l *List) Up() {
	if l.Cursor > 0 {
		l.Cursor--
		if l.Cursor < l.Offset {
			l.Offset--
		}
	}
}

// Visible returns the currently visible items.
func (l *List) Visible() []string {
	if len(l.Items) == 0 {
		return nil
	}
	end := l.Offset + l.PageSize
	if end > len(l.Items) {
		end = len(l.Items)
	}
	return l.Items[l.Offset:end]
}

// Selected returns the index of the selected item.
func (l *List) Selected() int {
	return l.Cursor
}

// IsSelected returns true if the given absolute index is the cursor.
func (l *List) IsSelected(absIdx int) bool {
	return absIdx == l.Cursor
}

// RelToAbs converts a relative (visible) index to absolute.
func (l *List) RelToAbs(relIdx int) int {
	return l.Offset + relIdx
}
