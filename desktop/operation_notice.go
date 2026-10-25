package main

import (
	"strings"

	"github.com/dshills/vellum/ui"
)

func (s *screen) operationNotice(message, help string, failed bool) ui.Node {
	if message == "" {
		return quiet("messages", "")
	}
	notice := &s.statusNotice
	notice.ID, notice.Text = "operation-notice", message
	notice.Tone = ui.NoticeInformation
	if failed {
		notice.Tone = ui.NoticeError
	} else if strings.HasPrefix(message, "Autosave") {
		notice.Tone = ui.NoticeWarning
	} else if strings.HasPrefix(message, "Saved ") {
		notice.Tone = ui.NoticeSuccess
	}
	notice.Style = ui.Style{GrowX: ui.Some(float32(1)), GrowY: ui.Some(float32(0)), Padding: ui.Some(ui.Insets{Top: 4, Bottom: 4, Left: 8, Right: 8}), Gap: ui.Some(float32(6)), Align: ui.Some(ui.AlignCenter)}
	node, err := notice.View()
	if err != nil {
		panic(err) // Known tone and nonempty text, with no optional action.
	}
	node.Children[0].Style.MinSize = ui.Some(ui.Size{Width: 16, Height: 16})
	node.Children[0].Style.MaxSize = node.Children[0].Style.MinSize
	text := &node.Children[1]
	text.ID = "messages"
	text.Accessibility.Help = help
	text.Style.MaxSize = ui.Some(ui.Size{Height: 48})
	text.Style.Typography = ui.Some(ui.Typography{Size: 12})
	text.Style.Foreground = ui.Some(muted)
	if failed {
		text.Style.Foreground = ui.Some(resolvePalette("light", false).Danger)
	}
	return node
}
