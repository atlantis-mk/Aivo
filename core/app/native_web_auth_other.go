//go:build !darwin || !cgo

package app

func openAINativeWebAuthAvailable() bool {
	return false
}

func startOpenAINativeWebAuthSession(manager *ProviderAuthManager, authURL string) (int64, bool) {
	return 0, false
}

func cancelOpenAINativeWebAuthSession(sessionID int64) {
}
