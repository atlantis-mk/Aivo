package app

import (
	"context"
	"errors"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"
)

func (m *ProviderAuthManager) ensureOAuthServer() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.server != nil {
		return nil
	}
	mux := http.NewServeMux()
	mux.HandleFunc(openAIOAuthPath, m.handleOpenAICallback)
	mux.HandleFunc("/cancel", func(w http.ResponseWriter, r *http.Request) {
		m.cancel("openai")
		_, _ = io.WriteString(w, "Login cancelled")
	})
	server := &http.Server{Addr: fmt.Sprintf("127.0.0.1:%d", openAIOAuthPort), Handler: mux}
	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		return err
	}
	m.server = server
	go func() {
		_ = server.Serve(listener)
	}()
	return nil
}

func (m *ProviderAuthManager) handleOpenAICallback(w http.ResponseWriter, r *http.Request) {
	if err := m.completeOpenAICallbackQuery(r.Context(), r.URL.Query()); err != nil {
		writeOAuthHTML(w, false, err.Error())
		return
	}
	writeOAuthHTML(w, true, "")
}

func (m *ProviderAuthManager) completeOpenAICallbackURL(ctx context.Context, rawCallbackURL string) error {
	callbackURL, err := url.Parse(rawCallbackURL)
	if err != nil {
		m.fail("openai", "invalid OAuth callback URL")
		return errors.New("Invalid OAuth callback URL.")
	}
	return m.completeOpenAICallbackQuery(ctx, callbackURL.Query())
}

func (m *ProviderAuthManager) completeOpenAICallbackQuery(ctx context.Context, query url.Values) error {
	if errText := query.Get("error"); errText != "" {
		message := firstNonEmpty(query.Get("error_description"), errText)
		m.fail("openai", message)
		return errors.New(message)
	}
	code := query.Get("code")
	state := query.Get("state")
	m.mu.Lock()
	flow := m.flows["openai"]
	m.mu.Unlock()
	if flow == nil || flow.State != state || code == "" || time.Now().After(flow.ExpiresAt) {
		m.fail("openai", "invalid or expired OAuth callback")
		return errors.New("Invalid or expired OAuth callback.")
	}
	tokens, err := exchangeOpenAICode(ctx, code, openAIBrowserRedirectURI(), flow.Verifier)
	if err != nil {
		m.fail("openai", err.Error())
		return err
	}
	if err := m.saveOpenAITokens(ctx, tokens, "oauth-browser"); err != nil {
		m.fail("openai", err.Error())
		return err
	}
	nativeAuthID := flow.NativeAuthID
	m.succeed("openai", extractOpenAIAccountID(tokens))
	if nativeAuthID != 0 {
		cancelOpenAINativeWebAuthSession(nativeAuthID)
	}
	if m.service.onAuthSuccess != nil {
		go func() {
			time.Sleep(800 * time.Millisecond)
			m.service.onAuthSuccess()
		}()
	}
	return nil
}

func openAIBrowserRedirectURI() string {
	return fmt.Sprintf("http://localhost:%d%s", openAIOAuthPort, openAIOAuthPath)
}

func writeOAuthHTML(w http.ResponseWriter, ok bool, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	title := "Authorization Successful"
	body := "You can close this window and return to Aivo."
	script := `<script>
setTimeout(function() {
  window.open("", "_self");
  window.close();
  setTimeout(function() {
    var body = document.getElementById("oauth-message");
    if (body) body.textContent = "Authorization is complete. Your browser may block automatic tab closing, so you can close this tab manually.";
  }, 500);
}, 2000);
</script>`
	if !ok {
		title = "Authorization Failed"
		body = message
		script = ""
	}
	_, _ = fmt.Fprintf(w, `<!doctype html><html><head><title>Aivo - %s</title><style>body{font-family:system-ui,-apple-system,sans-serif;display:flex;justify-content:center;align-items:center;height:100vh;margin:0;background:#131010;color:#f1ecec}.container{text-align:center;padding:2rem}p{color:#b7b1b1}</style></head><body><div class="container"><h1>%s</h1><p id="oauth-message">%s</p></div>%s</body></html>`, html.EscapeString(title), html.EscapeString(title), html.EscapeString(body), script)
}
