import { useCallback, useEffect, useRef, useState } from "react";
import { toast } from "sonner";

import type { domain } from "@/types/codex-domain";
import { openExternalURL, onDesktopEvent } from "@/lib/desktop-events";
import { normalizeProviderAuthUpdatedPayload } from "@/features/providers/provider-events";
import type { ProviderAuthMode } from "@/features/providers/provider-types";
import type { ProviderPickOption } from "@/features/projects/project-provider-picker-model";

export function useOpenAIProviderAuthState({
  authMode,
  selectedProvider,
  setLocalError,
}: {
  authMode: ProviderAuthMode;
  selectedProvider: ProviderPickOption | null;
  setLocalError: (error: string) => void;
}) {
  const [oauthStarted, setOauthStarted] = useState(false);
  const [oauthStartResult, setOauthStartResult] =
    useState<domain.ProviderAuthStartResult | null>(null);
  const [oauthStatus, setOauthStatus] =
    useState<domain.ProviderAuthStatus | null>(null);
  const [authSuccessMessage, setAuthSuccessMessage] = useState("");
  const authSuccessNotifiedRef = useRef(false);
  const oauthReady = oauthStatus?.status === "success";

  const markOpenAIAuthorized = useCallback(() => {
    setAuthSuccessMessage("OpenAI 授权已完成");
    setOauthStatus(
      (current) =>
        ({
          providerId: "openai",
          method: current?.method || authMode,
          status: "success",
          accountId: current?.accountId,
          instructions: current?.instructions,
          userCode: current?.userCode,
        }) as domain.ProviderAuthStatus,
    );
    if (!authSuccessNotifiedRef.current) {
      authSuccessNotifiedRef.current = true;
      toast.success("OpenAI 授权已完成");
    }
  }, [authMode]);

  function resetOpenAIAuthState() {
    setOauthStarted(false);
    setOauthStartResult(null);
    setOauthStatus(null);
    setAuthSuccessMessage("");
    authSuccessNotifiedRef.current = false;
  }

  async function startOrCheckOpenAIOAuth() {
    if (!selectedProvider || selectedProvider.id !== "openai") return false;
    if (oauthStatus?.status === "success") return true;
    setLocalError("请在设置页通过 Codex 桌面运行时完成 OpenAI 登录。");
    return false;
  }

  return {
    authSuccessMessage,
    oauthReady,
    oauthStarted,
    oauthStartResult,
    oauthStatus,
    resetOpenAIAuthState,
    startOrCheckOpenAIOAuth,
  };
}
