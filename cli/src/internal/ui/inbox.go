package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/gravitrone/nebula-core/cli/internal/api"
	"github.com/gravitrone/nebula-core/cli/internal/ui/components"
)

// --- Messages ---

type approvalsLoadedMsg struct{ items []api.Approval }
type approvalDoneMsg struct{ id string }
type approvalDiffLoadedMsg struct {
	id      string
	changes map[string]any
}

// --- Inbox Model ---

// InboxModel shows pending approval requests from agents.
type InboxModel struct {
	client        *api.Client
	items         []api.Approval
	list          *components.List
	loading       bool
	detail        *api.Approval
	filtering     bool
	filterBuf     string
	filtered      []int
	selected      map[string]bool
	confirming    bool
	confirmBulk   bool
	rejecting     bool
	rejectPreview bool
	rejectBuf     string
	bulkRejectIDs []string
	width         int
	height        int
}

// NewInboxModel builds the inbox UI model.
func NewInboxModel(client *api.Client) InboxModel {
	return InboxModel{
		client:   client,
		list:     components.NewList(15),
		selected: make(map[string]bool),
	}
}

func (m InboxModel) Init() tea.Cmd {
	m.loading = true
	return m.loadApprovals
}

func (m InboxModel) Update(msg tea.Msg) (InboxModel, tea.Cmd) {
	switch msg := msg.(type) {
	case approvalsLoadedMsg:
		m.loading = false
		m.items = msg.items
		m.applyFilter(true)
		return m, nil

	case approvalDoneMsg:
		m.detail = nil
		m.rejecting = false
		m.rejectPreview = false
		m.rejectBuf = ""
		m.bulkRejectIDs = nil
		m.confirming = false
		m.selected = make(map[string]bool)
		return m, m.loadApprovals

	case approvalDiffLoadedMsg:
		if m.detail != nil && m.detail.ID == msg.id {
			if m.detail.ChangeDetails == nil {
				m.detail.ChangeDetails = api.JSONMap{}
			}
			m.detail.ChangeDetails["changes"] = msg.changes
		}
		return m, nil

	case tea.KeyMsg:
		if m.confirming {
			switch {
			case isKey(msg, "y"):
				m.confirming = false
				return m.approveSelected()
			case isKey(msg, "n"), isBack(msg):
				m.confirming = false
				return m, nil
			}
			return m, nil
		}
		if m.rejectPreview {
			return m.handleRejectPreview(msg)
		}
		if m.filtering {
			return m.handleFilterInput(msg)
		}
		// Reject input mode
		if m.rejecting {
			return m.handleRejectInput(msg)
		}

		// Detail view
		if m.detail != nil {
			return m.handleDetailKeys(msg)
		}

		// List view
		switch {
		case isDown(msg):
			m.list.Down()
		case isUp(msg):
			m.list.Up()
		case isSpace(msg):
			m.toggleSelected()
		case isEnter(msg):
			if item, ok := m.selectedItem(); ok {
				m.detail = &item
				return m, m.loadApprovalDiff(item.ID)
			}
		case isKey(msg, "a"):
			m.confirming = true
			return m, nil
		case isKey(msg, "A"):
			if m.selectedCount() == 0 {
				m.selectAllFiltered()
			}
			m.confirming = true
			return m, nil
		case isKey(msg, "r"):
			return m.startReject()
		case isKey(msg, "f"):
			m.filtering = true
		case isKey(msg, "b"):
			m.toggleSelectAll()
		case isBack(msg):
			if len(m.selected) > 0 {
				m.selected = make(map[string]bool)
			}
		}
	}
	return m, nil
}

func (m InboxModel) View() string {
	if m.loading {
		return "  " + MutedStyle.Render("Loading approvals...")
	}

	if m.confirming {
		summary := m.approveSummaryRows()
		return components.Indent(components.ConfirmPreviewDialog("Approve Requests", summary, m.approveDiffRows(), m.width), 1)
	}

	if m.rejectPreview {
		summary := []components.TableRow{
			{Label: "Action", Value: "reject"},
			{Label: "Requests", Value: fmt.Sprintf("%d", len(m.bulkRejectIDs))},
			{Label: "Notes", Value: strings.TrimSpace(m.rejectBuf)},
		}
		diffs := []components.DiffRow{
			{Label: "status", From: "pending", To: "rejected"},
			{Label: "review_notes", From: "-", To: strings.TrimSpace(m.rejectBuf)},
		}
		return components.Indent(components.ConfirmPreviewDialog("Reject Requests", summary, diffs, m.width), 1)
	}

	if m.rejecting && m.detail != nil {
		return components.Indent(components.InputDialog("Reject: Enter Review Notes", m.rejectBuf), 1)
	}

	if m.filtering {
		return components.Indent(components.InputDialog("Filter Approvals", m.filterBuf), 1)
	}

	if m.detail != nil {
		return m.renderDetail()
	}

	if len(m.items) == 0 {
		return components.Indent(components.EmptyStateBox(
			"Inbox",
			"No pending approvals.",
			[]string{"Switch tabs with 1-9/0/-", "Open command palette with /"},
			m.width,
		), 1)
	}

	if len(m.filtered) == 0 {
		return components.Indent(components.EmptyStateBox(
			"Inbox",
			"No approvals match the filter.",
			[]string{"Press f to update filter", "Press esc to clear"},
			m.width,
		), 1)
	}

	contentWidth := components.BoxContentWidth(m.width)
	visible := m.list.Visible()
	previewWidth := contentWidth * 35 / 100
	if previewWidth < 38 {
		previewWidth = 38
	}
	if previewWidth > 56 {
		previewWidth = 56
	}

	gap := 3
	tableWidth := contentWidth
	sideBySide := contentWidth >= 110
	if sideBySide {
		tableWidth = contentWidth - previewWidth - gap
		if tableWidth < 60 {
			sideBySide = false
			tableWidth = contentWidth
		}
	}

	sepWidth := 1
	if b := lipgloss.RoundedBorder().Left; b != "" {
		sepWidth = lipgloss.Width(b)
	}

	// TableGrid draws separators only between columns.
	// 4 columns -> 3 separators.
	availableCols := tableWidth - (3 * sepWidth)
	if availableCols < 30 {
		availableCols = 30
	}

	actionWidth := 19
	whoWidth := 14
	atWidth := 11

	titleWidth := availableCols - (actionWidth + whoWidth + atWidth)
	if titleWidth < 12 {
		titleWidth = 12
	}
	cols := []components.TableColumn{
		{Header: "Title", Width: titleWidth, Align: lipgloss.Left},
		{Header: "Action", Width: actionWidth, Align: lipgloss.Left},
		{Header: "Who", Width: whoWidth, Align: lipgloss.Left},
		{Header: "At", Width: atWidth, Align: lipgloss.Left},
	}

	tableRows := make([][]string, 0, len(visible))
	var previewItem *api.Approval
	if item, ok := m.selectedItem(); ok {
		previewItem = &item
	}

	for i := range visible {
		absIdx := m.list.RelToAbs(i)
		item, ok := m.itemAtFilteredIndex(absIdx)
		if !ok {
			continue
		}

		marker := "  "
		if m.selected[item.ID] && m.list.IsSelected(absIdx) {
			marker = ">*"
		} else if m.selected[item.ID] {
			marker = "* "
		} else if m.list.IsSelected(absIdx) {
			marker = "> "
		}

		fullTitle := marker + " " + approvalTitle(item)
		title := components.ClampTextWidthEllipsis(fullTitle, titleWidth)
		action := components.ClampTextWidthEllipsis(humanizeApprovalType(item.RequestType), actionWidth)
		who := components.ClampTextWidthEllipsis(components.SanitizeOneLine(item.AgentName), whoWidth)
		when := item.CreatedAt.Format("01-02 15:04")

		tableRows = append(tableRows, []string{title, action, who, when})
	}

	title := "Inbox"
	countLine := fmt.Sprintf("%d pending", len(m.items))
	if m.filterBuf != "" {
		countLine = fmt.Sprintf("%s · filter: %s", countLine, m.filterBuf)
	}
	if count := m.selectedCount(); count > 0 {
		countLine = fmt.Sprintf("%s · selected: %d", countLine, count)
	}
	countLine = MutedStyle.Render(countLine)
	table := components.TableGrid(cols, tableRows, tableWidth)
	preview := ""
	if previewItem != nil {
		preview = renderApprovalPreview(*previewItem, m.selected[previewItem.ID], previewWidth)
	}

	body := table
	if sideBySide && preview != "" {
		body = lipgloss.JoinHorizontal(lipgloss.Top, table, strings.Repeat(" ", gap), preview)
	} else if preview != "" {
		body = table + "\n\n" + preview
	}

	content := countLine + "\n\n" + body + "\n"
	return components.Indent(components.TitledBox(title, content, m.width), 1)
}

// --- Helpers ---

func (m InboxModel) loadApprovals() tea.Msg {
	items, err := m.client.GetPendingApprovals()
	if err != nil {
		return errMsg{err}
	}
	return approvalsLoadedMsg{items}
}

func (m InboxModel) loadApprovalDiff(id string) tea.Cmd {
	return func() tea.Msg {
		diff, err := m.client.GetApprovalDiff(id)
		if err != nil {
			return errMsg{err}
		}
		return approvalDiffLoadedMsg{id: id, changes: diff.Changes}
	}
}

func (m InboxModel) approveSelected() (InboxModel, tea.Cmd) {
	ids := m.selectedIDs()
	if len(ids) == 0 && m.detail != nil {
		ids = append(ids, m.detail.ID)
	}
	if len(ids) == 0 {
		if item, ok := m.selectedItem(); ok {
			ids = append(ids, item.ID)
		}
	}
	if len(ids) == 0 {
		return m, nil
	}
	m.detail = nil
	return m, func() tea.Msg {
		for _, id := range ids {
			_, err := m.client.ApproveRequest(id)
			if err != nil {
				return errMsg{err}
			}
		}
		return approvalDoneMsg{""}
	}
}

func (m InboxModel) handleDetailKeys(msg tea.KeyMsg) (InboxModel, tea.Cmd) {
	switch {
	case isBack(msg):
		m.detail = nil
	case isKey(msg, "a"):
		m.confirming = true
		return m, nil
	case isKey(msg, "r"):
		m.rejecting = true
		m.rejectBuf = ""
	}
	return m, nil
}

func (m InboxModel) handleRejectInput(msg tea.KeyMsg) (InboxModel, tea.Cmd) {
	if m.detail == nil {
		m.rejecting = false
		m.rejectBuf = ""
		m.bulkRejectIDs = nil
		return m, nil
	}
	switch {
	case isBack(msg):
		if len(m.bulkRejectIDs) > 0 {
			m.detail = nil
		}
		m.rejecting = false
		m.rejectBuf = ""
		m.bulkRejectIDs = nil
	case isEnter(msg):
		ids := m.bulkRejectIDs
		if len(ids) == 0 {
			ids = []string{m.detail.ID}
		}
		m.rejecting = false
		m.rejectPreview = true
		m.bulkRejectIDs = ids
		return m, nil
	case isKey(msg, "backspace"):
		if len(m.rejectBuf) > 0 {
			m.rejectBuf = m.rejectBuf[:len(m.rejectBuf)-1]
		}
	default:
		if len(msg.String()) == 1 || msg.String() == " " {
			m.rejectBuf += msg.String()
		}
	}
	return m, nil
}

func (m InboxModel) renderDetail() string {
	a := m.detail
	var sections []string

	// Approval info table
	rows := []components.TableRow{
		{Label: "ID", Value: a.ID},
		{Label: "Type", Value: a.RequestType},
		{Label: "Status", Value: a.Status},
		{Label: "Agent", Value: a.AgentName},
		{Label: "Requested By", Value: a.RequestedBy},
		{Label: "Created", Value: a.CreatedAt.Format("2006-01-02 15:04")},
	}
	if a.JobID != nil {
		rows = append(rows, components.TableRow{Label: "Job ID", Value: *a.JobID})
	}
	if a.Notes != nil && *a.Notes != "" {
		rows = append(rows, components.TableRow{Label: "Review Notes", Value: *a.Notes})
	}
	sections = append(sections, components.Table("Approval Request", rows, m.width))

	// Change details
	if len(a.ChangeDetails) > 0 {
		var summaryRows []components.TableRow
		var diffRows []components.DiffRow
		nested := make(map[string]any)

		for k, v := range a.ChangeDetails {
			if k == "changes" {
				// Diff object with from/to pairs
				if changesMap, ok := v.(map[string]any); ok {
					for field, diff := range changesMap {
						if diffObj, ok := diff.(map[string]any); ok {
							from := formatAny(diffObj["from"])
							to := formatAny(diffObj["to"])
							if from == to {
								continue
							}
							diffRows = append(diffRows, components.DiffRow{
								Label: field,
								From:  from,
								To:    to,
							})
						}
					}
				}
				continue
			}
			switch val := v.(type) {
			case map[string]any:
				nested[k] = val
			case []any:
				parts := make([]string, len(val))
				for i, item := range val {
					parts[i] = fmt.Sprintf("%v", item)
				}
				summaryRows = append(summaryRows, components.TableRow{Label: k, Value: strings.Join(parts, ", ")})
			default:
				summaryRows = append(summaryRows, components.TableRow{Label: k, Value: fmt.Sprintf("%v", v)})
			}
		}

		if len(summaryRows) > 0 {
			sections = append(sections, components.Table("Change Details", summaryRows, m.width))
		}

		// Diff table for update requests
		if len(diffRows) > 0 {
			sections = append(sections, components.DiffTable("Changes", diffRows, m.width))
		}

		// Render each nested object as its own titled table
		for k, v := range nested {
			if obj, ok := v.(map[string]any); ok {
				var nestedRows []components.TableRow
				for sk, sv := range obj {
					switch sval := sv.(type) {
					case []any:
						parts := make([]string, len(sval))
						for i, item := range sval {
							parts[i] = fmt.Sprintf("%v", item)
						}
						nestedRows = append(nestedRows, components.TableRow{Label: sk, Value: strings.Join(parts, ", ")})
					default:
						nestedRows = append(nestedRows, components.TableRow{Label: sk, Value: fmt.Sprintf("%v", sv)})
					}
				}
				if len(nestedRows) > 0 {
					sections = append(sections, components.Table(k, nestedRows, m.width))
				}
			}
		}
	}

	// Hint
	sections = append(sections, "  "+MutedStyle.Render("a approve  |  r reject  |  esc back"))

	return components.Indent(strings.Join(sections, "\n\n"), 1)
}

func formatAny(v any) string {
	switch val := v.(type) {
	case map[string]any:
		lines := metadataLinesPlain(val, 0)
		return strings.Join(lines, "\n")
	case []any:
		parts := make([]string, len(val))
		for i, item := range val {
			parts[i] = fmt.Sprintf("%v", item)
		}
		return strings.Join(parts, ", ")
	case nil:
		return "-"
	default:
		return fmt.Sprintf("%v", v)
	}
}

func formatApprovalLine(a api.Approval) string {
	name := ""
	if n, ok := a.ChangeDetails["name"]; ok {
		name = fmt.Sprintf(": %q", components.SanitizeText(fmt.Sprintf("%v", n)))
	}
	return fmt.Sprintf(
		"[%s] %s%s",
		components.SanitizeText(a.RequestType),
		components.SanitizeText(a.Status),
		name,
	)
}

func approvalTitle(a api.Approval) string {
	if v, ok := a.ChangeDetails["name"]; ok {
		s := strings.TrimSpace(fmt.Sprintf("%v", v))
		if s != "" {
			return components.SanitizeOneLine(s)
		}
	}
	if v, ok := a.ChangeDetails["title"]; ok {
		s := strings.TrimSpace(fmt.Sprintf("%v", v))
		if s != "" {
			return components.SanitizeOneLine(s)
		}
	}
	if v, ok := a.ChangeDetails["entity_name"]; ok {
		s := strings.TrimSpace(fmt.Sprintf("%v", v))
		if s != "" {
			return components.SanitizeOneLine(s)
		}
	}

	reqType := strings.TrimSpace(components.SanitizeText(a.RequestType))

	// More descriptive fallbacks for request types that don't carry names/titles.
	switch reqType {
	case "create_relationship", "update_relationship":
		relType := strings.TrimSpace(fmt.Sprintf("%v", a.ChangeDetails["relationship_type"]))
		srcType := strings.TrimSpace(fmt.Sprintf("%v", a.ChangeDetails["source_type"]))
		tgtType := strings.TrimSpace(fmt.Sprintf("%v", a.ChangeDetails["target_type"]))

		relType = components.SanitizeOneLine(relType)
		srcType = components.SanitizeOneLine(srcType)
		tgtType = components.SanitizeOneLine(tgtType)

		if relType != "" && relType != "<nil>" {
			if srcType != "" && srcType != "<nil>" && tgtType != "" && tgtType != "<nil>" {
				return components.SanitizeOneLine(fmt.Sprintf("%s (%s -> %s)", relType, srcType, tgtType))
			}
			return relType
		}
	case "create_log", "update_log":
		logType := strings.TrimSpace(fmt.Sprintf("%v", a.ChangeDetails["log_type"]))
		logType = components.SanitizeOneLine(logType)
		if logType != "" && logType != "<nil>" {
			return components.SanitizeOneLine("log: " + logType)
		}
	}

	// Default: make it human readable.
	if reqType != "" {
		return components.SanitizeOneLine(humanizeApprovalType(reqType))
	}
	return ""
}

func humanizeApprovalType(t string) string {
	t = strings.TrimSpace(components.SanitizeText(t))
	if t == "" {
		return ""
	}
	parts := strings.Split(strings.ToLower(t), "_")
	for i := range parts {
		if parts[i] == "" {
			continue
		}
		parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
	}
	return strings.Join(parts, " ")
}

func renderApprovalPreview(a api.Approval, picked bool, width int) string {
	if width <= 0 {
		return ""
	}

	title := components.SanitizeOneLine(approvalTitle(a))
	action := components.SanitizeOneLine(humanizeApprovalType(a.RequestType))
	who := components.SanitizeOneLine(a.AgentName)
	status := components.SanitizeOneLine(a.Status)
	when := a.CreatedAt.Format("01-02 15:04")

	var lines []string
	lines = append(lines, MetaKeyStyle.Render("Selected"))
	for i, part := range wrapPreviewText(title, width) {
		if i == 0 {
			lines = append(lines, SelectedStyle.Render(part))
			continue
		}
		lines = append(lines, SelectedStyle.Render(part))
	}
	lines = append(lines, "")

	lines = append(lines, renderPreviewRow("Action", action, width))
	lines = append(lines, renderPreviewRow("Who", who, width))
	lines = append(lines, renderPreviewRow("At", when, width))
	lines = append(lines, renderPreviewRow("Status", status, width))
	if picked {
		lines = append(lines, renderPreviewRow("In batch", "yes", width))
	}

	if scopes := previewListValue(a.ChangeDetails, "scopes"); scopes != "" {
		lines = append(lines, renderPreviewRow("Scopes", scopes, width))
	}
	if tags := previewListValue(a.ChangeDetails, "tags"); tags != "" {
		lines = append(lines, renderPreviewRow("Tags", tags, width))
	}
	if typ := previewStringValue(a.ChangeDetails, "type"); typ != "" {
		lines = append(lines, renderPreviewRow("Type", typ, width))
	}
	if rel := previewStringValue(a.ChangeDetails, "relationship_type"); rel != "" {
		lines = append(lines, renderPreviewRow("Rel", rel, width))
	}
	if src := previewStringValue(a.ChangeDetails, "source_type"); src != "" {
		lines = append(lines, renderPreviewRow("From", src, width))
	}
	if tgt := previewStringValue(a.ChangeDetails, "target_type"); tgt != "" {
		lines = append(lines, renderPreviewRow("To", tgt, width))
	}
	if logType := previewStringValue(a.ChangeDetails, "log_type"); logType != "" {
		lines = append(lines, renderPreviewRow("Log", logType, width))
	}

	return padPreviewLines(lines, width)
}

func wrapPreviewText(text string, width int) []string {
	text = components.SanitizeOneLine(text)
	if width <= 0 || text == "" {
		return nil
	}
	if lipgloss.Width(text) <= width {
		return []string{text}
	}

	var out []string
	var line strings.Builder
	lineW := 0
	for _, r := range text {
		rw := lipgloss.Width(string(r))
		if rw < 1 {
			rw = 1
		}
		if lineW+rw > width && lineW > 0 {
			out = append(out, strings.TrimRight(line.String(), " "))
			line.Reset()
			lineW = 0
			if r == ' ' {
				continue
			}
		}
		line.WriteRune(r)
		lineW += rw
	}
	if line.Len() > 0 {
		out = append(out, strings.TrimRight(line.String(), " "))
	}
	return out
}

func renderPreviewRow(label, value string, width int) string {
	label = components.SanitizeOneLine(label)
	value = components.SanitizeOneLine(value)

	prefixWidth := lipgloss.Width(label) + 2 // ": "
	maxValue := width - prefixWidth
	if maxValue < 4 {
		maxValue = 4
	}
	value = components.ClampTextWidthEllipsis(value, maxValue)
	return MetaKeyStyle.Render(label) + MetaPunctStyle.Render(": ") + MetaValueStyle.Render(value)
}

func previewStringValue(m api.JSONMap, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	s := strings.TrimSpace(fmt.Sprintf("%v", v))
	if s == "" || s == "<nil>" {
		return ""
	}
	return components.SanitizeOneLine(s)
}

func previewListValue(m api.JSONMap, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	items, ok := v.([]any)
	if !ok || len(items) == 0 {
		return ""
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		s := strings.TrimSpace(fmt.Sprintf("%v", item))
		if s == "" || s == "<nil>" {
			continue
		}
		out = append(out, components.SanitizeOneLine(s))
	}
	return strings.Join(out, ", ")
}

func padPreviewLines(lines []string, width int) string {
	if width <= 0 || len(lines) == 0 {
		return ""
	}
	padded := make([]string, 0, len(lines))
	for _, line := range lines {
		line = components.ClampTextWidth(line, width)
		if w := lipgloss.Width(line); w < width {
			line += strings.Repeat(" ", width-w)
		}
		padded = append(padded, line)
	}
	return strings.Join(padded, "\n")
}

func (m *InboxModel) applyFilter(resetSelection bool) {
	if resetSelection {
		m.selected = make(map[string]bool)
	}
	m.filtered = m.filtered[:0]
	labels := make([]string, 0, len(m.items))
	filter := parseApprovalFilter(m.filterBuf)
	for i, a := range m.items {
		if matchesApprovalFilter(a, filter) {
			m.filtered = append(m.filtered, i)
			labels = append(labels, formatApprovalLine(a))
		}
	}
	m.list.SetItems(labels)
}

func (m *InboxModel) selectedItem() (api.Approval, bool) {
	idx := m.list.Selected()
	return m.itemAtFilteredIndex(idx)
}

func (m *InboxModel) selectAllFiltered() {
	for _, itemIdx := range m.filtered {
		if itemIdx < 0 || itemIdx >= len(m.items) {
			continue
		}
		m.selected[m.items[itemIdx].ID] = true
	}
}

func (m *InboxModel) toggleSelectAll() {
	if len(m.filtered) == 0 {
		return
	}
	allSelected := true
	for _, itemIdx := range m.filtered {
		if itemIdx < 0 || itemIdx >= len(m.items) {
			continue
		}
		if !m.selected[m.items[itemIdx].ID] {
			allSelected = false
			break
		}
	}
	if allSelected {
		m.selected = make(map[string]bool)
		return
	}
	m.selectAllFiltered()
}

func (m *InboxModel) itemAtFilteredIndex(filteredIdx int) (api.Approval, bool) {
	if filteredIdx < 0 || filteredIdx >= len(m.filtered) {
		return api.Approval{}, false
	}
	itemIdx := m.filtered[filteredIdx]
	if itemIdx < 0 || itemIdx >= len(m.items) {
		return api.Approval{}, false
	}
	return m.items[itemIdx], true
}

func (m *InboxModel) toggleSelected() {
	item, ok := m.selectedItem()
	if !ok {
		return
	}
	if m.selected[item.ID] {
		delete(m.selected, item.ID)
		return
	}
	m.selected[item.ID] = true
}

func (m *InboxModel) selectedIDs() []string {
	if len(m.selected) == 0 {
		return nil
	}
	ids := make([]string, 0, len(m.selected))
	for _, item := range m.items {
		if m.selected[item.ID] {
			ids = append(ids, item.ID)
		}
	}
	return ids
}

func (m *InboxModel) selectedCount() int {
	return len(m.selectedIDs())
}

func (m *InboxModel) startReject() (InboxModel, tea.Cmd) {
	ids := m.selectedIDs()
	if len(ids) > 0 {
		m.bulkRejectIDs = ids
		m.rejecting = true
		m.rejectPreview = false
		m.rejectBuf = ""
		m.detail = &api.Approval{ID: ids[0]}
		return *m, nil
	}
	if item, ok := m.selectedItem(); ok {
		m.detail = &item
		m.rejecting = true
		m.rejectPreview = false
		m.rejectBuf = ""
		return *m, nil
	}
	return *m, nil
}

func (m InboxModel) handleRejectPreview(msg tea.KeyMsg) (InboxModel, tea.Cmd) {
	switch {
	case isKey(msg, "y"):
		ids := append([]string(nil), m.bulkRejectIDs...)
		notes := m.rejectBuf
		m.rejectPreview = false
		m.rejectBuf = ""
		m.detail = nil
		m.bulkRejectIDs = nil
		return m, func() tea.Msg {
			for _, id := range ids {
				_, err := m.client.RejectRequest(id, notes)
				if err != nil {
					return errMsg{err}
				}
			}
			return approvalDoneMsg{""}
		}
	case isKey(msg, "n"), isBack(msg):
		m.rejectPreview = false
		m.rejecting = true
		return m, nil
	}
	return m, nil
}

func (m InboxModel) approveSummaryRows() []components.TableRow {
	ids := m.selectedIDs()
	if len(ids) == 0 && m.detail != nil {
		ids = append(ids, m.detail.ID)
	}
	if len(ids) == 0 {
		if item, ok := m.selectedItem(); ok {
			ids = append(ids, item.ID)
		}
	}

	rows := []components.TableRow{
		{Label: "Action", Value: "approve"},
		{Label: "Requests", Value: fmt.Sprintf("%d", len(ids))},
	}
	if len(ids) == 1 {
		rows = append(rows, components.TableRow{Label: "Request ID", Value: ids[0]})
	}
	return rows
}

func (m InboxModel) approveDiffRows() []components.DiffRow {
	if m.detail == nil {
		return nil
	}
	raw, ok := m.detail.ChangeDetails["changes"]
	if !ok {
		return nil
	}
	changesMap, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	rows := make([]components.DiffRow, 0, len(changesMap))
	for field, diff := range changesMap {
		diffObj, ok := diff.(map[string]any)
		if !ok {
			continue
		}
		from := formatAny(diffObj["from"])
		to := formatAny(diffObj["to"])
		if from == to {
			continue
		}
		rows = append(rows, components.DiffRow{
			Label: field,
			From:  from,
			To:    to,
		})
	}
	return rows
}

func (m InboxModel) handleFilterInput(msg tea.KeyMsg) (InboxModel, tea.Cmd) {
	switch {
	case isBack(msg):
		m.filtering = false
		m.filterBuf = ""
		m.applyFilter(true)
	case isEnter(msg):
		m.filtering = false
		m.applyFilter(true)
	case isKey(msg, "backspace"):
		if len(m.filterBuf) > 0 {
			m.filterBuf = m.filterBuf[:len(m.filterBuf)-1]
			m.applyFilter(true)
		}
	default:
		if len(msg.String()) == 1 || msg.String() == " " {
			m.filterBuf += msg.String()
			m.applyFilter(true)
		}
	}
	return m, nil
}

type approvalFilter struct {
	agent string
	req   string
	since *time.Time
	terms []string
}

func parseApprovalFilter(raw string) approvalFilter {
	filter := approvalFilter{}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return filter
	}
	for _, token := range strings.Fields(raw) {
		switch {
		case strings.HasPrefix(token, "agent:"):
			filter.agent = strings.ToLower(strings.TrimPrefix(token, "agent:"))
		case strings.HasPrefix(token, "type:"):
			filter.req = strings.ToLower(strings.TrimPrefix(token, "type:"))
		case strings.HasPrefix(token, "since:"):
			val := strings.TrimPrefix(token, "since:")
			if t := parseFilterTime(val); t != nil {
				filter.since = t
			}
		default:
			filter.terms = append(filter.terms, strings.ToLower(token))
		}
	}
	return filter
}

func parseFilterTime(value string) *time.Time {
	value = strings.TrimSpace(strings.ToLower(value))
	now := time.Now()
	switch value {
	case "today":
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		return &start
	case "yesterday":
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Add(-24 * time.Hour)
		return &start
	}
	if strings.HasSuffix(value, "h") {
		if dur, err := time.ParseDuration(value); err == nil {
			t := now.Add(-dur)
			return &t
		}
	}
	if strings.HasSuffix(value, "d") {
		days := strings.TrimSuffix(value, "d")
		if n, err := time.ParseDuration(days + "h"); err == nil {
			t := now.Add(-24 * n)
			return &t
		}
	}
	if t, err := time.ParseInLocation("2006-01-02", value, now.Location()); err == nil {
		return &t
	}
	return nil
}

func matchesApprovalFilter(a api.Approval, filter approvalFilter) bool {
	if filter.agent != "" && !strings.Contains(strings.ToLower(a.AgentName), filter.agent) {
		return false
	}
	if filter.req != "" && !strings.Contains(strings.ToLower(a.RequestType), filter.req) {
		return false
	}
	if filter.since != nil && a.CreatedAt.Before(*filter.since) {
		return false
	}
	if len(filter.terms) > 0 {
		search := strings.ToLower(formatApprovalLine(a))
		for _, term := range filter.terms {
			if !strings.Contains(search, term) {
				return false
			}
		}
	}
	return true
}
