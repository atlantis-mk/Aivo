import { useEffect, useState } from "react";
import { createRoot } from "react-dom/client";
import "./styles.css";

const defaultAccount: CodexAccount = {
  authMode: null,
  email: null,
  planType: null,
};

const App = () => {
  const [status, setStatus] = useState<RuntimeStatus>({
    state: "stopped",
    detail: "Checking local runtime…",
  });
  const [account, setAccount] = useState<CodexAccount>(defaultAccount);
  const [loginId, setLoginId] = useState<string | null>(null);
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);

  useEffect(() => {
    void window.aivoDesktop.runtime.getStatus().then(setStatus);
    const removeRuntimeListener = window.aivoDesktop.runtime.onStatus(setStatus);
    const removeAccountListener = window.aivoDesktop.codex.onAccount((next) => {
      setAccount(next);
      setLoginId(null);
      setMessage(next.authMode === "chatgpt" ? "Codex 账号已连接。" : "");
    });
    const removeLoginListener = window.aivoDesktop.codex.onLoginCompleted(
      (completion) => {
        if (!completion.success) {
          setLoginId(null);
          setMessage(completion.error ?? "登录未完成。");
          return;
        }
        setMessage("登录已完成，正在获取账号信息…");
      },
    );

    return () => {
      removeRuntimeListener();
      removeAccountListener();
      removeLoginListener();
    };
  }, []);

  const startRuntime = async (): Promise<void> => {
    setPending(true);
    try {
      setStatus(await window.aivoDesktop.runtime.start());
      setAccount(await window.aivoDesktop.codex.getAccount());
    } finally {
      setPending(false);
    }
  };

  const connectCodex = async (): Promise<void> => {
    setPending(true);
    setMessage("");
    try {
      const login = await window.aivoDesktop.codex.login();
      setLoginId(login.loginId);
      setStatus(await window.aivoDesktop.runtime.getStatus());
      setMessage("浏览器已打开，请在其中完成登录以连接 Codex。");
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "无法开始登录。");
    } finally {
      setPending(false);
    }
  };

  const cancelLogin = async (): Promise<void> => {
    if (!loginId) return;
    setPending(true);
    try {
      await window.aivoDesktop.codex.cancelLogin(loginId);
      setLoginId(null);
      setMessage("已取消登录。");
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "无法取消登录。");
    } finally {
      setPending(false);
    }
  };

  const logout = async (): Promise<void> => {
    setPending(true);
    setMessage("");
    try {
      await window.aivoDesktop.codex.logout();
      setAccount(defaultAccount);
      setMessage("已退出 Codex 登录。");
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "无法退出登录。");
    } finally {
      setPending(false);
    }
  };

  const stopRuntime = async (): Promise<void> => {
    setPending(true);
    try {
      setStatus(await window.aivoDesktop.runtime.stop());
    } finally {
      setPending(false);
    }
  };

  const connected = account.authMode === "chatgpt";

  return (
    <div className="app-shell">
      <aside className="sidebar">
        <div className="brand"><span>A</span> Aivo</div>
        <nav aria-label="Primary navigation">
          <button className="nav-item" type="button">工作区</button>
          <button className="nav-item active" type="button">服务提供商</button>
          <button className="nav-item" type="button" disabled>设置</button>
        </nav>
        <div className="runtime-summary">
          <span className={`status-dot ${status.state}`} />
          <div>
            <strong>本地运行时</strong>
            <small>{status.state === "ready" ? "已就绪" : status.state}</small>
          </div>
        </div>
      </aside>

      <main>
        <header className="page-heading">
          <p className="eyebrow">初始化设置</p>
          <h1>连接服务提供商</h1>
          <p>先连接 Codex 账号；其他提供商将在后续版本加入。</p>
        </header>

        <section className="setup-grid">
          <article className="provider-card">
            <div className="provider-heading">
              <div className="provider-icon">◒</div>
              <div>
                <h2>Codex</h2>
                <p>使用 ChatGPT 账号登录</p>
              </div>
              <span className={connected ? "connection-state connected" : "connection-state"}>
                {connected ? "已连接" : loginId ? "等待登录" : "未连接"}
              </span>
            </div>

            {connected ? (
              <div className="account-details">
                <div><span>账号</span><strong>{account.email ?? "ChatGPT 账号"}</strong></div>
                <div><span>订阅</span><strong>{account.planType ?? "可用"}</strong></div>
                <button className="button secondary" disabled={pending} onClick={() => void logout()} type="button">
                  退出登录
                </button>
              </div>
            ) : (
              <div className="connection-panel">
                <p>Codex 托管安全的浏览器登录，并在本地 Codex 运行时中保管凭据。</p>
                {loginId ? (
                  <button className="button secondary" disabled={pending} onClick={() => void cancelLogin()} type="button">
                    取消登录
                  </button>
                ) : (
                  <button className="button" disabled={pending} onClick={() => void connectCodex()} type="button">
                    {pending ? "正在打开浏览器…" : "使用 ChatGPT 登录"}
                  </button>
                )}
              </div>
            )}
            {message ? <p className="feedback" role="status">{message}</p> : null}
          </article>

          <article className="runtime-card" aria-live="polite">
            <p className="label">本地运行时</p>
            <h2><span className={`status-dot ${status.state}`} />{status.state}</h2>
            <p>{status.detail}</p>
            <button
              className="button secondary"
              type="button"
              onClick={() => void (status.state === "ready" ? stopRuntime() : startRuntime())}
              disabled={pending || status.state === "starting"}
            >
              {status.state === "ready" ? "停止运行时" : "启动运行时"}
            </button>
          </article>
        </section>
      </main>
    </div>
  );
};

createRoot(document.getElementById("root")!).render(<App />);
