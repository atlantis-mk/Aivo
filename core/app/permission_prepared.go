package app

import "sync"

var preparedExpectedHashes sync.Map

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
