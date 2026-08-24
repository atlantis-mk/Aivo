package app

import "sync"

type preparedPatchPlan struct {
	PatchText string
	Changes   []patchFileChange
}

var preparedPatchPlans sync.Map
var preparedExpectedHashes sync.Map

func storePreparedPatchPlan(toolCallID string, patchText string, changes []patchFileChange) {
	if toolCallID == "" {
		return
	}
	copied := append([]patchFileChange(nil), changes...)
	preparedPatchPlans.Store(toolCallID, preparedPatchPlan{PatchText: patchText, Changes: copied})
}

func preparedPatchPlanFor(toolCallID string, patchText string) ([]patchFileChange, bool) {
	if toolCallID == "" {
		return nil, false
	}
	value, ok := preparedPatchPlans.Load(toolCallID)
	if !ok {
		return nil, false
	}
	plan, ok := value.(preparedPatchPlan)
	if !ok || plan.PatchText != patchText {
		return nil, false
	}
	return append([]patchFileChange(nil), plan.Changes...), true
}

func storePreparedExpectedHash(toolCallID string, path string, hash string) {
	if toolCallID == "" || path == "" {
		return
	}
	preparedExpectedHashes.Store(toolCallID+"\x00"+cleanPatchPath(path), hash)
}

func preparedExpectedHash(toolCallID string, path string) string {
	if toolCallID == "" || path == "" {
		return ""
	}
	value, ok := preparedExpectedHashes.Load(toolCallID + "\x00" + cleanPatchPath(path))
	if !ok {
		return ""
	}
	hash, ok := value.(string)
	if !ok {
		return ""
	}
	return hash
}
