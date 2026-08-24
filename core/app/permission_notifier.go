package app

import "sync"

type permissionNotifier struct {
	mu      sync.Mutex
	waiters map[string][]chan struct{}
}

func newPermissionNotifier() *permissionNotifier {
	return &permissionNotifier{waiters: map[string][]chan struct{}{}}
}

func (n *permissionNotifier) watch(requestID string) <-chan struct{} {
	ch := make(chan struct{}, 1)
	if requestID == "" {
		close(ch)
		return ch
	}
	n.mu.Lock()
	n.waiters[requestID] = append(n.waiters[requestID], ch)
	n.mu.Unlock()
	return ch
}

func (n *permissionNotifier) resolve(requestID string) {
	if requestID == "" {
		return
	}
	n.mu.Lock()
	waiters := n.waiters[requestID]
	delete(n.waiters, requestID)
	n.mu.Unlock()
	for _, ch := range waiters {
		close(ch)
	}
}

func (n *permissionNotifier) forget(requestID string, watched <-chan struct{}) {
	if requestID == "" || watched == nil {
		return
	}
	n.mu.Lock()
	waiters := n.waiters[requestID]
	for index, ch := range waiters {
		if ch == watched {
			waiters = append(waiters[:index], waiters[index+1:]...)
			break
		}
	}
	if len(waiters) == 0 {
		delete(n.waiters, requestID)
	} else {
		n.waiters[requestID] = waiters
	}
	n.mu.Unlock()
}
