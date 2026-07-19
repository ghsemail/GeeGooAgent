package chattui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ghsemail/GeeGooAgent/internal/cli/chatcmd"
	"github.com/ghsemail/GeeGooAgent/internal/cli/chatui"
	"github.com/ghsemail/GeeGooAgent/internal/config"
)

var (
	styleDim = lipgloss.NewStyle().Foreground(lipgloss.Color(chatui.ColorDim))
	styleErr = lipgloss.NewStyle().Foreground(lipgloss.Color(chatui.ColorErr)).Bold(true)
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if msg.Width > 12 {
			m.input.Width = msg.Width - 8
		}
		m.rebuildBanner()
		m.layoutViewport()
		m.refreshViewport()
		return m, nil

	case statusTickMsg:
		if slotBusy(m.slots) {
			return m, tickStatus()
		}
		return m, nil

	case ProgressMsg:
		if msg.Slot >= 0 && msg.Slot < len(m.slots) {
			m.slots[msg.Slot].ApplyProgress(msg.Event, msg.Data, m.display)
			if msg.Slot == m.active {
				m.scrollFollow = true
				m.refreshViewport()
			}
		}
		return m, nil

	case TurnDoneMsg:
		if msg.Slot < 0 || msg.Slot >= len(m.slots) {
			return m, nil
		}
		s := m.slots[msg.Slot]
		reply := strings.TrimSpace(msg.Reply)
		if msg.Err != "" {
			s.Err = msg.Err
		}
		if reply != "" {
			s.upsertTurnReply(reply)
		}
		s.finalizeLiveSections()
		if msg.Slot == m.active {
			m.input.Focus()
			m.scrollFollow = true
			m.refreshViewport()
		}
		if msg.Slot >= 0 && msg.Slot < len(m.slots) {
			s := m.slots[msg.Slot]
			if msg.PlanPending {
				s.PlanPending = true
				s.PlanTools = append([]string(nil), msg.PlanTools...)
			} else {
				s.PlanPending = false
				s.PlanTools = nil
			}
		}
		return m, nil

	case InfoMsg:
		m.info = msg.Text
		return m, nil

	case DisplayUpdatedMsg:
		m.display = msg.Display
		m.display.Normalize()
		if h := m.activeHost(); h != nil && h.Repl != nil && h.Repl.UI != nil {
			h.Repl.UI.ApplyDisplay(m.display)
		}
		m.refreshViewport()
		return m, nil

	case NewSessionMsg:
		if msg.Slot != nil {
			msg.Slot.Index = len(m.slots)
			m.slots = append(m.slots, msg.Slot)
			m.active = len(m.slots) - 1
			m.sessionPicker = false
			m.info = "已新建会话 " + msg.Slot.shortTitle()
			m.bannerOpts = bannerOptsFromRepl(msg.Slot.Repl)
			m.rebuildBanner()
			m.refreshViewport()
		}
		return m, nil

	case approvalTickMsg:
		host := m.activeHost()
		if host != nil && !m.approvalPending && !m.clarifyPending && !m.clarifyAwaitingText && !m.activeSlotPlanPending() {
			if tool, args, ok := host.PollApproval(); ok {
				m.approvalPending = true
				m.approvalTool = tool
				m.approvalArgs = args
				m.info = fmt.Sprintf("写操作确认: %s — 输入 y 执行 / n 跳过", tool)
			} else if q, choices, ok := host.PollClarify(); ok {
				m.clarifyQuestion = q
				m.clarifyChoices = choices
				if len(choices) == 0 {
					m.clarifyAwaitingText = true
					m.info = "请回答上方问题"
				} else {
					m.clarifyPending = true
					m.clarifyFocus = 0
					m.info = ""
				}
			}
		}
		return m, tickApproval()

	case tea.MouseMsg:
		return m.handleMouse(msg)

	case tea.KeyMsg:
		if m.quitting {
			return m, tea.Quit
		}
		if m.sessionPicker {
			return m.handlePickerKeys(msg)
		}
		if s := m.activeSlot(); s != nil && s.PlanPending {
			switch strings.ToLower(msg.String()) {
			case "y", "yes":
				return m.submitPlanAnswer(true)
			case "n", "no":
				return m.submitPlanAnswer(false)
			}
			if msg.Type == tea.KeyEsc {
				return m.submitPlanAnswer(false)
			}
			return m, nil
		}
		if m.approvalPending {
			switch strings.ToLower(msg.String()) {
			case "y", "yes":
				if h := m.activeHost(); h != nil {
					h.AnswerApproval(true)
				}
				m.approvalPending = false
				m.info = "已批准"
				return m, nil
			case "n", "no":
				if h := m.activeHost(); h != nil {
					h.AnswerApproval(false)
				}
				m.approvalPending = false
				m.info = "已跳过"
				return m, nil
			}
			if msg.Type == tea.KeyEsc {
				if h := m.activeHost(); h != nil {
					h.AnswerApproval(false)
				}
				m.approvalPending = false
				m.info = "已跳过"
				return m, nil
			}
			return m, nil
		}
		if m.clarifyPending {
			if m.handleClarifyKey(msg) {
				m.refreshViewport()
				return m, nil
			}
		}
		if m.clarifyAwaitingText && msg.Type == tea.KeyEsc {
			m.submitClarifyAnswer("", true)
			m.input.SetValue("")
			m.refreshViewport()
			return m, nil
		}
		switch msg.Type {
		case tea.KeyCtrlC:
			s := m.activeSlot()
			if s != nil && s.Busy {
				select {
				case s.CancelCh <- struct{}{}:
				default:
				}
				m.info = "正在中断…"
				return m, nil
			}
			m.quitting = true
			return m, tea.Quit
		case tea.KeyCtrlX:
			m.sessionPicker = true
			m.pickerFocus = m.active
			m.info = ""
			return m, nil
		case tea.KeyCtrlN:
			return m, m.cmdNewSession()
		case tea.KeyEsc:
			s := m.activeSlot()
			if s != nil && s.Busy {
				select {
				case s.CancelCh <- struct{}{}:
				default:
				}
				return m, nil
			}
			return m, nil
		case tea.KeyPgUp:
			m.scrollFollow = false
			m.vp.ScrollUp(5)
			return m, nil
		case tea.KeyPgDown:
			m.vp.ScrollDown(5)
			if m.vp.AtBottom() {
				m.scrollFollow = true
			}
			return m, nil
		case tea.KeyUp:
			if m.slashMenuOpen() {
				if m.slashPick > 0 {
					m.slashPick--
				}
				return m, nil
			}
			m.moveFocus(-1)
			m.refreshViewport()
			return m, nil
		case tea.KeyDown:
			if m.slashMenuOpen() {
				if m.slashPick < len(m.slashMatches())-1 {
					m.slashPick++
				}
				return m, nil
			}
			m.moveFocus(1)
			m.refreshViewport()
			return m, nil
		case tea.KeyTab:
			if m.slashMenuOpen() {
				return m.acceptSlashSuggestion()
			}
		case tea.KeyEnter:
			if msg.Alt {
				return m, nil
			}
			if m.clarifyAwaitingText {
				text := strings.TrimSpace(m.input.Value())
				m.input.SetValue("")
				m.submitClarifyAnswer(text, true)
				m.refreshViewport()
				return m, nil
			}
			text := strings.TrimSpace(m.input.Value())
			if text == "" {
				return m, nil
			}
			m.input.SetValue("")
			return m.handleSubmit(text)
		case tea.KeySpace:
			s := m.activeSlot()
			if s != nil && strings.TrimSpace(m.input.Value()) == "" && s.Focus >= 0 && s.Focus < len(s.Blocks) {
				s.Blocks[s.Focus].ToggleExpand(m.display)
				m.refreshViewport()
				return m, nil
			}
		}
	}

	s := m.activeSlot()
	if s == nil || (!s.Busy && !m.clarifyAwaitingText) {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		m.clampSlashPick()
		return m, cmd
	}
	if m.clarifyAwaitingText {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.sessionPicker || m.approvalPending || m.clarifyPending || m.activeSlotPlanPending() {
		return m, nil
	}
	switch msg.Action {
	case tea.MouseActionPress:
		if msg.Button == tea.MouseButtonLeft {
			// Click in transcript region toggles nearest focused expandable block
			s := m.activeSlot()
			if s != nil && s.Focus >= 0 && s.Focus < len(s.Blocks) {
				k := s.Blocks[s.Focus].Kind
				if k == KindThinking || k == KindTools {
					s.Blocks[s.Focus].ToggleExpand(m.display)
					m.refreshViewport()
				}
			}
		}
	case tea.MouseActionMotion:
		// wheel via viewport
	}
	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg)
	if !m.vp.AtBottom() {
		m.scrollFollow = false
	} else {
		m.scrollFollow = true
	}
	return m, cmd
}

func (m Model) handlePickerKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.sessionPicker = false
		return m, nil
	case tea.KeyUp:
		if m.pickerFocus > 0 {
			m.pickerFocus--
		}
		return m, nil
	case tea.KeyDown:
		if m.pickerFocus < len(m.slots) { // include "+ new"
			m.pickerFocus++
		}
		return m, nil
	case tea.KeyCtrlN:
		m.sessionPicker = false
		return m, m.cmdNewSession()
	case tea.KeyCtrlD:
		return m.closePickerSession()
	case tea.KeyEnter:
		if m.pickerFocus >= len(m.slots) {
			m.sessionPicker = false
			return m, m.cmdNewSession()
		}
		m.active = m.pickerFocus
		m.sessionPicker = false
		m.info = "切换到 " + m.slots[m.active].shortTitle()
		m.bannerOpts = bannerOptsFromRepl(m.slots[m.active].Repl)
		m.rebuildBanner()
		m.refreshViewport()
		return m, nil
	}
	return m, nil
}

func (m Model) closePickerSession() (tea.Model, tea.Cmd) {
	if len(m.slots) <= 1 {
		m.info = "至少保留一个 live session"
		return m, nil
	}
	idx := m.pickerFocus
	if idx < 0 || idx >= len(m.slots) {
		idx = m.active
	}
	s := m.slots[idx]
	if s.Busy {
		m.info = "会话仍在运行，先中断再关闭"
		return m, nil
	}
	wasActive := m.active
	s.Repl.CloseSession()
	if s.SubmitCh != nil {
		close(s.SubmitCh)
		s.SubmitCh = nil
	}
	m.slots = append(m.slots[:idx], m.slots[idx+1:]...)
	for i := range m.slots {
		m.slots[i].Index = i
	}
	if wasActive == idx {
		if wasActive >= len(m.slots) {
			m.active = len(m.slots) - 1
		} else {
			m.active = wasActive
		}
	} else if wasActive > idx {
		m.active = wasActive - 1
	}
	if m.pickerFocus >= len(m.slots) {
		m.pickerFocus = len(m.slots) - 1
	}
	m.info = "已关闭会话"
	m.refreshViewport()
	return m, nil
}

func (m Model) cmdNewSession() tea.Cmd {
	factory := m.appFactory
	return func() tea.Msg {
		if factory == nil {
			return InfoMsg{Text: "无法新建会话"}
		}
		slot, err := factory("")
		if err != nil {
			return InfoMsg{Text: "新建失败: " + err.Error()}
		}
		return NewSessionMsg{Slot: slot}
	}
}

func (m *Model) moveFocus(delta int) {
	s := m.activeSlot()
	if s == nil || len(s.Blocks) == 0 {
		return
	}
	if s.Focus < 0 {
		if delta > 0 {
			s.Focus = 0
		} else {
			s.Focus = len(s.Blocks) - 1
		}
		return
	}
	s.Focus += delta
	if s.Focus < 0 {
		s.Focus = 0
	}
	if s.Focus >= len(s.Blocks) {
		s.Focus = len(s.Blocks) - 1
	}
}

func (m Model) handleSubmit(text string) (tea.Model, tea.Cmd) {
	if m.clarifyAwaitingText {
		m.submitClarifyAnswer(strings.TrimSpace(text), true)
		m.input.SetValue("")
		m.refreshViewport()
		return m, nil
	}
	if strings.HasPrefix(text, "/") {
		return m.handleSlash(text)
	}
	s := m.activeSlot()
	if s == nil {
		return m, nil
	}
	if s.Busy && !m.clarifyAwaitingText {
		m.info = "请等待当前回合结束，或 Esc 中断"
		return m, nil
	}
	s.Err = ""
	m.info = ""
	s.Blocks = append(s.Blocks, Block{
		ID: fmt.Sprintf("user-%d", s.Seq), Kind: KindUser, Title: "你", Body: text,
	})
	s.Seq++
	if s.Title == "" || s.Title == s.ID {
		s.Title = TruncateRunes(text, 32)
	}
	s.Busy = true
	s.Status = "thinking…"
	s.TurnStartedAt = time.Now()
	s.TurnEndedAt = time.Time{}
	m.scrollFollow = true
	m.refreshViewport()
	go func() { s.SubmitCh <- text }()
	return m, nil
}

func (m Model) submitPlanAnswer(approve bool) (tea.Model, tea.Cmd) {
	s := m.activeSlot()
	if s == nil || !s.PlanPending {
		return m, nil
	}
	text := "n"
	info := "已取消写操作"
	if approve {
		text = "y"
		info = "已确认，执行写操作…"
	}
	s.PlanPending = false
	s.PlanTools = nil
	s.Err = ""
	m.info = info
	s.Blocks = append(s.Blocks, Block{
		ID: fmt.Sprintf("user-%d", s.Seq), Kind: KindUser, Title: "你", Body: text,
	})
	s.Seq++
	s.Busy = true
	s.Status = "thinking…"
	s.TurnStartedAt = time.Now()
	s.TurnEndedAt = time.Time{}
	m.scrollFollow = true
	m.refreshViewport()
	go func() { s.SubmitCh <- text }()
	return m, nil
}

func (m Model) activeSlotPlanPending() bool {
	s := m.activeSlot()
	return s != nil && s.PlanPending
}

func (m Model) handleSlash(text string) (tea.Model, tea.Cmd) {
	fields := strings.Fields(text)
	cmd := strings.ToLower(fields[0])
	args := fields[1:]
	switch cmd {
	case "/exit", "/quit":
		m.quitting = true
		return m, tea.Quit
	case "/sessions", "/switch":
		m.sessionPicker = true
		m.pickerFocus = m.active
		return m, nil
	case "/mouse":
		mode := "toggle"
		if len(args) > 0 {
			mode = args[0]
		}
		cur := m.display.MouseTracking
		if NormalizeMouseMode(mode) == "toggle" {
			m.display.MouseTracking = CycleMouseMode(cur)
		} else {
			m.display.MouseTracking = NormalizeMouseMode(mode)
		}
		m.info = "mouse=" + m.display.MouseTracking + "（off=可选中文字 wheel=滚轮滚动；重启 chat 生效）"
		cp := m.configPath
		disp := m.display
		return m, func() tea.Msg {
			if cp != "" {
				_ = PersistDisplay(cp, disp)
			}
			return DisplayUpdatedMsg{Display: disp}
		}
	case "/stream":
		res := chatcmd.ApplyStream(&m.display, args)
		if !res.OK {
			m.info = res.Message
			return m, nil
		}
		if res.Display != nil {
			m.display = *res.Display
			if h := m.activeHost(); h != nil && h.Repl != nil && h.Repl.UI != nil {
				h.Repl.UI.ApplyDisplay(m.display)
			}
		}
		m.info = res.Message
		m.refreshViewport()
		if res.Persist {
			cp := m.configPath
			disp := m.display
			return m, func() tea.Msg {
				if cp != "" {
					_ = PersistDisplay(cp, disp)
				}
				return DisplayUpdatedMsg{Display: disp}
			}
		}
		return m, nil
	case "/reply":
		res := chatcmd.ApplyReplyFormat(&m.display, args)
		if !res.OK {
			m.info = res.Message
			return m, nil
		}
		if res.Display != nil {
			m.display = *res.Display
			if h := m.activeHost(); h != nil && h.Repl != nil && h.Repl.UI != nil {
				h.Repl.UI.ApplyDisplay(m.display)
			}
		}
		m.info = res.Message
		m.refreshViewport()
		if res.Persist {
			cp := m.configPath
			disp := m.display
			return m, func() tea.Msg {
				if cp != "" {
					_ = PersistDisplay(cp, disp)
				}
				return DisplayUpdatedMsg{Display: disp}
			}
		}
		return m, nil
	case "/details":
		res := chatcmd.ApplyDetails(&m.display, args)
		if !res.OK {
			m.info = res.Message
			return m, nil
		}
		if res.Display != nil {
			m.display = *res.Display
		}
		if res.ShowLast {
			if s := m.activeSlot(); s != nil {
				s.expandLastDetails()
			}
		}
		m.info = res.Message
		m.refreshViewport()
		if res.Persist {
			cp := m.configPath
			disp := m.display
			return m, func() tea.Msg {
				if cp != "" {
					_ = PersistDisplay(cp, disp)
				}
				return DisplayUpdatedMsg{Display: disp}
			}
		}
		return m, nil
	case "/verbose":
		on, ok := parseOnOff(args)
		if !ok {
			m.info = "用法: /verbose on|off"
			return m, nil
		}
		if h := m.activeHost(); h != nil {
			m.info = h.SetVerbose(on)
		}
		ApplyVerboseToDisplay(&m.display, on)
		cp := m.configPath
		disp := m.display
		m.refreshViewport()
		return m, func() tea.Msg {
			if cp != "" {
				_ = PersistDisplay(cp, disp)
			}
			return DisplayUpdatedMsg{Display: disp}
		}
	case "/dry-run":
		on, ok := parseOnOff(args)
		if !ok {
			m.info = "用法: /dry-run on|off"
			return m, nil
		}
		if h := m.activeHost(); h != nil {
			m.info = h.SetDryRun(on)
		} else {
			m.info = fmt.Sprintf("dry_run=%v", on)
		}
		return m, nil
	case "/session":
		if h := m.activeHost(); h != nil {
			m.info = h.SessionInfo() + fmt.Sprintf(" · live=%d/%d", m.active+1, len(m.slots))
		} else {
			m.info = "no session"
		}
		return m, nil
	default:
		host := m.activeHost()
		if host == nil {
			m.info = "无活动会话"
			return m, nil
		}
		quit, output := host.HandleSlash(text)
		if quit {
			m.quitting = true
			return m, tea.Quit
		}
		m.showSlashOutput(output)
		if cmd == "/model" || cmd == "/think" {
			if host.Repl != nil {
				m.bannerOpts = bannerOptsFromRepl(host.Repl)
				m.rebuildBanner()
			}
		}
		m.refreshViewport()
		return m, nil
	}
}

func (m *Model) showSlashOutput(output string) {
	output = strings.TrimSpace(output)
	if output == "" {
		return
	}
	if !strings.Contains(output, "\n") && len(output) < 120 {
		m.info = output
		return
	}
	s := m.activeSlot()
	if s == nil {
		m.info = TruncateRunes(output, 120)
		return
	}
	expanded := true
	s.Blocks = append(s.Blocks, Block{
		ID:           fmt.Sprintf("slash-%d", s.Seq),
		Kind:         KindActivity,
		Title:        "命令输出",
		Body:         output,
		UserExpanded: &expanded,
	})
	s.Seq++
	m.scrollFollow = true
}

func (m *Model) layoutViewport() {
	footer := m.footerLineCount()
	h := m.height - footer - m.fixedWelcomeBannerLines()
	if h < minViewportHeight {
		h = minViewportHeight
	}
	w := m.width
	if w < 20 {
		w = 20
	}
	m.vp.Width = w
	m.vp.Height = h
}

func (m Model) footerLineCount() int {
	n := 1 // status bar
	if m.approvalPending || m.clarifyPending || m.clarifyAwaitingText || m.info != "" || m.activeSlotPlanPending() {
		n++
	}
	if !m.approvalPending && !m.clarifyPending && !m.activeSlotPlanPending() {
		n++ // input chrome
	}
	if m.slashMenuOpen() {
		n += len(m.slashMatches())
	}
	return n
}

func (m *Model) refreshViewport() {
	m.layoutViewport()
	content := m.renderTranscript()
	if m.scrollFollow && !m.showWelcomeBanner() {
		content = chatui.AnchorContentBottomKeepingPrefix(m.banner, content, m.vp.Height)
	}
	m.vp.SetContent(content)
	if m.scrollFollow {
		if m.showWelcomeBanner() {
			m.vp.YOffset = 0
		} else {
			m.vp.GotoBottom()
		}
	}
}

func (m Model) View() string {
	if m.sessionPicker {
		var b strings.Builder
		b.WriteString(formatSessionList(m.slots, m.active, m.pickerFocus))
		return b.String()
	}

	var b strings.Builder
	if m.canFixWelcomeBanner() {
		b.WriteString(welcomeBannerTopPadding())
		b.WriteString(m.banner)
	}
	b.WriteString(m.vp.View())
	b.WriteByte('\n')

	if m.approvalPending {
		b.WriteString(styleErr.Render(fmt.Sprintf("⚠ 确认写操作 %s: %s  [y/n]", m.approvalTool, m.approvalArgs)))
		b.WriteByte('\n')
	} else if s := m.activeSlot(); s != nil && s.PlanPending {
		b.WriteString(styleErr.Render(fmt.Sprintf("⚠ Plan 待确认: %s  [y 执行 / n 取消]", formatPlanTools(s.PlanTools))))
		b.WriteByte('\n')
	} else if m.clarifyPending {
		b.WriteString(chatui.RenderClarifyPanel(m.clarifyQuestion, m.clarifyDisplayOptions(), m.clarifyFocus, m.width))
		b.WriteByte('\n')
	} else if m.clarifyAwaitingText && m.clarifyQuestion != "" {
		b.WriteString(chatui.RenderClarifyPanel(m.clarifyQuestion, nil, 0, m.width))
		b.WriteByte('\n')
	} else if m.info != "" {
		b.WriteString(styleDim.Render(m.info))
		b.WriteByte('\n')
	}

	b.WriteString(chatui.RenderHermesStatusBar(m.statusBarOpts(), m.width))
	b.WriteByte('\n')
	if matches := m.slashMatches(); m.slashMenuOpen() {
		b.WriteString(renderSlashMenu(matches, m.slashPick, m.width))
		b.WriteByte('\n')
	}
	if !m.approvalPending && !m.clarifyPending && !m.activeSlotPlanPending() {
		opts := m.statusBarOpts()
		b.WriteString(renderInputLine(m.input, opts.Model, m.width))
	}
	return b.String()
}

func slotBusy(slots []*LiveSlot) bool {
	for _, s := range slots {
		if s != nil && s.Busy {
			return true
		}
	}
	return false
}

func findLastKind(blocks []Block, kind SectionKind) int {
	for i := len(blocks) - 1; i >= 0; i-- {
		if blocks[i].Kind == kind {
			return i
		}
	}
	return -1
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

var _ = textinput.New
var _ = config.ModeCollapsed
