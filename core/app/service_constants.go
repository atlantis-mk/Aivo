package app

import "errors"

var ErrMaxStepsExceeded = errors.New("maximum tool calling steps exceeded")

const (
	sessionMetadataRememberedDeferredTools = "rememberedDeferredTools"
	sessionMetadataActiveSkills            = "activeSkills"
)
