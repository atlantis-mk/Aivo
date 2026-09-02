import { useCallback, useEffect, useRef, useState } from "react";
import { toast } from "sonner";

import type { domain } from "../../../bridge/go/models";
import { BrowserOpenURL, EventsOn } from "../../../bridge/runtime/runtime";
import { normalizeProviderAuthUpdatedPayload } from "@/features/providers/provider-events";
import type { ProviderAuthMode } from "@/features/providers/provider-types";
import { hasAppBridge } from "@/lib/app-config";
import { startProviderAuth } from "@/services/aivo";
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

  useEffect(() => {
    if (!hasAppBridge()) return;
    return EventsOn("provider_auth.updated", (...payloads: unknown[]) => {
      const status =
        normalizeProviderAuthUpdatedPayload<domain.ProviderAuthStatus>(payloads);
      if (!status || status.providerId !== "openai") return;
      void window.aivo?.focusWindow?.();
      setOauthStarted(true);
      setOauthStatus(status);
      if (status.status === "failed") {
        setLocalError(status.error || "OpenAI 授权失败。");
        return;
      }
      if (status.status !== "success") return;
      markOpenAIAuthorized();
    });
  }, [markOpenAIAuthorized, setLocalError]);

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
    if (!hasAppBridge()) {
      setLocalError("OpenAI OAuth 需要 Aivo 桌面后端支持。");
      return false;
    }
    if (!oauthStarted) {
      const start = await startProviderAuth({
        providerId: "openai",
        method: authMode,
      });
      setOauthStartResult(start);
      setOauthStatus({
        providerId: start.providerId,
        method: start.method,
        status: start.status,
        instructions: start.instructions,
        userCode: start.userCode,
      } as domain.ProviderAuthStatus);
      setOauthStarted(true);
      if (start.url) await openExternalURL(start.url);
      return false;
    }
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

async function openExternalURL(url: string) {
  await BrowserOpenURL(url);
}
