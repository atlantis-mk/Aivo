//go:build darwin && cgo

package app

/*
#cgo darwin CFLAGS: -x objective-c -fobjc-arc -fblocks -mmacosx-version-min=10.13
#cgo darwin LDFLAGS: -framework Cocoa -framework AuthenticationServices
#include <stdlib.h>

int AivoStartASWebAuthenticationSession(const char *authURL, const char *callbackScheme, long long sessionID);
void AivoCancelASWebAuthenticationSession(long long sessionID);
*/
import "C"

import (
	"context"
	"sync"
	"sync/atomic"
	"unsafe"
)

var nativeWebAuthRegistry = struct {
	mu        sync.Mutex
	nextID    int64
	callbacks map[int64]func(callbackURL string, errorMessage string)
}{
	callbacks: map[int64]func(callbackURL string, errorMessage string){},
}

func openAINativeWebAuthAvailable() bool {
	return true
}

func startOpenAINativeWebAuthSession(manager *ProviderAuthManager, authURL string) (int64, bool) {
	sessionID := atomic.AddInt64(&nativeWebAuthRegistry.nextID, 1)
	nativeWebAuthRegistry.mu.Lock()
	nativeWebAuthRegistry.callbacks[sessionID] = func(callbackURL string, errorMessage string) {
		if errorMessage != "" {
			manager.fail("openai", errorMessage)
			return
		}
		if err := manager.completeOpenAICallbackURL(context.Background(), callbackURL); err != nil {
			manager.fail("openai", err.Error())
		}
	}
	nativeWebAuthRegistry.mu.Unlock()

	cAuthURL := C.CString(authURL)
	cCallbackScheme := C.CString("http")
	started := C.AivoStartASWebAuthenticationSession(cAuthURL, cCallbackScheme, C.longlong(sessionID)) == 1
	C.free(unsafe.Pointer(cAuthURL))
	C.free(unsafe.Pointer(cCallbackScheme))
	if !started {
		nativeWebAuthRegistry.mu.Lock()
		delete(nativeWebAuthRegistry.callbacks, sessionID)
		nativeWebAuthRegistry.mu.Unlock()
	}
	return sessionID, started
}

func cancelOpenAINativeWebAuthSession(sessionID int64) {
	nativeWebAuthRegistry.mu.Lock()
	delete(nativeWebAuthRegistry.callbacks, sessionID)
	nativeWebAuthRegistry.mu.Unlock()
	C.AivoCancelASWebAuthenticationSession(C.longlong(sessionID))
}

//export aivoASWebAuthComplete
func aivoASWebAuthComplete(sessionID C.longlong, callbackURL *C.char, errorMessage *C.char) {
	id := int64(sessionID)
	var callback string
	var message string
	if callbackURL != nil {
		callback = C.GoString(callbackURL)
		C.free(unsafe.Pointer(callbackURL))
	}
	if errorMessage != nil {
		message = C.GoString(errorMessage)
		C.free(unsafe.Pointer(errorMessage))
	}

	nativeWebAuthRegistry.mu.Lock()
	handler := nativeWebAuthRegistry.callbacks[id]
	delete(nativeWebAuthRegistry.callbacks, id)
	nativeWebAuthRegistry.mu.Unlock()
	if handler != nil {
		handler(callback, message)
	}
}
