package common

import "github.com/slack-go/slack"

// NewInputBlock this is the same as slack.NewInputBlock but add the optional arg
func NewInputBlock(
	blockID string,
	label, hint *slack.TextBlockObject,
	element slack.BlockElement,
	optional bool,
) *slack.InputBlock {
	return &slack.InputBlock{
		Type:     slack.MBTInput,
		BlockID:  blockID,
		Label:    label,
		Element:  element,
		Hint:     hint,
		Optional: optional,
	}
}
