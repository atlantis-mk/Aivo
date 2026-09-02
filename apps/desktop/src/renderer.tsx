import { useEffect, useState } from "react";
import { createRoot } from "react-dom/client";
import "./styles.css";

const App = () => {
  const [status, setStatus] = useState<RuntimeStatus>({
    state: "stopped",
    detail: "Checking local runtime…",
  });
  const [pending, setPending] = useState(false);

  useEffect(() => {
    void window.aivoDesktop.runtime.getStatus().then(setStatus);
    return window.aivoDesktop.runtime.onStatus(setStatus);
  }, []);

  const startRuntime = async (): Promise<void> => {
    setPending(true);
    try {
      setStatus(await window.aivoDesktop.runtime.start());
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

  const action = status.state === "ready" ? stopRuntime : startRuntime;
  const label =
    status.state === "ready" ? "Stop runtime" : "Start local runtime";

  return (
    <main>
      <section className="hero">
        <p className="eyebrow">AIVO DESKTOP · DEVELOPMENT PREVIEW</p>
        <h1>Local agent workspace</h1>
        <p className="lede">
          This desktop shell communicates only with a local Codex app-server
          process.
        </p>
      </section>

      <section className="runtime-card" aria-live="polite">
        <div>
          <p className="label">Runtime</p>
          <h2>
            <span className={`status-dot ${status.state}`} />
            {status.state}
          </h2>
          <p className="detail">{status.detail}</p>
        </div>
        <button
          type="button"
          onClick={() => void action()}
          disabled={pending || status.state === "starting"}
        >
          {pending ? "Working…" : label}
        </button>
      </section>

      <section className="next">
        <h2>Next milestone</h2>
        <p>
          Once the local runtime is ready, this window will create and display
          app-server threads.
        </p>
      </section>
    </main>
  );
};

createRoot(document.getElementById("root")!).render(<App />);
