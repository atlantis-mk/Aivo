import {
  type ComponentProps,
  useCallback,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { Link, useNavigate, useRouterState } from "@tanstack/react-router";
import { toast } from "sonner";
import { BrowserOpenURL, EventsOn } from "../../../bridge/runtime/runtime";
import {
  Archive,
  ArrowDown,
  ArrowLeft,
  ArrowRight,
  ArrowUp,
  Check,
  ChevronDown,
  Circle,
  Bot,
  CornerDownLeft,
  Copy,
  Ellipsis,
  File,
  FileText,
  FolderOpen,
  Hand,
  Image,
  LayoutGrid,
  Maximize2,
  Mic,
  Minimize2,
  Pause,
  PanelBottom,
  PanelLeft,
  PanelRight,
  Pencil,
  Pin,
  Plug,
  Plus,
  Search,
  Settings,
  ShieldAlert,
  ShieldCheck,
  Smartphone,
  SquarePen,
  Wrench,
  X,
} from "lucide-react";

import {
  EnvironmentSummaryPanel,
} from "@/components/app-top-bar";
import { AnimatedTitle } from "@/components/animated-title";
import { SubmittedPromptContent } from "@/features/projects/conversation-timeline";
import {
  TerminalDockPanel,
  TerminalDockProvider,
  TerminalDockTrigger,
} from "@/features/projects/terminal/terminal-dock";
import { useTerminalDock } from "@/features/projects/terminal/terminal-dock-store";
import { TerminalPanelContent } from "@/features/projects/terminal/terminal-panel";
import {
  useProjectPreferencesStore,
  type ProjectPanelLayout,
} from "@/features/projects/project-preferences-store";
import {
  getTurnElapsedSeconds,
  sameToolCalls,
  type ConversationAssistantTextPart,
  type ConversationTurn,
  type ConversationUserAttachment,
} from "@/features/projects/conversation-timeline-model";
import {
  BUILTIN_BROWSER_TAB_ID,
  ToolActivitySidebar,
} from "@/features/projects/tool-activity-sidebar";
import {
  annotateToolActivityTabsWithFileStates,
  appendShellOutputToTabs,
  toolActivityTabsFromToolCall,
  toolActivityTabsFromToolCalls,
  upsertToolActivityTabs,
  type ShellOutputPayload,
  type ToolActivityFileTab,
  type ToolActivityTab,
} from "@/features/projects/tool-activity-model";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarInset,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarProvider,
  useSidebar,
} from "@/components/ui/sidebar";
import {
  Attachment,
  AttachmentAction,
  AttachmentActions,
  AttachmentContent,
  AttachmentDescription,
  AttachmentGroup,
  AttachmentMedia,
  AttachmentTitle,
} from "@/components/ui/attachment";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Badge } from "@/components/ui/badge";
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandSeparator,
} from "@/components/ui/command";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  HoverCard,
  HoverCardContent,
  HoverCardTrigger,
} from "@/components/ui/hover-card";
import { Input } from "@/components/ui/input";
import {
  Item,
  ItemContent,
  ItemDescription,
  ItemGroup,
  ItemMedia,
  ItemTitle,
} from "@/components/ui/item";
import { Kbd } from "@/components/ui/kbd";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import { Spinner } from "@/components/ui/spinner";
import { Switch } from "@/components/ui/switch";
import { cn } from "@/lib/utils";
import { hasAppBridge, useAppConfig } from "@/lib/app-config";
import type {
  CatalogState,
  ModelInfo,
  ProviderInfo,
} from "@/lib/provider-catalog";
import {
  ProviderConnectionDialogs,
  type CustomProviderForm,
  type CustomProviderProtocol,
  type CustomProviderRow,
  type ProviderAuthMode,
  type ProviderChoice,
  type ProviderDialogStep,
} from "@/features/setup/provider-connection-dialogs";
import { PluginMcpSettingsContent } from "@/features/projects/plugin-mcp-settings-dialog";
import { connectPreviewProvider } from "@/lib/preview-state";
import {
  archiveSession,
  applySessionTurnFileState,
  approvePermissionRequest,
  cancelAgentRun,
  cancelSessionTurn,
  connectProvider,
  createSession,
  deleteSessionEvent,
  denyPermissionRequest,
  getCodingContext,
  getSessionActiveTools,
  getSessionTurnDiff,
  getPermissionMode,
  listAgentModes,
  listAgentRuns,
  listPlugins,
  listRecentProjects,
  listPermissionRequests,
  listQuestionRequests,
  listSessionEvents,
  listSessionToolCalls,
  listSessionTurns,
  listToolCatalog,
  listSessions,
  listTodoItems,
  refreshProviderModels,
  rejectQuestionRequest,
  replyQuestionRequest,
  retrySessionTurn,
  selectProjectDirectory,
  setProjectSidebarHidden,
  setSessionActiveTools,
  setSessionAgentMode,
  setPermissionMode,
  submitSessionMessage,
  startProviderAuth,
  updateSessionEvent,
  updateModelPreferences,
  upsertProject,
  type AgentModeDefinition,
  type AgentModeId,
  type AgentRun,
  type PermissionMode,
  type PermissionRequest,
  type PluginListItem,
  type QuestionRequest,
  type TodoItem,
  type ToolCatalogEntry,
} from "@/services/aivo";
import type { domain } from "../../../bridge/go/models";

const CONVERSATION_OPEN_ANIMATION_MS = 520;
const OPEN_CONVERSATION_FROM_EMPTY_DELAY = CONVERSATION_OPEN_ANIMATION_MS + 60;
const EMPTY_COMPOSER_VERTICAL_OFFSET = 8;
const MARKDOWN_CONTENT_RESIZE_EVENT = "aivo-markdown-content-resize";
const SCROLL_BOTTOM_SENTINEL = 9_999_999;
const FORCE_BOTTOM_FRAME_COUNT = 18;
const SCROLL_BOTTOM_ANIMATION_MS = 220;
const SHOW_SCROLL_TO_BOTTOM_DISTANCE = 96;
const PROJECT_LEFT_SIDEBAR_MIN_WIDTH = 210;
const PROJECT_RIGHT_SIDEBAR_MIN_WIDTH = 240;
const PROJECT_BOTTOM_PANEL_MIN_HEIGHT = 180;
const PROJECT_MAIN_MIN_WIDTH = 360;
const PROJECT_UPPER_MIN_HEIGHT = 240;
const PROJECT_PANEL_TRANSITION_MS = 200;
const BROWSER_REVEAL_AFTER_PANEL_MS = PROJECT_PANEL_TRANSITION_MS + 40;
const SHOULD_MOUNT_TOOL_ACTIVITY_SIDEBAR = true;
const SHOULD_AUTO_OPEN_TOOL_ACTIVITY_SIDEBAR = true;

type PermissionActionState =
  "idle" | "approving" | "denying" | "approved" | "denied";

type ToolActivitySessionState = {
  activeTabId: string;
  browserInitialUrls: Record<string, string>;
  browserTabIds: string[];
  closedItemIds: string[];
  isOpen: boolean;
  tabs: ToolActivityTab[];
};

type ProjectConversationGroup = {
  conversations: domain.Session[];
  project: domain.AssistantProject;
  projectPath: string;
};

type PendingAssistantDelta = {
  sessionId: string;
  text: string;
  turnId?: string;
};

type ConversationTimelineHandlerRefs = {
  onDeleteAssistantMessage: (turn: ConversationTurn) => void;
  onDeleteTurn: (turn: ConversationTurn) => void;
  onEditUserMessage: (turn: ConversationTurn) => void;
  onOpenSession: (sessionId: string) => void;
  onRetryTurn: (turn: ConversationTurn) => void;
};

function clampNumber(value: number, min: number, max: number) {
  if (!Number.isFinite(value)) return min;
  return Math.min(Math.max(value, min), Math.max(min, max));
}

const writePermissionToolNames = new Set([
  "apply_patch",
  "write_file",
  "edit_file",
]);

type ModelOption = ModelInfo & {
  providerName: string;
};

type ComposerAttachment = {
  id: string;
  name: string;
  mimeType: string;
  size: number;
  kind: "image" | "file";
  data: string;
  previewUrl?: string;
};

type ProviderProtocol = CustomProviderProtocol;
type ProviderPickOption = ProviderChoice & {
  type: string;
  baseUrl?: string;
  defaultModelId?: string;
  models: ModelInfo[];
};

const providerIconModules = import.meta.glob<string>(
  "@/assets/icons/provider/*.svg",
  {
    eager: true,
    import: "default",
    query: "?url",
  },
);

const providerProtocolDefaults: Record<string, ProviderProtocol> = {
  openai: "openai",
  "claude-code": "anthropic",
  anthropic: "anthropic",
  google: "google",
  gemini: "google",
  openrouter: "openrouter",
  "kimi-for-coding": "anthropic",
  minimax: "anthropic",
  "minimax-cn": "anthropic",
  "minimax-coding-plan": "anthropic",
  "minimax-cn-coding-plan": "anthropic",
  "perplexity-agent": "openai",
  vivgrid: "openai",
};

const providerBaseURLDefaults: Record<string, string> = {
  "302ai": "https://api.302.ai/v1",
  abacus: "https://routellm.abacus.ai/v1",
  aihubmix: "https://aihubmix.com/v1",
  alibaba: "https://dashscope-intl.aliyuncs.com/compatible-mode/v1",
  "alibaba-cn": "https://dashscope.aliyuncs.com/compatible-mode/v1",
  "alibaba-coding-plan": "https://coding-intl.dashscope.aliyuncs.com/v1",
  "alibaba-coding-plan-cn": "https://coding.dashscope.aliyuncs.com/v1",
  anthropic: "https://api.anthropic.com/v1",
  baseten: "https://inference.baseten.co/v1",
  bailing: "https://api.tbox.cn/api/llm/v1/chat/completions",
  berget: "https://api.berget.ai/v1",
  cerebras: "https://api.cerebras.ai/v1",
  chutes: "https://llm.chutes.ai/v1",
  clarifai: "https://api.clarifai.com/v2/ext/openai/v1",
  "cloudferro-sherlock": "https://api-sherlock.cloudferro.com/openai/v1/",
  "cloudflare-workers-ai":
    "https://api.cloudflare.com/client/v4/accounts/${CLOUDFLARE_ACCOUNT_ID}/ai/v1",
  cortecs: "https://api.cortecs.ai/v1",
  deepinfra: "https://api.deepinfra.com/v1/openai",
  deepseek: "https://api.deepseek.com",
  digitalocean: "https://inference.do-ai.run/v1",
  dinference: "https://api.dinference.com/v1",
  drun: "https://chat.d.run/v1",
  evroc: "https://models.think.evroc.com/v1",
  fastrouter: "https://go.fastrouter.ai/api/v1",
  "fireworks-ai": "https://api.fireworks.ai/inference/v1/",
  friendli: "https://api.friendli.ai/serverless/v1",
  "github-copilot": "https://api.githubcopilot.com",
  "github-models": "https://models.github.ai/inference",
  google: "https://generativelanguage.googleapis.com/v1beta",
  groq: "https://api.groq.com/openai/v1",
  helicone: "https://ai-gateway.helicone.ai/v1",
  huggingface: "https://router.huggingface.co/v1",
  iflowcn: "https://apis.iflow.cn/v1",
  inception: "https://api.inceptionlabs.ai/v1/",
  inference: "https://inference.net/v1",
  "io-net": "https://api.intelligence.io.solutions/api/v1",
  jiekou: "https://api.jiekou.ai/openai",
  kilo: "https://api.kilo.ai/api/gateway",
  "kimi-for-coding": "https://api.kimi.com/coding/v1",
  "kuae-cloud-coding-plan": "https://coding-plan-endpoint.kuaecloud.net/v1",
  llama: "https://api.llama.com/compat/v1/",
  lmstudio: "http://127.0.0.1:1234/v1",
  lucidquery: "https://lucidquery.com/api/v1",
  meganova: "https://api.meganova.ai/v1",
  minimax: "https://api.minimax.io/anthropic/v1",
  "minimax-cn": "https://api.minimaxi.com/anthropic/v1",
  "minimax-coding-plan": "https://api.minimax.io/anthropic/v1",
  "minimax-cn-coding-plan": "https://api.minimaxi.com/anthropic/v1",
  modelscope: "https://api-inference.modelscope.cn/v1",
  moark: "https://moark.com/v1",
  moonshotai: "https://api.moonshot.ai/v1",
  "moonshotai-cn": "https://api.moonshot.cn/v1",
  morph: "https://api.morphllm.com/v1",
  "nano-gpt": "https://nano-gpt.com/api/v1",
  nebius: "https://api.tokenfactory.nebius.com/v1",
  "novita-ai": "https://api.novita.ai/openai",
  nvidia: "https://integrate.api.nvidia.com/v1",
  "ollama-cloud": "https://ollama.com/v1",
  opencode: "https://opencode.ai/zen/v1",
  "opencode-go": "https://opencode.ai/zen/go/v1",
  openrouter: "https://openrouter.ai/api/v1",
  ovhcloud: "https://oai.endpoints.kepler.ai.cloud.ovh.net/v1",
  perplexity: "https://api.perplexity.ai",
  "perplexity-agent": "https://api.perplexity.ai/v1",
  poe: "https://api.poe.com/v1",
  "privatemode-ai": "http://localhost:8080/v1",
  "qihang-ai": "https://api.qhaigc.net/v1",
  "qiniu-ai": "https://api.qnaigc.com/v1",
  requesty: "https://router.requesty.ai/v1",
  scaleway: "https://api.scaleway.ai/v1",
  siliconflow: "https://api.siliconflow.com/v1",
  "siliconflow-cn": "https://api.siliconflow.cn/v1",
  stackit: "https://api.openai-compat.model-serving.eu01.onstackit.cloud/v1",
  stepfun: "https://api.stepfun.com/v1",
  submodel: "https://llm.submodel.ai/v1",
  synthetic: "https://api.synthetic.new/openai/v1",
  "tencent-coding-plan": "https://api.lkeap.cloud.tencent.com/coding/v3",
  togetherai: "https://api.together.xyz/v1",
  upstage: "https://api.upstage.ai/v1/solar",
  venice: "https://api.venice.ai/api/v1",
  vivgrid: "https://api.vivgrid.com/v1",
  vultr: "https://api.vultrinference.com/v1",
  wandb: "https://api.inference.wandb.ai/v1",
  xai: "https://api.x.ai/v1",
  xiaomi: "https://api.xiaomimimo.com/v1",
  zai: "https://api.z.ai/api/paas/v4",
  "zai-coding-plan": "https://api.z.ai/api/coding/paas/v4",
  zenmux: "https://zenmux.ai/api/v1",
  zhipuai: "https://open.bigmodel.cn/api/paas/v4",
  "zhipuai-coding-plan": "https://open.bigmodel.cn/api/coding/paas/v4",
};

export function ProjectSelectionScreen() {
  const [prompt, setPrompt] = useState("");
  const [turns, setTurns] = useState<ConversationTurn[]>([]);
  const [sessions, setSessions] = useState<domain.Session[]>([]);
  const [pendingPermissionRequests, setPendingPermissionRequests] = useState<
    PermissionRequest[]
  >([]);
  const [pendingQuestionRequests, setPendingQuestionRequests] = useState<
    QuestionRequest[]
  >([]);
  const [
    pendingPermissionCountsBySessionId,
    setPendingPermissionCountsBySessionId,
  ] = useState<Record<string, number>>({});
  const [runningConversationIds, setRunningConversationIds] = useState<
    string[]
  >([]);
  const pinnedConversationIds = useProjectPreferencesStore(
    (state) => state.pinnedConversationIds,
  );
  const setPinnedConversationIds = useProjectPreferencesStore(
    (state) => state.setPinnedConversationIds,
  );
  const archivedConversationIds = useProjectPreferencesStore(
    (state) => state.archivedConversationIds,
  );
  const setArchivedConversationIds = useProjectPreferencesStore(
    (state) => state.setArchivedConversationIds,
  );
  const defaultActiveToolNames = useProjectPreferencesStore(
    (state) => state.defaultActiveToolNames,
  );
  const setDefaultActiveToolNames = useProjectPreferencesStore(
    (state) => state.setDefaultActiveToolNames,
  );
  const hiddenTodoPlanKeys = useProjectPreferencesStore(
    (state) => state.hiddenTodoPlanKeys,
  );
  const setHiddenTodoPlanKeyForSession = useProjectPreferencesStore(
    (state) => state.setHiddenTodoPlanKey,
  );
  const [activeSessionId, setActiveSessionId] = useState("");
  const [toolActivationDialogOpen, setToolActivationDialogOpen] = useState(false);
  const [composerHeight, setComposerHeight] = useState(116);
  const [composerExtraHeight, setComposerExtraHeight] = useState(0);
  const [isOpeningConversationFromEmpty, setOpeningConversationFromEmpty] =
    useState(false);
  const [isRevealingHistoryConversation, setRevealingHistoryConversation] =
    useState(false);
  const [isPinnedSummaryOpen, setPinnedSummaryOpen] = useState(false);
  const [isRightSidebarOpen, setRightSidebarOpen] = useState(false);
  const [isBrowserRevealReady, setBrowserRevealReady] = useState(false);
  const [builtinBrowserInitialUrls, setBuiltinBrowserInitialUrls] = useState<
    Record<string, string>
  >({});
  const [builtinBrowserReadyTokens, setBuiltinBrowserReadyTokens] = useState<
    Record<string, number>
  >({});
  const [builtinBrowserTabIds, setBuiltinBrowserTabIds] = useState<string[]>([]);
  const [toolActivityTabs, setToolActivityTabs] = useState<ToolActivityTab[]>(
    [],
  );
  const [activeToolActivityTabId, setActiveToolActivityTabId] = useState("");
  const [canDockPinnedSummary, setCanDockPinnedSummary] = useState(false);
  const [shouldShiftPinnedSummaryLayout, setShouldShiftPinnedSummaryLayout] =
    useState(false);
  const [codingWorkspaceRoot, setCodingWorkspaceRoot] = useState("");
  const [recentProjects, setRecentProjects] = useState<
    domain.AssistantProject[]
  >([]);
  const [selectedProjectPath, setSelectedProjectPath] = useState("");
  const [selectedProviderId, setSelectedProviderId] = useState("");
  const [selectedModelId, setSelectedModelId] = useState("");
  const [reasoningEffort, setReasoningEffort] = useState("medium");
  const [serviceTier, setServiceTier] = useState("default");
  const [permissionMode, setLocalPermissionMode] =
    useState<PermissionMode>("request_approval");
  const [agentModes, setAgentModes] = useState<AgentModeDefinition[]>([]);
  const [agentMode, setAgentMode] = useState<AgentModeId>("code");
  const [agentRuns, setAgentRuns] = useState<AgentRun[]>([]);
  const [todoItems, setTodoItems] = useState<TodoItem[]>([]);
  const [visibleTodoPlanItems, setVisibleTodoPlanItems] = useState<TodoItem[]>(
    [],
  );
  const hiddenTodoPlanKey = activeSessionId
    ? hiddenTodoPlanKeys[activeSessionId] ?? ""
    : "";
  const [showScrollToBottomButton, setShowScrollToBottomButton] =
    useState(false);
  const [composerAttachments, setComposerAttachments] = useState<
    ComposerAttachment[]
  >([]);
  const [isComposerDropActive, setComposerDropActive] = useState(false);
  const { catalog, config } = useAppConfig();
  const navigate = useNavigate();
  const pathname = useRouterState({ select: (state) => state.location.pathname });
  const mainRef = useRef<HTMLDivElement>(null);
  const messagesScrollRootRef = useRef<HTMLDivElement>(null);
  const messagesViewportRef = useRef<HTMLDivElement>(null);
  const messagesContentRef = useRef<HTMLDivElement>(null);
  const stickToBottomRef = useRef(true);
  const previousTurnCountRef = useRef(0);
  const activeSessionIdRef = useRef("");
  const pendingStopRequestedRef = useRef(false);
  const pendingAssistantDeltaRef = useRef<PendingAssistantDelta | null>(null);
  const assistantDeltaFrameRef = useRef(0);
  const composerDropDepthRef = useRef(0);
  const conversationTimelineHandlersRef =
    useRef<ConversationTimelineHandlerRefs>({
      onDeleteAssistantMessage: () => undefined,
      onDeleteTurn: () => undefined,
      onEditUserMessage: () => undefined,
      onOpenSession: () => undefined,
      onRetryTurn: () => undefined,
    });
  const toolActivitySessionStatesRef = useRef(
    new Map<string, ToolActivitySessionState>(),
  );
  const toolActivityTabsRef = useRef<ToolActivityTab[]>([]);
  const builtinBrowserInitialUrlsRef = useRef<Record<string, string>>({});
  const pendingBuiltinBrowserReadyRef = useRef(
    new Map<string, Set<() => void>>(),
  );
  const builtinBrowserTabIdsRef = useRef<string[]>([]);
  const activeToolActivityTabIdRef = useRef("");
  const isRightSidebarOpenRef = useRef(false);
  const closedToolActivityItemIdsRef = useRef(new Set<string>());
  const sidebarConversationSelectionRef = useRef(0);
  const snapNextMessageScrollRef = useRef(false);
  const forceStickToBottomRef = useRef(false);
  const forceBottomFrameRef = useRef(0);
  const forceBottomRemainingFramesRef = useRef(0);
  const scrollAnimationFrameRef = useRef(0);
  const resizeScrollFrameRef = useRef(0);
  const composerFrameRef = useRef<HTMLDivElement>(null);
  const pendingComposerTransitionRectRef = useRef<DOMRect | null>(null);
  const composerTransitionFrameRef = useRef(0);
  const composerTransitionTimeoutRef = useRef(0);
  const hasTurns = turns.length > 0;
  const showConversationLayout = hasTurns || isOpeningConversationFromEmpty;
  const hasPendingTurn = turns.some(
    (turn) => !turn.responseCompletedAt && !turn.stopped,
  );
  const hasPendingPermissionRequest = pendingPermissionRequests.length > 0;
  const hasPendingQuestionRequest = pendingQuestionRequests.length > 0;
  const hasPendingInteractionRequest =
    hasPendingPermissionRequest || hasPendingQuestionRequest;
  const conversationTitle =
    sessions.find((session) => session.id === activeSessionId)?.title ||
    turns[0]?.prompt ||
    "";
  const activeSession = sessions.find(
    (session) => session.id === activeSessionId,
  );
  const activeParentSessionId = activeSession?.parentSessionId || "";
  const isSubagentSession = Boolean(activeParentSessionId);
  const activeSubagentRun = useMemo(
    () =>
      isSubagentSession
        ? agentRuns.find((run) => run.sessionId === activeSessionId)
        : undefined,
    [activeSessionId, agentRuns, isSubagentSession],
  );
  const activeRunningSubagentRun =
    activeSubagentRun?.status === "running" ? activeSubagentRun : undefined;
  const activeWorkspaceRoot = activeSession?.projectPath || codingWorkspaceRoot;
  const composerProjectPath = activeSessionId
    ? activeWorkspaceRoot
    : selectedProjectPath;
  const composerProject = useMemo(
    () =>
      recentProjects.find(
        (project) => project.rootPath === composerProjectPath,
      ) ?? null,
    [composerProjectPath, recentProjects],
  );
  const canUseTerminalPanel = !hasPendingInteractionRequest;
  const activeProjectPage = pathname.startsWith("/projects/plugins")
    ? "plugins"
    : "chat";
  const canToggleEnvironmentSummaryPanel =
    activeProjectPage === "chat" && Boolean(activeSessionId);
  const shouldShowEnvironmentSummaryPanel =
    canToggleEnvironmentSummaryPanel && isPinnedSummaryOpen;

  useEffect(() => {
    if (activeProjectPage === "chat" && activeSessionId) return;
    setPinnedSummaryOpen(false);
  }, [activeProjectPage, activeSessionId]);

  useEffect(() => {
    if (pathname === "/projects") {
      void navigate({ to: "/projects/chat", replace: true });
    }
  }, [navigate, pathname]);

  useEffect(() => {
    if (!hasAppBridge()) return;
    let cancelled = false;
    void listAgentModes(false)
      .then((modes) => {
        if (!cancelled) {
          setAgentModes(modes.filter((mode) => !mode.hidden));
        }
      })
      .catch(() => undefined);
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    const sessionMode =
      ((
        activeSession as
          (domain.Session & { agentMode?: AgentModeId }) | undefined
      )?.agentMode as AgentModeId | undefined) || "code";
    setAgentMode(sessionMode);
  }, [activeSession]);

  useEffect(() => {
    setVisibleTodoPlanItems([]);
  }, [activeSessionId]);

  const refreshAgentRuntimeState = useCallback(
    async (sessionId = activeSessionIdRef.current) => {
      if (!hasAppBridge() || !sessionId) {
        setAgentRuns([]);
        setTodoItems([]);
        setVisibleTodoPlanItems([]);
        return;
      }
      const projectPath = activeWorkspaceRoot || "";
      const [runs, todos] = await Promise.all([
        listAgentRuns(sessionId, 12).catch(() => [] as AgentRun[]),
        listTodoItems(sessionId, projectPath, 12).catch(() => [] as TodoItem[]),
      ]);
      setAgentRuns(runs);
      setTodoItems(todos);
    },
    [activeWorkspaceRoot],
  );

  useEffect(() => {
    if (!hasAppBridge() || !activeSessionId) {
      setAgentRuns([]);
      setTodoItems([]);
      setVisibleTodoPlanItems([]);
      return;
    }
    let cancelled = false;
    void refreshAgentRuntimeState(activeSessionId).then(() => {
      if (cancelled) return;
    });
    return () => {
      cancelled = true;
    };
  }, [activeSessionId, refreshAgentRuntimeState]);

  useEffect(() => {
    setVisibleTodoPlanItems((current) =>
      todoItems.length > 0 ? todoItems : current,
    );
  }, [todoItems]);

  const visibleTodoPlanKey = useMemo(
    () => getTodoPlanKey(visibleTodoPlanItems),
    [visibleTodoPlanItems],
  );
  const isVisibleTodoPlanComplete =
    visibleTodoPlanItems.length > 0 &&
    visibleTodoPlanItems.every((todo) => isTodoDone(todo.status));
  const shouldShowTodoFloatingStatus =
    visibleTodoPlanItems.length > 0 &&
    visibleTodoPlanKey !== hiddenTodoPlanKey;

  useEffect(() => {
    if (!hasAppBridge() || !activeSessionId) {
      setCodingWorkspaceRoot("");
      return;
    }
    if (activeSession?.projectPath) {
      setCodingWorkspaceRoot("");
      return;
    }
    let cancelled = false;
    void getCodingContext(activeSessionId)
      .then((context) => {
        if (!cancelled) {
          setCodingWorkspaceRoot(context?.projectPath || "");
        }
      })
      .catch(() => {
        if (!cancelled) setCodingWorkspaceRoot("");
      });
    return () => {
      cancelled = true;
    };
  }, [activeSession?.projectPath, activeSessionId]);

  const lastTurn = turns.at(-1);
  const lastTurnStateKey = lastTurn
    ? [
        turns.length,
        lastTurn.responseVisible ? "visible" : "hidden",
        lastTurn.stopped ? "stopped" : "running",
        lastTurn.responseText.length,
        lastTurn.preToolText.length,
        lastTurn.assistantPreambles
          ?.map((part) => `${part.id}:${part.text.length}`)
          .join("|") ?? "",
        lastTurn.toolCalls.length,
        lastTurn.toolCalls
          .map((toolCall) =>
            [
              toolCall.id,
              toolCall.status,
              toolCall.timeUpdated ?? "",
              toolCall.resultSummary?.length ?? 0,
              toolCall.error?.length ?? 0,
            ].join(":"),
          )
          .join("|"),
      ].join("-")
    : "empty";
  const modelProviders = useMemo(
    () =>
      getConnectedModelProviders(
        config,
        catalog?.providers ?? [],
        catalog?.connectedProviders ?? [],
      ),
    [catalog?.connectedProviders, catalog?.providers, config],
  );
  const activeProvider = useMemo(
    () => getActiveProvider(config, modelProviders, selectedProviderId),
    [config, modelProviders, selectedProviderId],
  );
  const modelOptions = useMemo(
    () => getModelOptions(activeProvider, catalog?.models ?? []),
    [activeProvider, catalog?.models],
  );
  const allModelOptions = useMemo(
    () => getAllModelOptions(modelProviders, catalog?.models ?? []),
    [catalog?.models, modelProviders],
  );
  const defaultModelId = getDefaultModelId(
    config,
    activeProvider,
    modelOptions,
  );
  const modelOptionsKey = modelOptions.map((model) => model.id).join("|");
  const activeModelId = normalizeModelId(
    activeProvider?.id,
    selectedModelId || defaultModelId,
  );
  const activeModelRef =
    activeProvider && activeModelId
      ? { providerId: activeProvider.id, modelId: activeModelId }
      : undefined;
  const visibleSessions = useMemo(
    () =>
      sessions.filter(
        (session) =>
          !session.parentSessionId &&
          !archivedConversationIds.includes(session.id),
      ),
    [archivedConversationIds, sessions],
  );
  const projectConversationGroups = useMemo(
    () =>
      buildProjectConversationGroups(
        visibleSessions,
        recentProjects,
        selectedProjectPath,
      ),
    [recentProjects, selectedProjectPath, visibleSessions],
  );
  const visibleConversations = useMemo(
    () =>
      visibleSessions.filter(
        (session) =>
          !isSessionGroupedUnderProject(
            session,
            projectConversationGroups,
          ),
      ),
    [projectConversationGroups, visibleSessions],
  );
  const handleComposerHeightChange = useCallback((height: number) => {
    const nextHeight = Math.round(height);
    setComposerHeight((currentHeight) =>
      currentHeight === nextHeight ? currentHeight : nextHeight,
    );
  }, []);
  const stopComposerTransition = useCallback(() => {
    if (composerTransitionFrameRef.current) {
      window.cancelAnimationFrame(composerTransitionFrameRef.current);
    }
    if (composerTransitionTimeoutRef.current) {
      window.clearTimeout(composerTransitionTimeoutRef.current);
    }
    composerTransitionFrameRef.current = 0;
    composerTransitionTimeoutRef.current = 0;
    const composerElement = composerFrameRef.current;
    if (!composerElement) return;
    composerElement.style.transform = "";
    composerElement.style.transition = "";
  }, []);
  const captureComposerTransitionStart = useCallback(() => {
    const composerElement = composerFrameRef.current;
    stopComposerTransition();
    pendingComposerTransitionRectRef.current =
      composerElement?.getBoundingClientRect() ?? null;
    if (composerElement) {
      composerElement.style.transition = "none";
    }
  }, [stopComposerTransition]);

  const flushPendingAssistantDelta = useCallback(() => {
    if (assistantDeltaFrameRef.current) {
      window.cancelAnimationFrame(assistantDeltaFrameRef.current);
      assistantDeltaFrameRef.current = 0;
    }

    const pending = pendingAssistantDeltaRef.current;
    pendingAssistantDeltaRef.current = null;
    if (!pending?.text) return;

    setTurns((currentTurns) => {
      const index = currentTurns.findLastIndex(
        (turn) => !turn.stopped && !turn.responseCompletedAt,
      );
      if (index < 0) return currentTurns;
      return currentTurns.map((turn, turnIndex) => {
        if (turnIndex !== index) return turn;
        return {
          ...turn,
          responseText: turn.responseText + pending.text,
          responseVisible: true,
          thinkingSeconds: getTurnElapsedSeconds(turn),
          turnId: turn.turnId || pending.turnId,
        };
      });
    });
  }, []);

  const cancelPendingAssistantDelta = useCallback(() => {
    if (assistantDeltaFrameRef.current) {
      window.cancelAnimationFrame(assistantDeltaFrameRef.current);
      assistantDeltaFrameRef.current = 0;
    }
    pendingAssistantDeltaRef.current = null;
  }, []);

  const enqueueAssistantDelta = useCallback(
    (payload: { delta: string; sessionId?: string; turnId?: string }) => {
      if (!payload.delta || !payload.sessionId) return;

      const current = pendingAssistantDeltaRef.current;
      if (
        current &&
        (current.sessionId !== payload.sessionId ||
          current.turnId !== payload.turnId)
      ) {
        flushPendingAssistantDelta();
      }

      pendingAssistantDeltaRef.current = {
        sessionId: payload.sessionId,
        text: `${pendingAssistantDeltaRef.current?.text ?? ""}${payload.delta}`,
        turnId: payload.turnId,
      };

      if (assistantDeltaFrameRef.current) return;
      assistantDeltaFrameRef.current =
        window.requestAnimationFrame(flushPendingAssistantDelta);
    },
    [flushPendingAssistantDelta],
  );

  useEffect(() => {
    flushPendingAssistantDelta();
    activeSessionIdRef.current = activeSessionId;
  }, [activeSessionId, flushPendingAssistantDelta]);

  useEffect(() => {
    toolActivityTabsRef.current = toolActivityTabs;
    builtinBrowserInitialUrlsRef.current = builtinBrowserInitialUrls;
    builtinBrowserTabIdsRef.current = builtinBrowserTabIds;
    activeToolActivityTabIdRef.current = activeToolActivityTabId;
    isRightSidebarOpenRef.current = isRightSidebarOpen;
    if (!activeSessionId) return;
    const tabs = toolActivityTabs.filter(
      (tab) => !toolActivityTabIsClosed(tab, closedToolActivityItemIdsRef.current),
    );
    const activeTabId = tabs.some((tab) => tab.id === activeToolActivityTabId)
      || builtinBrowserTabIds.includes(activeToolActivityTabId)
      ? activeToolActivityTabId
      : builtinBrowserTabIds.at(-1) || tabs.at(-1)?.id || "";
    toolActivitySessionStatesRef.current.set(activeSessionId, {
      activeTabId,
      browserInitialUrls: builtinBrowserInitialUrls,
      browserTabIds: builtinBrowserTabIds,
      closedItemIds: [...closedToolActivityItemIdsRef.current],
      isOpen: isRightSidebarOpen && (tabs.length > 0 || builtinBrowserTabIds.length > 0),
      tabs,
    });
  }, [
    activeSessionId,
    activeToolActivityTabId,
    builtinBrowserInitialUrls,
    builtinBrowserTabIds,
    isRightSidebarOpen,
    toolActivityTabs,
  ]);

  useEffect(() => {
    if (activeProjectPage !== "chat" || !isRightSidebarOpen) {
      setBrowserRevealReady(false);
      return;
    }

    setBrowserRevealReady(false);
    const timeout = window.setTimeout(() => {
      requestAnimationFrame(() => setBrowserRevealReady(true));
    }, BROWSER_REVEAL_AFTER_PANEL_MS);
    return () => window.clearTimeout(timeout);
  }, [activeProjectPage, isRightSidebarOpen]);

  useEffect(() => {
    const browser = window.aivo?.browser;
    if (!browser || builtinBrowserTabIds.length === 0) return;

    const visibleBrowserTabId =
      activeProjectPage === "chat" &&
      isRightSidebarOpen &&
      isBrowserRevealReady &&
      builtinBrowserTabIds.includes(activeToolActivityTabId)
        ? activeToolActivityTabId
        : "";

    for (const browserTabId of builtinBrowserTabIds) {
      if (browserTabId === visibleBrowserTabId) continue;
      void browser.setVisible(browserTabId, false).catch(() => undefined);
    }
  }, [
    activeProjectPage,
    activeToolActivityTabId,
    builtinBrowserTabIds,
    isBrowserRevealReady,
    isRightSidebarOpen,
  ]);

  useEffect(() => {
    if (!hasAppBridge() || !activeSessionId) return;
    let cancelled = false;
    void getPermissionMode(activeSessionId)
      .then((state) => {
        if (!cancelled) {
          setLocalPermissionMode(normalizePermissionMode(state?.mode));
        }
      })
      .catch(() => undefined);
    return () => {
      cancelled = true;
    };
  }, [activeSessionId]);

  useEffect(() => {
    setReasoningEffort(normalizeReasoningEffort(config?.reasoningEffort));
  }, [config?.reasoningEffort]);

  useEffect(() => {
    setServiceTier(normalizeServiceTier(config?.serviceTier));
  }, [config?.serviceTier]);

  useEffect(() => {
    setSelectedModelId((currentModelId) => {
      if (
        currentModelId &&
        modelOptions.some((model) => model.id === currentModelId)
      ) {
        return currentModelId;
      }
      return defaultModelId;
    });
  }, [defaultModelId, modelOptionsKey, modelOptions]);

  function selectModel(option: ModelOption) {
    setSelectedProviderId(option.providerId);
    setSelectedModelId(option.id);
    const nextServiceTier = providerSupportsServiceTier(option.providerId)
      ? serviceTier
      : "default";
    setServiceTier(nextServiceTier);
    void rememberModelPreferences(
      { providerId: option.providerId, modelId: option.id },
      reasoningEffort,
      nextServiceTier,
    );
  }

  function selectReasoningEffort(nextReasoningEffort: string) {
    const normalized = normalizeReasoningEffort(nextReasoningEffort);
    setReasoningEffort(normalized);
    void rememberModelPreferences(activeModelRef, normalized, serviceTier);
  }

  function selectServiceTier(nextServiceTier: string) {
    const normalized = normalizeServiceTier(nextServiceTier);
    setServiceTier(normalized);
    void rememberModelPreferences(activeModelRef, reasoningEffort, normalized);
  }

  function selectPermissionMode(nextMode: PermissionMode) {
    const normalized = normalizePermissionMode(nextMode);
    setLocalPermissionMode(normalized);
    const sessionId = activeSessionIdRef.current;
    if (!sessionId || !hasAppBridge()) return;
    void setPermissionMode(sessionId, normalized).catch(() => {
      toast.error("权限模式保存失败");
    });
  }

  function selectAgentMode(nextMode: AgentModeId) {
    setAgentMode(nextMode);
    const sessionId = activeSessionIdRef.current;
    if (!sessionId || !hasAppBridge()) return;
    void setSessionAgentMode(sessionId, nextMode)
      .then((session) => {
        setSessions((currentSessions) =>
          currentSessions.map((currentSession) =>
            currentSession.id === session.id ? session : currentSession,
          ),
        );
      })
      .catch(() => {
        toast.error("Agent 模式保存失败");
      });
  }

  async function cancelActiveSubagentRun() {
    if (!activeRunningSubagentRun?.id || !hasAppBridge()) return;
    try {
      await cancelAgentRun(activeRunningSubagentRun.id);
      await refreshAgentRuntimeState(activeSessionId);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "取消子代理失败");
    }
  }

  async function rememberModelPreferences(
    model: domain.ModelRef | undefined,
    nextReasoningEffort: string,
    nextServiceTier: string,
  ) {
    if (!model && !nextReasoningEffort && !nextServiceTier) return;
    if (!hasAppBridge()) return;
    try {
      await updateModelPreferences({
        model,
        reasoningEffort: normalizeReasoningEffort(nextReasoningEffort),
        serviceTier: normalizeServiceTier(nextServiceTier),
      } as domain.ModelPreferencesInput);
    } catch {
      // Preference persistence should not block composing or sending a message.
    }
  }

  function setConversationRunning(sessionId: string, running: boolean) {
    if (!sessionId) return;
    setRunningConversationIds((currentIds) => {
      const alreadyRunning = currentIds.includes(sessionId);
      if (running) {
        return alreadyRunning ? currentIds : [sessionId, ...currentIds];
      }
      return alreadyRunning
        ? currentIds.filter((currentId) => currentId !== sessionId)
        : currentIds;
    });
  }

  function setPendingPermissionCountForSession(
    sessionId: string,
    count: number,
  ) {
    setPendingPermissionCountsBySessionId((currentCounts) => {
      if ((currentCounts[sessionId] ?? 0) === count) {
        return currentCounts;
      }
      if (count === 0) {
        const nextCounts = { ...currentCounts };
        delete nextCounts[sessionId];
        return nextCounts;
      }
      return {
        ...currentCounts,
        [sessionId]: count,
      };
    });
  }

  function saveCurrentToolActivitySessionState(
    sessionId = activeSessionIdRef.current,
  ) {
    if (!sessionId) return;
    const tabs = toolActivityTabsRef.current.filter(
      (tab) => !toolActivityTabIsClosed(tab, closedToolActivityItemIdsRef.current),
    );
    const browserTabIds = builtinBrowserTabIdsRef.current;
    const currentActiveId = activeToolActivityTabIdRef.current;
    const activeTabId = tabs.some((tab) => tab.id === currentActiveId)
      || browserTabIds.includes(currentActiveId)
      ? currentActiveId
      : browserTabIds.at(-1) || tabs.at(-1)?.id || "";
    toolActivitySessionStatesRef.current.set(sessionId, {
      activeTabId,
      browserInitialUrls: builtinBrowserInitialUrlsRef.current,
      browserTabIds,
      closedItemIds: [...closedToolActivityItemIdsRef.current],
      isOpen: isRightSidebarOpenRef.current && (tabs.length > 0 || browserTabIds.length > 0),
      tabs,
    });
  }

  function restoreToolActivitySessionState(sessionId: string) {
    const savedState = toolActivitySessionStatesRef.current.get(sessionId);
    const tabs = savedState?.tabs ?? [];
    const browserTabIds = savedState?.browserTabIds ?? [];
    closedToolActivityItemIdsRef.current = new Set(savedState?.closedItemIds ?? []);
    setToolActivityTabs(tabs);
    setBuiltinBrowserInitialUrls(savedState?.browserInitialUrls ?? {});
    setBuiltinBrowserTabIds(browserTabIds);
    setActiveToolActivityTabId(
      tabs.some((tab) => tab.id === savedState?.activeTabId)
        || browserTabIds.includes(savedState?.activeTabId || "")
        ? savedState?.activeTabId || ""
        : browserTabIds.at(-1) || tabs.at(-1)?.id || "",
    );
    setRightSidebarOpen(
      Boolean(
        SHOULD_AUTO_OPEN_TOOL_ACTIVITY_SIDEBAR &&
          savedState?.isOpen &&
          (tabs.length > 0 || browserTabIds.length > 0),
      ),
    );
  }

  const mergeToolActivityFromCall = useCallback((toolCall: domain.ToolCall) => {
    const nextTabs = toolActivityTabsFromToolCall(toolCall).filter(
      (tab) => !toolActivityTabIsClosed(tab, closedToolActivityItemIdsRef.current),
    );
    if (nextTabs.length === 0) return;
    setToolActivityTabs((currentTabs) =>
      upsertToolActivityTabs(currentTabs, nextTabs),
    );
    setActiveToolActivityTabId(nextTabs.at(-1)?.id ?? "");
    if (SHOULD_AUTO_OPEN_TOOL_ACTIVITY_SIDEBAR) {
      setRightSidebarOpen(true);
    }
    if (toolCall.turnId && toolCall.status === "success" && hasAppBridge()) {
      void getSessionTurnDiff({
        sessionId: toolCall.sessionId,
        turnId: toolCall.turnId,
      })
        .then((diff) => {
          setToolActivityTabs((currentTabs) =>
            annotateToolActivityTabsWithFileStates(currentTabs, diff.files),
          );
        })
        .catch(() => undefined);
    }
  }, []);

  async function refreshToolActivityTabs(sessionId = activeSessionIdRef.current) {
    if (!hasAppBridge() || !sessionId) return;
    const toolCalls = (await listSessionToolCalls(sessionId).catch(
      () => [] as domain.ToolCall[],
    )) ?? [];
    const baseTabs = toolActivityTabsFromToolCalls(toolCalls).filter(
      (tab) => !toolActivityTabIsClosed(tab, closedToolActivityItemIdsRef.current),
    );
    const tabs = await annotateToolActivityTabsWithTurnDiff(sessionId, baseTabs);
    setToolActivityTabs(tabs);
    setActiveToolActivityTabId((currentId) =>
      tabs.some((tab) => tab.id === currentId) ||
      builtinBrowserTabIdsRef.current.includes(currentId)
        ? currentId
        : builtinBrowserTabIdsRef.current.at(-1) || tabs.at(-1)?.id || "",
    );
    setRightSidebarOpen(
      (current) => current && (tabs.length > 0 || builtinBrowserTabIdsRef.current.length > 0),
    );
  }

  async function annotateToolActivityTabsWithTurnDiff(
    sessionId: string,
    tabs: ToolActivityTab[],
  ) {
    const turnIds = Array.from(
      new Set(
        tabs.flatMap((tab) =>
          tab.kind === "file" && tab.turnId ? [tab.turnId] : [],
        ),
      ),
    );
    if (turnIds.length === 0) return tabs;
    const states = (
      await Promise.all(
        turnIds.map(async (turnId) => {
          try {
            const diff = await getSessionTurnDiff({ sessionId, turnId });
            return diff.files;
          } catch {
            return [];
          }
        }),
      )
    ).flat();
    return annotateToolActivityTabsWithFileStates(tabs, states);
  }

  async function applyToolActivityFileState(
    tab: ToolActivityFileTab,
    targetState: "before" | "after",
  ) {
    const sessionId = activeSessionIdRef.current;
    if (!hasAppBridge() || !sessionId || !tab.turnId) return;
    try {
      await applySessionTurnFileState({
        sessionId,
        turnId: tab.turnId,
        toolCallId: tab.toolCallId,
        path: tab.relativePath || tab.path,
        targetState,
      });
      await refreshToolActivityTabs(sessionId);
      await loadConversationTurns(sessionId);
      toast.success(targetState === "before" ? "已回滚文件改动" : "已恢复文件改动");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    }
  }

  const refreshPendingPermissionRequests = useCallback(
    async (sessionId = activeSessionIdRef.current) => {
      if (!hasAppBridge() || !sessionId) {
        setPendingPermissionRequests([]);
        return;
      }
      const requests =
        (await listPermissionRequests(sessionId, "pending").catch(
          () => [] as PermissionRequest[],
        )) ?? [];
      if (activeSessionIdRef.current !== sessionId) return;
      setPendingPermissionRequests((currentRequests) =>
        samePermissionRequests(currentRequests, requests)
          ? currentRequests
          : requests,
      );
      setPendingPermissionCountForSession(sessionId, requests.length);
      if (requests.length > 0) {
        setTurns((currentTurns) =>
          mergePendingPermissionToolCalls(currentTurns, requests),
        );
      }
    },
    [],
  );

  const refreshPendingQuestionRequests = useCallback(
    async (sessionId = activeSessionIdRef.current) => {
      if (!hasAppBridge() || !sessionId) {
        setPendingQuestionRequests([]);
        return;
      }
      const requests =
        (await listQuestionRequests(sessionId, "pending").catch(
          () => [] as QuestionRequest[],
        )) ?? [];
      if (activeSessionIdRef.current !== sessionId) return;
      setPendingQuestionRequests((currentRequests) =>
        sameQuestionRequests(currentRequests, requests)
          ? currentRequests
          : requests,
      );
    },
    [],
  );

  useEffect(() => {
    if (!activeSessionId) {
      setPendingPermissionRequests([]);
      return;
    }
    void refreshPendingPermissionRequests(activeSessionId);
  }, [activeSessionId, refreshPendingPermissionRequests]);

  useEffect(() => {
    if (!activeSessionId) {
      setPendingQuestionRequests([]);
      return;
    }
    void refreshPendingQuestionRequests(activeSessionId);
  }, [activeSessionId, refreshPendingQuestionRequests]);

  async function addComposerAttachmentFiles(files: FileList | null) {
    if (!files?.length) return;
    const result = await readComposerAttachmentFiles(
      Array.from(files),
      activeModelRef,
      modelOptions.find((model) => model.id === activeModelId),
    );
    for (const message of result.rejections) {
      toast.error(message);
    }
    if (result.attachments.length === 0) return;
    setComposerAttachments((current) => [
      ...current,
      ...result.attachments,
    ]);
  }

  function handleComposerDragEnter(event: React.DragEvent<HTMLDivElement>) {
    if (!dragEventHasFiles(event)) return;
    event.preventDefault();
    event.stopPropagation();
    composerDropDepthRef.current += 1;
    setComposerDropActive(true);
  }

  function handleComposerDragOver(event: React.DragEvent<HTMLDivElement>) {
    if (!dragEventHasFiles(event)) return;
    event.preventDefault();
    event.stopPropagation();
    event.dataTransfer.dropEffect = "copy";
    setComposerDropActive(true);
  }

  function handleComposerDragLeave(event: React.DragEvent<HTMLDivElement>) {
    if (!dragEventHasFiles(event)) return;
    event.preventDefault();
    event.stopPropagation();
    composerDropDepthRef.current = Math.max(0, composerDropDepthRef.current - 1);
    if (composerDropDepthRef.current === 0) {
      setComposerDropActive(false);
    }
  }

  function handleComposerDrop(event: React.DragEvent<HTMLDivElement>) {
    if (!dragEventHasFiles(event)) return;
    event.preventDefault();
    event.stopPropagation();
    composerDropDepthRef.current = 0;
    setComposerDropActive(false);
    void addComposerAttachmentFiles(event.dataTransfer.files);
  }

  function removeComposerAttachment(id: string) {
    setComposerAttachments((current) =>
      current.filter((attachment) => attachment.id !== id),
    );
  }

  useEffect(() => {
    if (!hasAppBridge()) return;
    void listSessions(50)
      .then((nextSessions) => setSessions(nextSessions ?? []))
      .catch(() => setSessions([]));
  }, []);

  const refreshRecentProjects = useCallback(async () => {
    if (!hasAppBridge()) return;
    try {
      const projects = await listRecentProjects(20);
      setRecentProjects((projects ?? []).filter(projectIsUserSelectable));
    } catch {
      setRecentProjects([]);
    }
  }, []);

  useEffect(() => {
    void refreshRecentProjects();
  }, [refreshRecentProjects]);

  function selectComposerProject(project: domain.AssistantProject) {
    if (activeSessionIdRef.current) {
      startNewConversation({ preservePrompt: true });
    }
    setSelectedProjectPath(project.rootPath);
  }

  function clearComposerProject() {
    if (activeSessionIdRef.current) {
      startNewConversation({ preservePrompt: true });
    }
    setSelectedProjectPath("");
  }

  async function addComposerProject() {
    if (!hasAppBridge()) return;
    try {
      const rootPath = await selectProjectDirectory();
      if (!rootPath) return;
      const project = await upsertProject(rootPath);
      if (projectIsUserSelectable(project)) {
        setRecentProjects((currentProjects) =>
          upsertRecentProject(currentProjects, project),
        );
      }
      if (activeSessionIdRef.current) {
        startNewConversation({ preservePrompt: true });
      }
      setSelectedProjectPath(rootPath);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "选择项目失败");
    }
  }

  useEffect(() => {
    if (!hasAppBridge()) return;
    return EventsOn("session.updated", (...payloads: unknown[]) => {
      const payload = normalizeSessionUpdatedPayload(payloads);
      if (payload.session) {
        setSessions((currentSessions) =>
          upsertSession(currentSessions, payload.session!),
        );
      }
    });
  }, []);

  useEffect(() => {
    if (!hasAppBridge()) return;
    const handleTurnEvent = (...payloads: unknown[]) => {
      const turn = normalizeTurnUpdatedPayload(payloads);
      if (!turn?.id || !turn.sessionId) return;
      setConversationRunning(turn.sessionId, turn.status === "running");
      if (turn.sessionId !== activeSessionIdRef.current) return;
      setTurns((currentTurns) => mergeRuntimeTurn(currentTurns, turn));
    };
    const offStarted = EventsOn("turn.started", handleTurnEvent);
    const offCompleted = EventsOn("turn.completed", handleTurnEvent);
    const offFailed = EventsOn("turn.failed", handleTurnEvent);
    const offCancelled = EventsOn("turn.cancelled", handleTurnEvent);
    return () => {
      offStarted();
      offCompleted();
      offFailed();
      offCancelled();
    };
  }, []);

  useEffect(() => {
    if (!hasAppBridge()) return;
    const handleToolCallCreatedEvent = (...payloads: unknown[]) => {
      const toolCall = normalizeToolCallUpdatedPayload(payloads);
      if (!toolCall?.id || !toolCall.sessionId) return;
      if (toolCall.sessionId !== activeSessionIdRef.current) return;
      flushPendingAssistantDelta();
      setTurns((currentTurns) =>
        moveOpenResponseTextToAssistantPreambleBeforeTool(
          mergeSingleToolCall(currentTurns, toolCall),
          toolCall,
        ),
      );
      mergeToolActivityFromCall(toolCall);
      if (isDelegateTaskToolName(toolCall.name)) {
        void refreshAgentRuntimeState(toolCall.sessionId);
      }
    };
    const handleToolCallUpdatedEvent = (...payloads: unknown[]) => {
      const toolCall = normalizeToolCallUpdatedPayload(payloads);
      if (!toolCall?.id || !toolCall.sessionId) return;
      if (toolCall.sessionId !== activeSessionIdRef.current) return;
      setTurns((currentTurns) => mergeSingleToolCall(currentTurns, toolCall));
      mergeToolActivityFromCall(toolCall);
      if (isDelegateTaskToolName(toolCall.name)) {
        void refreshAgentRuntimeState(toolCall.sessionId);
      }
    };
    const offCreated = EventsOn("tool_call.created", handleToolCallCreatedEvent);
    const offUpdated = EventsOn("tool_call.updated", handleToolCallUpdatedEvent);
    return () => {
      offCreated();
      offUpdated();
    };
  }, [flushPendingAssistantDelta, mergeToolActivityFromCall, refreshAgentRuntimeState]);

  useEffect(() => {
    if (!hasAppBridge()) return;
    return EventsOn("todo_items.updated", (...payloads: unknown[]) => {
      const payload = normalizeTodoItemsUpdatedPayload(payloads);
      if (!payload || payload.sessionId !== activeSessionIdRef.current) return;
      if (
        payload.projectPath &&
        activeWorkspaceRoot &&
        payload.projectPath !== activeWorkspaceRoot
      ) {
        return;
      }
      setTodoItems(payload.items);
    });
  }, [activeWorkspaceRoot]);

  useEffect(() => {
    if (!hasAppBridge()) return;
    return EventsOn("permission.requested", (...payloads: unknown[]) => {
      const permission = normalizePermissionEventPayload(payloads);
      if (!permission?.id || !permission.sessionId) return;
      if (permission.sessionId !== activeSessionIdRef.current) {
        setPendingPermissionCountsBySessionId((currentCounts) => ({
          ...currentCounts,
          [permission.sessionId!]: Math.max(
            currentCounts[permission.sessionId!] ?? 0,
            1,
          ),
        }));
        return;
      }
      setPendingPermissionRequests((currentRequests) => {
        const nextRequests = upsertPermissionRequest(
          currentRequests,
          permission,
        );
        return samePermissionRequests(currentRequests, nextRequests)
          ? currentRequests
          : nextRequests;
      });
      setPendingPermissionCountForSession(permission.sessionId, 1);
      setTurns((currentTurns) =>
        mergePendingPermissionToolCalls(currentTurns, [permission]),
      );
      mergeToolActivityFromCall(permissionToolCall(permission));
    });
  }, [mergeToolActivityFromCall]);

  useEffect(() => {
    if (!hasAppBridge()) return;
    return EventsOn("permission.resolved", (...payloads: unknown[]) => {
      const permission = normalizePermissionEventPayload(payloads);
      if (!permission?.id || !permission.sessionId) return;
      setPendingPermissionCountsBySessionId((currentCounts) => {
        const currentCount = currentCounts[permission.sessionId!] ?? 0;
        const nextCount = Math.max(0, currentCount - 1);
        if (nextCount === currentCount) return currentCounts;
        if (nextCount === 0) {
          const nextCounts = { ...currentCounts };
          delete nextCounts[permission.sessionId!];
          return nextCounts;
        }
        return { ...currentCounts, [permission.sessionId!]: nextCount };
      });
      if (permission.sessionId !== activeSessionIdRef.current) return;
      setPendingPermissionRequests((currentRequests) => {
        const nextRequests = currentRequests.filter(
          (currentRequest) => currentRequest.id !== permission.id,
        );
        return nextRequests.length === currentRequests.length
          ? currentRequests
          : nextRequests;
      });
    });
  }, []);

  useEffect(() => {
    if (!hasAppBridge()) return;
    return EventsOn("question.requested", (...payloads: unknown[]) => {
      const question = normalizeQuestionEventPayload(payloads);
      if (!question?.id || !question.sessionId) return;
      if (question.sessionId !== activeSessionIdRef.current) return;
      setPendingQuestionRequests((currentRequests) => {
        const nextRequests = upsertQuestionRequest(currentRequests, question);
        return sameQuestionRequests(currentRequests, nextRequests)
          ? currentRequests
          : nextRequests;
      });
    });
  }, []);

  useEffect(() => {
    if (!hasAppBridge()) return;
    return EventsOn("question.resolved", (...payloads: unknown[]) => {
      const question = normalizeQuestionEventPayload(payloads);
      if (!question?.id || !question.sessionId) return;
      if (question.sessionId !== activeSessionIdRef.current) return;
      setPendingQuestionRequests((currentRequests) => {
        const nextRequests = currentRequests.filter(
          (currentRequest) => currentRequest.id !== question.id,
        );
        return nextRequests.length === currentRequests.length
          ? currentRequests
          : nextRequests;
      });
    });
  }, []);

  useEffect(() => {
    if (!hasAppBridge()) return;
    return EventsOn("assistant.delta", (...payloads: unknown[]) => {
      const payload = normalizeAssistantDeltaPayload(payloads);
      const delta = payload.delta;
      if (!delta || payload?.sessionId !== activeSessionIdRef.current) return;
      if (window.localStorage.getItem("aivo:debug-stream") === "1") {
        console.debug("[aivo-stream] assistant.delta", {
          length: delta.length,
          preview: delta.slice(0, 120),
          sessionId: payload.sessionId,
          turnId: payload.turnId,
        });
      }
      enqueueAssistantDelta({
        delta,
        sessionId: payload.sessionId,
        turnId: payload.turnId,
      });
    });
  }, [enqueueAssistantDelta]);

  useEffect(() => {
    if (!hasAppBridge()) return;
    return EventsOn("shell.output", (...payloads: unknown[]) => {
      const payload = normalizeShellOutputPayload(payloads);
      if (!payload.toolCallId || payload.sessionId !== activeSessionIdRef.current) {
        return;
      }
      if (closedToolActivityItemIdsRef.current.has(toolActivityToolCallKey(payload.toolCallId))) {
        return;
      }
      setToolActivityTabs((currentTabs) =>
        appendShellOutputToTabs(currentTabs, payload),
      );
      setActiveToolActivityTabId(`command:shell:${payload.sessionId || "current"}`);
        if (SHOULD_AUTO_OPEN_TOOL_ACTIVITY_SIDEBAR) {
          setRightSidebarOpen(true);
        }
    });
  }, []);

  useEffect(() => {
    if (!hasAppBridge()) return;
    return EventsOn("events.reconnected", () => {
      void listSessions(50)
        .then((nextSessions) => setSessions(nextSessions ?? []))
        .catch(() => undefined);
      const sessionId = activeSessionIdRef.current;
      if (sessionId) {
        void loadConversationTurns(sessionId, { snapToBottomAfterLoad: true });
        void listPermissionRequests(sessionId, "pending")
          .then((requests) => {
            const nextRequests = requests ?? [];
            setPendingPermissionRequests((currentRequests) =>
              samePermissionRequests(currentRequests, nextRequests)
                ? currentRequests
                : nextRequests,
            );
            setPendingPermissionCountForSession(sessionId, nextRequests.length);
          })
          .catch(() => undefined);
        void listQuestionRequests(sessionId, "pending")
          .then((requests) => {
            const nextRequests = requests ?? [];
            setPendingQuestionRequests((currentRequests) =>
              sameQuestionRequests(currentRequests, nextRequests)
                ? currentRequests
                : nextRequests,
            );
          })
          .catch(() => undefined);
      }
    });
  }, [activeSessionId]);

  useEffect(() => {
    const now = Date.now();
    setTurns((currentTurns) =>
      updatePermissionPauseState(currentTurns, pendingPermissionRequests, now),
    );
  }, [pendingPermissionRequests]);

  useLayoutEffect(() => {
    const mainElement = mainRef.current;
    if (!mainElement) return;

    const updatePinnedSummaryLayout = () => {
      const messageWidth = 680;
      const summaryWidth = 288;
      const sideInset = 24;
      const summaryGap = 24;
      const dockShift = 160;
      const mainWidth = mainElement.clientWidth;
      const contentWidth = Math.min(messageWidth, Math.max(0, mainWidth - 48));
      const contentRight = (mainWidth + contentWidth) / 2;
      const summaryLeft = mainWidth - sideInset - summaryWidth;
      const hasNaturalDockSpace = summaryLeft - contentRight >= summaryGap;
      const hasShiftedDockSpace =
        summaryLeft - (contentRight - dockShift) >= summaryGap;
      const nextCanDock = hasNaturalDockSpace || hasShiftedDockSpace;
      const nextShouldShift = !hasNaturalDockSpace && hasShiftedDockSpace;

      setCanDockPinnedSummary((current) =>
        current === nextCanDock ? current : nextCanDock,
      );
      setShouldShiftPinnedSummaryLayout((current) =>
        current === nextShouldShift ? current : nextShouldShift,
      );
    };

    updatePinnedSummaryLayout();
    const resizeObserver = new ResizeObserver(updatePinnedSummaryLayout);
    resizeObserver.observe(mainElement);

    return () => resizeObserver.disconnect();
  }, []);

  async function submitPrompt() {
    const nextPrompt = prompt.trim();
    if ((!nextPrompt && composerAttachments.length === 0) || hasPendingTurn) return;
    const activeModel = modelOptions.find((model) => model.id === activeModelId);
    const unsupportedAttachment = composerAttachments.find(
      (attachment) =>
        !modelSupportsAttachment(
          activeModelRef,
          activeModel,
          attachment.kind,
          attachment.mimeType,
        ),
    );
    if (unsupportedAttachment) {
      toast.error(
        `当前模型不支持${attachmentKindLabel(unsupportedAttachment.kind, unsupportedAttachment.mimeType)}：${unsupportedAttachment.name}`,
      );
      return;
    }
    const localTurnId = crypto.randomUUID();
    const startedAt = Date.now();
    const submittedAttachments = composerAttachments;
    const submittedTimelineAttachments = submittedAttachments.map(
      composerAttachmentToConversationAttachment,
    );
    const displayPrompt = nextPrompt || formatAttachmentOnlyPrompt(submittedAttachments);
    setTurns((currentTurns) => [
      ...currentTurns,
      {
        id: localTurnId,
        activityVisible: false,
        assistantPreambles: [],
        attachments: submittedTimelineAttachments,
        prompt: displayPrompt,
        preToolText: "",
        responseText: "",
        responseCompletedAt: null,
        responseVisible: false,
        startedAt,
        submittedAt: new Date(),
        stopped: false,
        thinkingSeconds: 0,
        toolCalls: [],
      },
    ]);
    setPrompt("");
    setComposerAttachments([]);
    if (!hasAppBridge()) {
      setTurns((currentTurns) =>
        currentTurns.map((turn) =>
          turn.id === localTurnId
            ? {
                ...turn,
                responseCompletedAt: new Date(),
                responseText:
                  "当前运行环境未连接 Aivo 后端，无法发送真实 provider 请求。",
                responseVisible: true,
                thinkingSeconds: getTurnElapsedSeconds({ startedAt }),
                toolCalls: [],
              }
            : turn,
        ),
      );
      return;
    }
    let submittedSessionId = activeSessionId;
    try {
      let sessionId = submittedSessionId;
      if (!sessionId) {
        const session = await createSession({
          type: "coding",
          source: "desktop",
          projectPath: selectedProjectPath,
          model: activeModelRef,
          agentMode,
        } as domain.CreateSessionRequest & { agentMode?: AgentModeId });
	        sessionId = session.id;
	        activeSessionIdRef.current = session.id;
	        setActiveSessionId(session.id);
	        setCodingWorkspaceRoot(session.projectPath || selectedProjectPath);
	        if (defaultActiveToolNames.length > 0) {
	          await setSessionActiveTools(session.id, defaultActiveToolNames);
	        }
	      }
      submittedSessionId = sessionId;
      await setPermissionMode(sessionId, permissionMode);
      setConversationRunning(sessionId, true);
      const run = await submitSessionMessage({
        sessionId,
        text: nextPrompt,
        attachments: submittedAttachments.map((attachment) => ({
          id: attachment.id,
          name: attachment.name,
          mimeType: attachment.mimeType,
          kind: attachment.kind,
          data: attachment.data,
          size: attachment.size,
        })),
        model: activeModelRef,
        agentMode,
        reasoningEffort,
        serviceTier:
          activeModelRef &&
          providerSupportsServiceTier(activeModelRef.providerId)
            ? serviceTier
            : "default",
      } as domain.SubmitSessionMessageRequest & {
        agentMode?: AgentModeId;
        attachments?: Array<{
          id: string;
          name: string;
          mimeType: string;
          kind: string;
          data: string;
          size: number;
        }>;
      });
      void refreshPendingPermissionRequests(sessionId);
      if (run.turn?.id || run.userEvent?.id) {
        setTurns((currentTurns) =>
          currentTurns.map((turn) =>
            turn.id === localTurnId
              ? {
                  ...turn,
                  turnId: run.turn?.id || turn.turnId,
                  userEventId: run.userEvent?.id || turn.userEventId,
                }
              : turn,
          ),
        );
        if (pendingStopRequestedRef.current) {
          pendingStopRequestedRef.current = false;
          await cancelSessionTurn({
            turnId: run.turn.id,
            reason: "User stopped generation",
          } as domain.CancelTurnRequest);
          void refreshPendingPermissionRequests(sessionId);
          setConversationRunning(sessionId, false);
          setSessions((await listSessions(50)) ?? []);
          return;
        }
      }
      if (run.turn?.status !== "running") {
        await loadConversationTurns(sessionId, {
          pendingTurnId: localTurnId,
          pendingPrompt: displayPrompt,
          pendingAttachments: submittedTimelineAttachments,
          pendingStartedAt: startedAt,
          fallbackAssistantEvent: run.assistantEvent,
        });
      }
      setSessions((await listSessions(50)) ?? []);
    } catch (err) {
      setComposerAttachments((current) => [...submittedAttachments, ...current]);
      setConversationRunning(submittedSessionId, false);
      setTurns((currentTurns) =>
        currentTurns.map((turn) =>
          turn.id === localTurnId
            ? {
                ...turn,
                responseCompletedAt: new Date(),
                responseText: err instanceof Error ? err.message : String(err),
                responseVisible: true,
                thinkingSeconds: getTurnElapsedSeconds({ startedAt }),
                toolCalls: [],
              }
            : turn,
        ),
      );
    }
  }

  async function stopPendingTurn() {
    const sessionId = activeSessionIdRef.current;
    const turnToStop = [...turns]
      .reverse()
      .find((turn) => !turn.responseCompletedAt && !turn.stopped);
    setConversationRunning(sessionId, false);
    setTurns((currentTurns) =>
      currentTurns.map((turn) =>
        turn.responseCompletedAt || turn.stopped
          ? turn
          : {
              ...turn,
              stopped: true,
              thinkingSeconds: getTurnElapsedSeconds(turn),
            },
      ),
    );
    if (!hasAppBridge()) return;
    if (!turnToStop?.turnId) {
      pendingStopRequestedRef.current = true;
      return;
    }
    try {
      await cancelSessionTurn({
        turnId: turnToStop.turnId,
        reason: "User stopped generation",
      } as domain.CancelTurnRequest);
      void refreshPendingPermissionRequests(sessionId);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    }
  }

  async function editConversationUserMessage(turn: ConversationTurn) {
    if (!hasAppBridge() || !activeSessionIdRef.current || !turn.userEventId)
      return;
    const nextContent = window.prompt("编辑消息", turn.prompt);
    if (nextContent === null) return;
    const content = nextContent.trim();
    if (!content) {
      toast.error("消息不能为空");
      return;
    }
    try {
      await updateSessionEvent({ eventId: turn.userEventId, content });
      await loadConversationTurns(activeSessionIdRef.current);
      setSessions((await listSessions(50)) ?? []);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    }
  }

  async function deleteConversationTurn(turn: ConversationTurn) {
    if (!hasAppBridge() || !activeSessionIdRef.current) return;
    const eventIds = [turn.userEventId, turn.assistantEventId].filter(
      Boolean,
    ) as string[];
    if (eventIds.length === 0) return;
    try {
      await Promise.all(
        eventIds.map((eventId) => deleteSessionEvent({ eventId })),
      );
      await loadConversationTurns(activeSessionIdRef.current);
      setSessions((await listSessions(50)) ?? []);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    }
  }

  async function deleteConversationAssistantMessage(turn: ConversationTurn) {
    if (!hasAppBridge() || !activeSessionIdRef.current || !turn.assistantEventId)
      return;
    try {
      await deleteSessionEvent({ eventId: turn.assistantEventId });
      await loadConversationTurns(activeSessionIdRef.current);
      setSessions((await listSessions(50)) ?? []);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    }
  }

  async function retryConversationTurn(turn: ConversationTurn) {
    const sessionId = activeSessionIdRef.current;
    if (!hasAppBridge() || !sessionId || !turn.turnId || hasPendingTurn) return;
    try {
      setConversationRunning(sessionId, true);
      await retrySessionTurn({
        sessionId,
        turnId: turn.turnId,
        model: activeModelRef,
        agentMode,
        reasoningEffort,
        serviceTier:
          activeModelRef &&
          providerSupportsServiceTier(activeModelRef.providerId)
            ? serviceTier
            : "default",
      });
      await loadConversationTurns(sessionId, { snapToBottomAfterLoad: true });
      void refreshPendingPermissionRequests(sessionId);
      setSessions((await listSessions(50)) ?? []);
    } catch (err) {
      setConversationRunning(sessionId, false);
      toast.error(err instanceof Error ? err.message : String(err));
    }
  }

  async function loadConversationTurns(
    sessionId: string,
    options: {
      pendingTurnId?: string;
      pendingPrompt?: string;
      pendingAttachments?: ConversationUserAttachment[];
      pendingStartedAt?: number;
      fallbackAssistantEvent?: domain.SessionEvent;
      snapToBottomAfterLoad?: boolean;
    } = {},
  ) {
    const [events, toolCalls, runtimeTurns] = await Promise.all([
      listSessionEvents(sessionId, false, 100),
      listSessionToolCalls(sessionId).catch(() => [] as domain.ToolCall[]),
      listSessionTurns(sessionId, 100).catch(() => [] as domain.Turn[]),
    ]);
    let nextTurns = turnsFromEvents(
      events ?? [],
      toolCalls ?? [],
      runtimeTurns ?? [],
    );
    if (
      nextTurns.length === 0 &&
      options.fallbackAssistantEvent &&
      options.pendingPrompt
    ) {
      const submittedAt = new Date(options.pendingStartedAt ?? Date.now());
      const completedAt = parseTime(options.fallbackAssistantEvent.timeCreated);
      nextTurns = [
        {
          id: options.pendingTurnId ?? options.fallbackAssistantEvent.id,
          activityVisible: (toolCalls ?? []).some(
            (toolCall) =>
              toolCall.turnId === options.fallbackAssistantEvent?.turnId,
          ),
          assistantPreambles: [],
          prompt: options.pendingPrompt,
          attachments:
            options.pendingAttachments ??
            mergePreservedTurnAttachments(options.pendingTurnId, turns),
          preToolText: "",
          responseCompletedAt: completedAt,
          responseText: options.fallbackAssistantEvent.content ?? "",
          responseVisible: true,
          startedAt: submittedAt.getTime(),
          stopped: false,
          submittedAt,
          thinkingSeconds: getTurnElapsedSeconds({
            startedAt: options.pendingStartedAt ?? submittedAt.getTime(),
          }),
          toolCalls: toolCallsForTurn(
            toolCalls ?? [],
            options.fallbackAssistantEvent.turnId,
          ),
          turnId: options.fallbackAssistantEvent.turnId,
          assistantEventId: options.fallbackAssistantEvent.id,
        },
      ];
    }
    const hydratedTurns = mergeTurnPauseMetadata(
      applyPendingTurnMetadata(nextTurns, options),
      turns,
    );
    setConversationRunning(sessionId, hasRunningTurn(hydratedTurns));
    if (activeSessionIdRef.current !== sessionId) {
      return;
    }
    if (options.snapToBottomAfterLoad) {
      snapNextMessageScrollRef.current = true;
      forceStickToBottomRef.current = true;
      stickToBottomRef.current = true;
      previousTurnCountRef.current = hydratedTurns.length;
      startForceScrollToBottom();
    }
    setTurns(hydratedTurns);
  }

  async function openConversation(session: domain.Session) {
    if (!hasAppBridge()) return;

    const isDifferentSession = session.id !== activeSessionIdRef.current;
    if (isDifferentSession) {
      saveCurrentToolActivitySessionState();
    }
    const shouldAnimateFromEmpty = !hasTurns && isDifferentSession;
    const selectionId = sidebarConversationSelectionRef.current + 1;
    sidebarConversationSelectionRef.current = selectionId;

    if (shouldAnimateFromEmpty) {
      captureComposerTransitionStart();
      setOpeningConversationFromEmpty(true);
      setRevealingHistoryConversation(false);
    } else {
      setRevealingHistoryConversation(false);
    }

    activeSessionIdRef.current = session.id;
    setActiveSessionId(session.id);
    restoreToolActivitySessionState(session.id);

    try {
      if (shouldAnimateFromEmpty) {
        await delay(OPEN_CONVERSATION_FROM_EMPTY_DELAY);
      }
      if (sidebarConversationSelectionRef.current !== selectionId) {
        return;
      }
      setRevealingHistoryConversation(isDifferentSession);
      await loadConversationTurns(session.id, {
        snapToBottomAfterLoad: isDifferentSession,
      });
      await refreshPendingPermissionRequests(session.id);
    } finally {
      if (
        shouldAnimateFromEmpty &&
        sidebarConversationSelectionRef.current === selectionId
      ) {
        setOpeningConversationFromEmpty(false);
      }
    }
  }

  async function openConversationById(sessionId: string) {
    if (!hasAppBridge() || !sessionId) return;
    let session = sessions.find((candidate) => candidate.id === sessionId);
    if (!session) {
      const nextSessions = (await listSessions(50)) ?? [];
      setSessions(nextSessions);
      session = nextSessions.find((candidate) => candidate.id === sessionId);
    }
    if (!session) {
      toast.error("找不到子代理会话");
      return;
    }
    await openConversation(session);
  }

  async function selectSidebarConversation(session: domain.Session) {
    await openConversation(session);
  }

  const updateStickToBottom = useCallback((viewport: HTMLDivElement) => {
    if (forceStickToBottomRef.current) {
      stickToBottomRef.current = true;
      setShowScrollToBottomButton(false);
      return;
    }

    const distanceToBottom =
      viewport.scrollHeight - viewport.scrollTop - viewport.clientHeight;
    const isAtBottom = distanceToBottom < SHOW_SCROLL_TO_BOTTOM_DISTANCE;
    stickToBottomRef.current = isAtBottom;
    setShowScrollToBottomButton(!isAtBottom);
  }, []);

  const scrollMessagesToBottom = useCallback(
    (behavior: ScrollBehavior = "smooth") => {
      const viewport = messagesViewportRef.current;
      if (!viewport) return;

      if (behavior === "auto") {
        viewport.scrollTop = SCROLL_BOTTOM_SENTINEL;
        setShowScrollToBottomButton(false);
        return;
      }

      setShowScrollToBottomButton(false);
      viewport.scrollTo({
        behavior,
        top: SCROLL_BOTTOM_SENTINEL,
      });
    },
    [],
  );

  const handleScrollToBottomButtonClick = useCallback(() => {
    stickToBottomRef.current = true;
    setShowScrollToBottomButton(false);
    scrollMessagesToBottom("smooth");
  }, [scrollMessagesToBottom]);

  const hideCompletedTodoPlan = useCallback(() => {
    if (!isVisibleTodoPlanComplete || !visibleTodoPlanKey) return;
    setHiddenTodoPlanKeyForSession(activeSessionId, visibleTodoPlanKey);
  }, [
    activeSessionId,
    isVisibleTodoPlanComplete,
    setHiddenTodoPlanKeyForSession,
    visibleTodoPlanKey,
  ]);

  const stopScrollAnimation = useCallback(() => {
    if (scrollAnimationFrameRef.current) {
      window.cancelAnimationFrame(scrollAnimationFrameRef.current);
    }
    scrollAnimationFrameRef.current = 0;
  }, []);

  const animateMessagesToBottom = useCallback(() => {
    const viewport = messagesViewportRef.current;
    if (!viewport) return;

    if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) {
      scrollMessagesToBottom("auto");
      return;
    }

    stopScrollAnimation();

    const startTop = viewport.scrollTop;
    const startTime = performance.now();

    const scrollStep = (time: number) => {
      const currentViewport = messagesViewportRef.current;
      if (!currentViewport) {
        scrollAnimationFrameRef.current = 0;
        return;
      }

      const progress = Math.min(
        1,
        (time - startTime) / SCROLL_BOTTOM_ANIMATION_MS,
      );
      const easedProgress = 1 - Math.pow(1 - progress, 3);
      const targetTop = Math.max(
        0,
        currentViewport.scrollHeight - currentViewport.clientHeight,
      );
      currentViewport.scrollTop =
        startTop + (targetTop - startTop) * easedProgress;

      if (progress < 1) {
        scrollAnimationFrameRef.current =
          window.requestAnimationFrame(scrollStep);
        return;
      }

      scrollAnimationFrameRef.current = 0;
      currentViewport.scrollTop = SCROLL_BOTTOM_SENTINEL;
    };

    scrollAnimationFrameRef.current = window.requestAnimationFrame(scrollStep);
  }, [scrollMessagesToBottom, stopScrollAnimation]);

  const updateStickToBottomFromCurrentViewport = useCallback(() => {
    const viewport = messagesViewportRef.current;
    if (!viewport) return;
    updateStickToBottom(viewport);
  }, [updateStickToBottom]);

  const startForceScrollToBottom = useCallback(
    (frameCount = FORCE_BOTTOM_FRAME_COUNT) => {
      forceStickToBottomRef.current = true;
      forceBottomRemainingFramesRef.current = Math.max(
        forceBottomRemainingFramesRef.current,
        frameCount,
      );
      if (forceBottomFrameRef.current) return;

      const scrollStep = () => {
        if (!scrollAnimationFrameRef.current) {
          scrollMessagesToBottom("auto");
        }
        forceBottomRemainingFramesRef.current -= 1;

        if (forceBottomRemainingFramesRef.current > 0) {
          forceBottomFrameRef.current =
            window.requestAnimationFrame(scrollStep);
          return;
        }

        forceBottomFrameRef.current = 0;
        forceStickToBottomRef.current = false;
        updateStickToBottomFromCurrentViewport();
      };

      forceBottomFrameRef.current = window.requestAnimationFrame(scrollStep);
    },
    [scrollMessagesToBottom, updateStickToBottomFromCurrentViewport],
  );

  const scheduleResizeScrollToBottom = useCallback(() => {
    if (!stickToBottomRef.current && !forceStickToBottomRef.current) return;
    if (resizeScrollFrameRef.current) return;

    resizeScrollFrameRef.current = window.requestAnimationFrame(() => {
      resizeScrollFrameRef.current = 0;
      if (!stickToBottomRef.current && !forceStickToBottomRef.current) return;
      if (forceStickToBottomRef.current && scrollAnimationFrameRef.current) {
        startForceScrollToBottom(6);
        return;
      }

      scrollMessagesToBottom("auto");
      if (forceStickToBottomRef.current) {
        startForceScrollToBottom(6);
      }
    });
  }, [scrollMessagesToBottom, startForceScrollToBottom]);

  const stopForceScrollToBottom = useCallback(() => {
    if (forceBottomFrameRef.current) {
      window.cancelAnimationFrame(forceBottomFrameRef.current);
    }
    if (resizeScrollFrameRef.current) {
      window.cancelAnimationFrame(resizeScrollFrameRef.current);
    }
    forceBottomFrameRef.current = 0;
    forceBottomRemainingFramesRef.current = 0;
    resizeScrollFrameRef.current = 0;
    forceStickToBottomRef.current = false;
    stopScrollAnimation();
  }, [stopScrollAnimation]);

  useEffect(() => {
    return () => {
      cancelPendingAssistantDelta();
      stopForceScrollToBottom();
      stopComposerTransition();
    };
  }, [
    cancelPendingAssistantDelta,
    stopComposerTransition,
    stopForceScrollToBottom,
  ]);

  useLayoutEffect(() => {
    const fromRect = pendingComposerTransitionRectRef.current;
    const composerElement = composerFrameRef.current;
    pendingComposerTransitionRectRef.current = null;
    if (!fromRect || !composerElement) return;

    if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) return;

    const toRect = composerElement.getBoundingClientRect();
    const deltaX = fromRect.left - toRect.left;
    const deltaY = fromRect.top - toRect.top;
    if (Math.abs(deltaX) < 0.5 && Math.abs(deltaY) < 0.5) return;

    stopComposerTransition();
    composerElement.style.transition = "none";
    composerElement.style.transform = `translate(${deltaX}px, ${deltaY}px)`;
    composerElement.getBoundingClientRect();

    composerTransitionFrameRef.current = window.requestAnimationFrame(() => {
      composerTransitionFrameRef.current = 0;
      composerElement.style.transition = `transform ${CONVERSATION_OPEN_ANIMATION_MS}ms cubic-bezier(0.22,1,0.36,1)`;
      composerElement.style.transform = "";
      composerTransitionTimeoutRef.current = window.setTimeout(() => {
        composerTransitionTimeoutRef.current = 0;
        composerElement.style.transition = "";
      }, CONVERSATION_OPEN_ANIMATION_MS + 40);
    });
  }, [activeSessionId, showConversationLayout, stopComposerTransition]);

  useLayoutEffect(() => {
    const root = messagesScrollRootRef.current;
    if (!root || !showConversationLayout) {
      messagesViewportRef.current = null;
      return;
    }

    const viewport = root.querySelector<HTMLDivElement>(
      '[data-slot="scroll-area-viewport"]',
    );
    if (!viewport) {
      messagesViewportRef.current = null;
      return;
    }

    messagesViewportRef.current = viewport;

    const handleScroll = () => {
      updateStickToBottom(viewport);
    };

    viewport.addEventListener("scroll", handleScroll, { passive: true });
    updateStickToBottom(viewport);

    return () => {
      viewport.removeEventListener("scroll", handleScroll);
      if (messagesViewportRef.current === viewport) {
        messagesViewportRef.current = null;
      }
    };
  }, [showConversationLayout, updateStickToBottom]);

  useLayoutEffect(() => {
    if (!hasTurns) {
      previousTurnCountRef.current = 0;
      stickToBottomRef.current = true;
      setShowScrollToBottomButton(false);
      return;
    }

    const hasNewTurn = turns.length > previousTurnCountRef.current;
    previousTurnCountRef.current = turns.length;

    if (snapNextMessageScrollRef.current) {
      snapNextMessageScrollRef.current = false;
      stickToBottomRef.current = true;
      animateMessagesToBottom();
      startForceScrollToBottom();
      return;
    }

    if (hasNewTurn) {
      stickToBottomRef.current = true;
      scrollMessagesToBottom("smooth");
      return;
    }

    if (stickToBottomRef.current) {
      scrollMessagesToBottom("auto");
    }
  }, [
    composerHeight,
    hasTurns,
    isPinnedSummaryOpen,
    shouldShiftPinnedSummaryLayout,
    lastTurnStateKey,
    animateMessagesToBottom,
    scrollMessagesToBottom,
    startForceScrollToBottom,
    turns.length,
  ]);

  useLayoutEffect(() => {
    const contentElement = messagesContentRef.current;
    if (!contentElement || !hasTurns) return;

    const resizeObserver = new ResizeObserver(scheduleResizeScrollToBottom);

    resizeObserver.observe(contentElement);

    return () => {
      resizeObserver.disconnect();
    };
  }, [hasTurns, scheduleResizeScrollToBottom]);

  useLayoutEffect(() => {
    if (!hasTurns) return;

    function handleMarkdownContentResize() {
      scheduleResizeScrollToBottom();
    }

    window.addEventListener(
      MARKDOWN_CONTENT_RESIZE_EVENT,
      handleMarkdownContentResize,
    );

    return () => {
      window.removeEventListener(
        MARKDOWN_CONTENT_RESIZE_EVENT,
        handleMarkdownContentResize,
      );
    };
  }, [hasTurns, scheduleResizeScrollToBottom]);

  const startNewConversation = ({
    preservePrompt = false,
  }: { preservePrompt?: boolean } = {}) => {
    saveCurrentToolActivitySessionState();
    sidebarConversationSelectionRef.current += 1;
    snapNextMessageScrollRef.current = false;
    stopForceScrollToBottom();
    setOpeningConversationFromEmpty(false);
    setRevealingHistoryConversation(false);
    if (!preservePrompt) {
      setPrompt("");
    }
    setTurns([]);
    activeSessionIdRef.current = "";
    setActiveSessionId("");
    closedToolActivityItemIdsRef.current = new Set();
    setToolActivityTabs([]);
    setActiveToolActivityTabId("");
    setRightSidebarOpen(false);
  };

  function startNewProjectConversation(projectPath: string) {
    if (!projectPath) return;
    startNewConversation();
    setSelectedProjectPath(projectPath);
  }

  async function hideSidebarProject(projectPath: string) {
    if (!projectPath) return;
    setRecentProjects((currentProjects) =>
      currentProjects.filter((project) => project.rootPath !== projectPath),
    );
    if (!activeSessionIdRef.current && selectedProjectPath === projectPath) {
      setSelectedProjectPath("");
    }
    if (!hasAppBridge()) return;
    try {
      await setProjectSidebarHidden(projectPath, true);
      await refreshRecentProjects();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "移除项目失败");
      await refreshRecentProjects();
    }
  }

  function togglePinnedConversation(sessionId: string) {
    setPinnedConversationIds((currentIds) =>
      currentIds.includes(sessionId)
        ? currentIds.filter((id) => id !== sessionId)
        : [sessionId, ...currentIds],
    );
  }

  async function archiveConversation(sessionId: string) {
    setArchivedConversationIds((currentIds) =>
      currentIds.includes(sessionId) ? currentIds : [sessionId, ...currentIds],
    );
    setPinnedConversationIds((currentIds) =>
      currentIds.filter((id) => id !== sessionId),
    );
    setPendingPermissionCountsBySessionId((currentCounts) => {
      if (!(sessionId in currentCounts)) return currentCounts;
      const nextCounts = { ...currentCounts };
      delete nextCounts[sessionId];
      return nextCounts;
    });
    setConversationRunning(sessionId, false);
    if (sessionId === activeSessionIdRef.current) {
      startNewConversation();
    }

    if (!hasAppBridge()) return;
    try {
      await archiveSession(sessionId);
    } catch {
      setArchivedConversationIds((currentIds) =>
        currentIds.filter((id) => id !== sessionId),
      );
    }
  }

  const emptyComposerBottom = `calc(50% - ${Math.round(
    composerHeight / 2,
  )}px - ${EMPTY_COMPOSER_VERTICAL_OFFSET}px - ${composerExtraHeight}px)`;
  const composerBottom = showConversationLayout ? "1rem" : emptyComposerBottom;
  const composerBottomSm = showConversationLayout
    ? "1.5rem"
    : emptyComposerBottom;
  const emptyComposerTop = "calc(50% - 2.5rem)";

  function closeToolActivityTab(tabId: string) {
    setToolActivityTabs((currentTabs) => {
      const closedTab = currentTabs.find((tab) => tab.id === tabId);
      if (closedTab) {
        for (const key of toolActivityCloseKeys(closedTab)) {
          closedToolActivityItemIdsRef.current.add(key);
        }
      }
      const nextTabs = currentTabs.filter((tab) => tab.id !== tabId);
      setActiveToolActivityTabId((currentId) => {
        if (currentId !== tabId) return currentId;
        return builtinBrowserTabIdsRef.current.at(-1) || nextTabs.at(-1)?.id || "";
      });
      if (nextTabs.length === 0 && builtinBrowserTabIdsRef.current.length === 0) {
        setRightSidebarOpen(false);
      }
      return nextTabs;
    });
  }

  const waitForBuiltinBrowserReady = useCallback((tabId: string) => {
    return new Promise<void>((resolve) => {
      let settled = false;
      const resolvers =
        pendingBuiltinBrowserReadyRef.current.get(tabId) ?? new Set<() => void>();
      const finish = () => {
        if (settled) return;
        settled = true;
        window.clearTimeout(timeout);
        resolvers.delete(finish);
        if (resolvers.size === 0) {
          pendingBuiltinBrowserReadyRef.current.delete(tabId);
        }
        resolve();
      };
      const timeout = window.setTimeout(finish, 1800);
      resolvers.add(finish);
      pendingBuiltinBrowserReadyRef.current.set(tabId, resolvers);
    });
  }, []);

  const handleBuiltinBrowserReady = useCallback((tabId: string) => {
    const resolvers = pendingBuiltinBrowserReadyRef.current.get(tabId);
    if (!resolvers) return;
    for (const resolve of [...resolvers]) {
      resolve();
    }
  }, []);

  const openBuiltinBrowser = useCallback((targetUrl?: string, requestedTabId?: string) => {
    const nextTabId =
      requestedTabId?.trim()
      || (builtinBrowserTabIdsRef.current.length === 0
        ? BUILTIN_BROWSER_TAB_ID
        : `${BUILTIN_BROWSER_TAB_ID}-${Date.now().toString(36)}-${Math.random()
            .toString(36)
            .slice(2, 8)}`);
    const ready = waitForBuiltinBrowserReady(nextTabId);
    if (targetUrl) {
      setBuiltinBrowserInitialUrls((currentInitialUrls) => ({
        ...currentInitialUrls,
        [nextTabId]: targetUrl,
      }));
    }
    setBuiltinBrowserReadyTokens((currentTokens) => ({
      ...currentTokens,
      [nextTabId]: (currentTokens[nextTabId] ?? 0) + 1,
    }));
    setBuiltinBrowserTabIds((currentTabIds) =>
      currentTabIds.includes(nextTabId)
        ? currentTabIds
        : [...currentTabIds, nextTabId],
    );
    setActiveToolActivityTabId(nextTabId);
    setRightSidebarOpen(true);
    if (activeProjectPage !== "chat") {
      void navigate({ to: "/projects/chat" });
    }
    return ready;
  }, [activeProjectPage, navigate, waitForBuiltinBrowserReady]);

  useEffect(() => {
    const unsubscribe = window.aivo?.browser?.onOpenRequest?.((payload) => {
      const tabId = payload?.tabId?.trim() || BUILTIN_BROWSER_TAB_ID;
      return openBuiltinBrowser(payload?.url || undefined, tabId);
    });
    return () => unsubscribe?.();
  }, [openBuiltinBrowser]);

  function closeBuiltinBrowser(tabId = activeToolActivityTabIdRef.current) {
    const browserTabId = builtinBrowserTabIdsRef.current.includes(tabId)
      ? tabId
      : builtinBrowserTabIdsRef.current.at(-1) || "";
    if (!browserTabId) return;
    const browser = window.aivo?.browser;
    if (browser) {
      void browser.close(browserTabId).catch(() => undefined);
    }
    setBuiltinBrowserTabIds((currentTabIds) => {
      const nextTabIds = currentTabIds.filter((id) => id !== browserTabId);
      setBuiltinBrowserInitialUrls((currentInitialUrls) => {
        if (!(browserTabId in currentInitialUrls)) return currentInitialUrls;
        const { [browserTabId]: _closedInitialUrl, ...nextInitialUrls } =
          currentInitialUrls;
        return nextInitialUrls;
      });
      setBuiltinBrowserReadyTokens((currentTokens) => {
        if (!(browserTabId in currentTokens)) return currentTokens;
        const { [browserTabId]: _closedReadyToken, ...nextTokens } =
          currentTokens;
        return nextTokens;
      });
      pendingBuiltinBrowserReadyRef.current.delete(browserTabId);
      setActiveToolActivityTabId((currentId) => {
        if (currentId !== browserTabId) return currentId;
        return nextTabIds.at(-1) || toolActivityTabsRef.current.at(-1)?.id || "";
      });
      if (nextTabIds.length === 0 && toolActivityTabsRef.current.length === 0) {
        setRightSidebarOpen(false);
      }
      return nextTabIds;
    });
  }

  conversationTimelineHandlersRef.current = {
    onDeleteAssistantMessage: (turn) => {
      void deleteConversationAssistantMessage(turn);
    },
    onDeleteTurn: (turn) => {
      void deleteConversationTurn(turn);
    },
    onEditUserMessage: (turn) => {
      void editConversationUserMessage(turn);
    },
    onOpenSession: (sessionId) => {
      void openConversationById(sessionId);
    },
    onRetryTurn: (turn) => {
      void retryConversationTurn(turn);
    },
  };

  const conversationTimelineHandlers = useMemo(
    () => ({
      onDeleteAssistantMessage: (turn: ConversationTurn) =>
        conversationTimelineHandlersRef.current.onDeleteAssistantMessage(turn),
      onDeleteTurn: (turn: ConversationTurn) =>
        conversationTimelineHandlersRef.current.onDeleteTurn(turn),
      onEditUserMessage: (turn: ConversationTurn) =>
        conversationTimelineHandlersRef.current.onEditUserMessage(turn),
      onOpenSession: (sessionId: string) =>
        conversationTimelineHandlersRef.current.onOpenSession(sessionId),
      onRetryTurn: (turn: ConversationTurn) =>
        conversationTimelineHandlersRef.current.onRetryTurn(turn),
    }),
    [],
  );

	  return (
	      <TerminalDockProvider>
	        <ToolActivationDialog
	          activeSessionId={activeSessionId}
	          defaultActiveToolNames={defaultActiveToolNames}
	          onDefaultActiveToolNamesChange={setDefaultActiveToolNames}
	          onOpenChange={setToolActivationDialogOpen}
	          open={toolActivationDialogOpen}
	          usedToolNames={usedToolNamesFromTurns(turns)}
	          workspaceRoot={activeWorkspaceRoot}
	        />
	        <ProjectColumnShell
          bottomPanel={(bottomHeight) => (
            <TerminalPanelContent
              key={activeWorkspaceRoot || "no-workspace"}
              enabled
              height={bottomHeight}
              terminalEnabled={canUseTerminalPanel}
              workspaceRoot={activeWorkspaceRoot}
            />
          )}
          leftSidebar={
            <ConversationSidebar
              activeConversationId={activeSessionId}
              activeProjectPage={activeProjectPage}
              conversations={visibleConversations}
              onNewConversation={() => {
                startNewConversation();
                void navigate({ to: "/projects/chat" });
              }}
              onNewProjectConversation={(projectPath) => {
                startNewProjectConversation(projectPath);
                void navigate({ to: "/projects/chat" });
              }}
              onArchiveConversation={(sessionId) =>
                void archiveConversation(sessionId)
              }
              onHideProject={hideSidebarProject}
              onSelectConversation={(session) => {
                void navigate({ to: "/projects/chat" });
                void selectSidebarConversation(session);
              }}
              onTogglePinnedConversation={togglePinnedConversation}
              pendingPermissionCountsBySessionId={
                pendingPermissionCountsBySessionId
              }
              pinnedConversationIds={pinnedConversationIds}
              projectGroups={projectConversationGroups}
              runningConversationIds={runningConversationIds}
              selectedProjectPath={selectedProjectPath}
              topBar={null}
            />
          }
          topBar={
            ({ leftSidebarState, onToggleLeftSidebar }) => (
              <ProjectTopBar
                canShowTerminalPanel={canUseTerminalPanel}
                conversationTitle={conversationTitle}
                hasConversation={Boolean(activeSessionId)}
                leftSidebarState={leftSidebarState}
                onNewPage={() => {
                  startNewConversation();
                  void navigate({ to: "/projects/chat" });
                }}
                onToggleLeftSidebar={onToggleLeftSidebar}
                pageIcon={
                  activeProjectPage === "plugins" ? (
                    <Plug
                      aria-hidden="true"
                      className="size-4 shrink-0 text-muted-foreground"
                    />
                  ) : undefined
                }
                pageTitle={
                  activeProjectPage === "plugins" ? "Plugins & MCP" : undefined
                }
                showTerminalButton={activeProjectPage === "chat"}
              />
            )
          }
          mainTopBar={
            <ProjectMainTopBar
              conversationTitle={conversationTitle}
              hasConversation={Boolean(activeSessionId)}
              isLayoutPanelOpen={isPinnedSummaryOpen}
              onToggleLayoutPanel={() => {
                setPinnedSummaryOpen((current) => !current);
              }}
              pageIcon={
                activeProjectPage === "plugins" ? (
                  <Plug
                    aria-hidden="true"
                    className="size-4 shrink-0 text-muted-foreground"
                  />
                ) : undefined
              }
              pageTitle={
                activeProjectPage === "plugins" ? "Plugins & MCP" : undefined
              }
              rightOpen={activeProjectPage === "chat" && isRightSidebarOpen}
              showLayoutButton={canToggleEnvironmentSummaryPanel}
              showTerminalButton={activeProjectPage === "chat"}
            />
          }
          main={
            activeProjectPage === "plugins" ? (
              <PluginMcpSettingsContent
                className="bg-background"
                sessionId={activeSessionId}
                workspaceRoot={activeWorkspaceRoot}
              />
            ) : (
              <>
                <div id="conversation-main" className="min-h-0 flex-1">
                  <div
                        className="relative h-full min-h-0 overflow-hidden px-4 sm:px-6"
                        onDragEnter={handleComposerDragEnter}
                        onDragLeave={handleComposerDragLeave}
                        onDragOver={handleComposerDragOver}
                        onDrop={handleComposerDrop}
                        ref={mainRef}
                        style={
                          {
                            "--composer-height": `${composerHeight}px`,
                            "--conversation-bottom-height": `${composerHeight}px`,
                            "--conversation-composer-bottom": composerBottom,
                            "--conversation-composer-bottom-sm": composerBottomSm,
                            "--empty-composer-top": emptyComposerTop,
                          } as React.CSSProperties
                        }
                  >
                        <h1
                          className={cn(
                            "absolute left-1/2 top-[calc(50%-100px)] w-[calc(100%-2rem)] -translate-x-1/2 text-center text-3xl leading-none tracking-normal text-foreground transition-all duration-500 ease-out",
                            showConversationLayout
                              ? "-translate-y-8 opacity-0"
                              : "translate-y-0 opacity-100",
                          )}
                        >
                          我们该做什么？
                        </h1>

                        <div
                          className={cn(
                            "pointer-events-none absolute inset-4 z-40 flex items-center justify-center rounded-2xl border border-dashed border-primary/50 bg-background/75 text-sm font-medium text-foreground shadow-lg shadow-foreground/5 backdrop-blur-sm transition-opacity duration-150 sm:inset-6",
                            isComposerDropActive
                              ? "opacity-100"
                              : "opacity-0",
                          )}
                        >
                          <div className="flex items-center gap-2 rounded-full bg-card px-4 py-2 shadow-sm">
                            <File className="size-4 text-primary" />
                            <span>拖放文件或图片以添加到输入框</span>
                          </div>
                        </div>

                        {showConversationLayout && (
                          <div
                            className="absolute inset-0 z-0"
                            ref={messagesScrollRootRef}
                          >
                            <ScrollArea className="h-full [&_[data-slot=scroll-area-scrollbar]]:mt-2">
                              {hasTurns ? (
                                <SubmittedPromptContent
                                  agentRuns={agentRuns}
                                  contentRef={messagesContentRef}
                                  dockPinnedSummary={
                                    isPinnedSummaryOpen &&
                                    shouldShiftPinnedSummaryLayout
                                  }
                                  onOpenSession={
                                    conversationTimelineHandlers.onOpenSession
                                  }
                                  onDeleteAssistantMessage={
                                    conversationTimelineHandlers.onDeleteAssistantMessage
                                  }
                                  onDeleteTurn={
                                    conversationTimelineHandlers.onDeleteTurn
                                  }
                                  onEditUserMessage={
                                    conversationTimelineHandlers.onEditUserMessage
                                  }
                                  onRetryTurn={
                                    conversationTimelineHandlers.onRetryTurn
                                  }
                                  revealFromHistory={
                                    isRevealingHistoryConversation
                                  }
                                  reservePermissionDock={
                                    hasPendingInteractionRequest
                                  }
                                  turns={turns}
                                />
                              ) : null}
                            </ScrollArea>
                          </div>
                        )}

                        {shouldShowEnvironmentSummaryPanel && (
                          <aside
                            className={cn(
                              "absolute right-4 top-9 hidden min-[1050px]:block sm:right-6",
                              canDockPinnedSummary ? "z-10" : "z-20",
                            )}
	                          >
	                            <EnvironmentSummaryPanel
	                              onOpenTools={() => setToolActivationDialogOpen(true)}
	                            />
	                          </aside>
                        )}

                        {showConversationLayout && (
                          <div className="pointer-events-none absolute bottom-0 left-0 right-0 z-5 h-10 bg-background" />
                        )}

                        {hasPendingPermissionRequest && showConversationLayout ? (
                          <PermissionApprovalDock
                            dockPinnedSummary={
                              isPinnedSummaryOpen &&
                              shouldShiftPinnedSummaryLayout
                            }
                            permissions={pendingPermissionRequests}
                          />
                        ) : hasPendingQuestionRequest && showConversationLayout ? (
                          <QuestionRequestDock
                            dockPinnedSummary={
                              isPinnedSummaryOpen &&
                              shouldShiftPinnedSummaryLayout
                            }
                            request={pendingQuestionRequests[0]}
                          />
                        ) : (
                          <div
                            ref={composerFrameRef}
                            className={cn(
                              "absolute left-1/2 z-30 w-[calc(100%-2rem)] max-w-[680px] -translate-x-1/2 sm:w-[calc(100%-48px)]",
                              showConversationLayout
                                ? "bottom-[var(--conversation-composer-bottom)] will-change-[bottom,transform] sm:bottom-[var(--conversation-composer-bottom-sm)]"
                                : "top-[var(--empty-composer-top)] will-change-[top,transform]",
                              "transition-[bottom,top,transform,margin] duration-[520ms] ease-[cubic-bezier(0.22,1,0.36,1)]",
                              showConversationLayout &&
                                isPinnedSummaryOpen &&
                                shouldShiftPinnedSummaryLayout &&
                                "min-[1050px]:-ml-40",
                            )}
                          >
                            {showConversationLayout &&
                            (showScrollToBottomButton ||
                              shouldShowTodoFloatingStatus) ? (
                              <div className="absolute bottom-[calc(100%+0.75rem)] left-1/2 z-10 flex -translate-x-1/2 flex-col items-center gap-2">
                                {showScrollToBottomButton ? (
                                  <Button
                                    aria-label="滚动到最新消息"
                                    className="rounded-full"
                                    onClick={handleScrollToBottomButtonClick}
                                    size="icon"
                                    type="button"
                                  >
                                    <ArrowDown />
                                  </Button>
                                ) : null}
                                {shouldShowTodoFloatingStatus ? (
                                  <div className="flex items-center gap-1">
                                    <TodoFloatingStatus
                                      todoItems={visibleTodoPlanItems}
                                    />
                                    {isVisibleTodoPlanComplete ? (
                                      <Button
                                        aria-label="隐藏计划列表"
                                        onClick={hideCompletedTodoPlan}
                                        size="icon"
                                        type="button"
                                      >
                                        <X />
                                      </Button>
                                    ) : null}
                                  </div>
                                ) : null}
                              </div>
                            ) : null}
                            {isSubagentSession ? (
                              <SubagentSessionActionBar
                                agentRun={activeSubagentRun}
                                onBack={() =>
                                  void openConversationById(activeParentSessionId)
                                }
                                onCancel={
                                  activeRunningSubagentRun
                                    ? () => void cancelActiveSubagentRun()
                                    : undefined
                                }
                                onHeightChange={handleComposerHeightChange}
                              />
                            ) : (
                              <PromptComposer
                                onHeightChange={handleComposerHeightChange}
                                onPromptChange={setPrompt}
                                onSubmit={
                                  hasPendingTurn
                                    ? stopPendingTurn
                                    : submitPrompt
                                }
                                pending={hasPendingTurn}
                                prompt={prompt}
                                modelId={activeModelId}
                                modelLabel={
                                  formatModelTriggerLabel(
                                    getModelLabel(modelOptions, activeModelId),
                                  ) || "模型"
                                }
                                modelOptions={modelOptions}
                                allModelOptions={allModelOptions}
                                onAddAttachments={addComposerAttachmentFiles}
                                onModelSelect={selectModel}
                                onAgentModeSelect={selectAgentMode}
                                onExtraHeightChange={setComposerExtraHeight}
                                onPermissionModeSelect={selectPermissionMode}
                                onProjectAdd={() => void addComposerProject()}
                                onProjectClear={clearComposerProject}
                                onProjectSelect={selectComposerProject}
                                onReasoningEffortSelect={selectReasoningEffort}
                                onRemoveAttachment={removeComposerAttachment}
                                onServiceTierSelect={selectServiceTier}
                                permissionMode={permissionMode}
                                project={composerProject}
                                projectPath={composerProjectPath}
                                projects={recentProjects}
                                agentMode={agentMode}
                                agentModes={agentModes}
                                attachments={composerAttachments}
                                reasoningEffort={reasoningEffort}
                                serviceTier={serviceTier}
                                showProjectPicker={
                                  !activeSessionId && !showConversationLayout
                                }
                                showServiceTier={Boolean(
                                  activeModelRef &&
                                  providerSupportsServiceTier(
                                    activeModelRef.providerId,
                                  ),
                                )}
                              />
                            )}
                          </div>
                        )}
                        </div>
                      </div>
            </>
            )
          }
          rightOpen={activeProjectPage === "chat" && isRightSidebarOpen}
          onRightOpenChange={setRightSidebarOpen}
          rightSidebar={
            activeProjectPage === "chat" && SHOULD_MOUNT_TOOL_ACTIVITY_SIDEBAR ? (
              <ToolActivitySidebar
                activeTabId={activeToolActivityTabId}
                browserInitialUrls={builtinBrowserInitialUrls}
                browserReadyTokens={builtinBrowserReadyTokens}
                browserTabIds={builtinBrowserTabIds}
                browserVisible={isBrowserRevealReady}
                onActiveTabChange={setActiveToolActivityTabId}
                onApplyFileState={(tab, targetState) => {
                  void applyToolActivityFileState(tab, targetState);
                }}
                onBrowserReady={handleBuiltinBrowserReady}
                onCloseBrowser={closeBuiltinBrowser}
                onCloseTab={closeToolActivityTab}
                onOpenBrowser={openBuiltinBrowser}
                tabs={toolActivityTabs}
              />
            ) : null
          }
          showTerminalButton={activeProjectPage === "chat"}
        />
      </TerminalDockProvider>
  );
}

function ToolActivationDialog({
  activeSessionId,
  defaultActiveToolNames,
  onDefaultActiveToolNamesChange,
  onOpenChange,
  open,
  usedToolNames,
  workspaceRoot,
}: {
  activeSessionId: string;
  defaultActiveToolNames: string[];
  onDefaultActiveToolNamesChange: (
    updater: string[] | ((current: string[]) => string[]),
  ) => void;
  onOpenChange: (open: boolean) => void;
  open: boolean;
  usedToolNames: string[];
  workspaceRoot: string;
}) {
  const [activeToolNames, setActiveToolNames] = useState<string[]>([]);
  const [savedToolNames, setSavedToolNames] = useState<string[]>([]);
  const [tools, setTools] = useState<ToolCatalogEntry[]>([]);
  const [pluginLabels, setPluginLabels] = useState<Record<string, string>>({});
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const defaultActiveToolNamesRef = useRef(defaultActiveToolNames);
  const activeToolSet = useMemo(() => new Set(activeToolNames), [activeToolNames]);
  const usedToolSet = useMemo(() => new Set(usedToolNames), [usedToolNames]);
  const toggleableTools = useMemo(
    () => tools.filter(isToggleableCatalogTool),
    [tools],
  );
  const groupedTools = useMemo(
    () => groupToolCatalogEntries(toggleableTools, pluginLabels),
    [pluginLabels, toggleableTools],
  );
  const inactiveCount = Math.max(0, toggleableTools.length - activeToolNames.length);
  const hasDraftChanges = !toolNameListsEqual(activeToolNames, savedToolNames);

  useEffect(() => {
    defaultActiveToolNamesRef.current = defaultActiveToolNames;
  }, [defaultActiveToolNames]);

  useEffect(() => {
    if (!open) return;
    let cancelled = false;
    setLoading(true);
    void Promise.all([
      listToolCatalog(workspaceRoot),
      activeSessionId
        ? getSessionActiveTools(activeSessionId).catch(() => ({
            sessionId: activeSessionId,
            toolNames: defaultActiveToolNamesRef.current,
          }))
        : Promise.resolve({
            sessionId: "",
            toolNames: defaultActiveToolNamesRef.current,
          }),
      listPlugins(true).catch(() => [] as PluginListItem[]),
    ])
      .then(([catalogTools, activeTools, plugins]) => {
        if (cancelled) return;
        const normalizedActiveTools = normalizeToolNames(activeTools.toolNames);
        setTools(catalogTools);
        setActiveToolNames(normalizedActiveTools);
        setSavedToolNames(normalizedActiveTools);
        setPluginLabels(pluginLabelsById(plugins));
      })
      .catch((err) => {
        if (!cancelled) {
          toast.error(err instanceof Error ? err.message : "加载工具失败");
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [activeSessionId, open, workspaceRoot]);

  async function submitActiveToolNames() {
    const normalized = normalizeToolNames(activeToolNames);
    setSaving(true);
    try {
      if (!activeSessionId) {
        setSavedToolNames(normalized);
        onDefaultActiveToolNamesChange(normalized);
        onOpenChange(false);
        return;
      }
      const saved = await setSessionActiveTools(activeSessionId, normalized);
      const savedNames = normalizeToolNames(saved.toolNames);
      setActiveToolNames(savedNames);
      setSavedToolNames(savedNames);
      onDefaultActiveToolNamesChange(savedNames);
      onOpenChange(false);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "更新工具失败");
    } finally {
      setSaving(false);
    }
  }

  function toggleTool(name: string, enabled: boolean) {
    const next = new Set(activeToolNames);
    if (enabled) {
      next.add(name);
    } else {
      next.delete(name);
    }
    setActiveToolNames(normalizeToolNames([...next]));
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex h-[min(760px,86vh)] max-w-[calc(100%-2rem)] flex-col gap-0 overflow-hidden p-0 sm:max-w-3xl">
        <DialogHeader className="border-b px-5 py-4 pr-12">
          <DialogTitle>工具</DialogTitle>
          <DialogDescription>管理当前对话可用工具和默认激活列表。</DialogDescription>
        </DialogHeader>

        <div className="flex min-h-0 flex-1 flex-col">
          <div className="flex flex-col gap-3 px-5 py-4 sm:flex-row sm:items-center sm:justify-between">
            <div className="flex flex-wrap items-center gap-2">
              <Badge variant="secondary">激活 {activeToolNames.length}</Badge>
              <Badge variant="outline">未激活 {inactiveCount}</Badge>
              <Badge variant="outline">已使用 {usedToolNames.length}</Badge>
            </div>
            <div className="text-xs text-muted-foreground">
              {toggleableTools.length} 个可配置工具
            </div>
          </div>

          <Separator />

          <ScrollArea className="min-h-0 flex-1 px-5 py-4 [&_[data-slot=scroll-area-viewport]]:overflow-x-hidden [&_[data-slot=scroll-area-viewport]>div]:!block [&_[data-slot=scroll-area-viewport]>div]:!min-w-0">
            {loading ? (
              <ToolActivationDialogSkeleton />
            ) : groupedTools.length === 0 ? (
              <Empty className="min-h-48 border">
                <EmptyMedia variant="icon">
                  <Wrench />
                </EmptyMedia>
                <EmptyHeader>
                  <EmptyTitle>没有可激活工具</EmptyTitle>
                  <EmptyDescription>
                    当前插件和运行环境没有提供可手动激活的工具。
                  </EmptyDescription>
                </EmptyHeader>
              </Empty>
            ) : (
              <div className="flex min-w-0 flex-col gap-4">
                {groupedTools.map((group) => (
                  <section key={group.id} className="flex min-w-0 flex-col gap-2">
                    <div className="flex items-center gap-2">
                      <div className="min-w-0 flex-1 truncate text-sm font-medium">
                        {group.label}
                      </div>
                      <Badge variant="outline">{group.tools.length}</Badge>
                    </div>

                    <div className="overflow-hidden rounded-md border">
                      {group.tools.map((tool) => {
                        const active = activeToolSet.has(tool.name);
                        const switchId = toolActivationSwitchId(tool.name);
                        return (
                          <label
                            className="grid w-full min-w-0 cursor-pointer grid-cols-[minmax(0,1fr)_auto] items-center gap-3 overflow-hidden border-b p-3 transition-colors last:border-b-0 hover:bg-muted/50"
                            htmlFor={switchId}
                            key={tool.name}
                          >
                            <div className="min-w-0 flex-1 overflow-hidden">
                              <div className="grid min-w-0 grid-cols-[minmax(0,1fr)_auto] items-center gap-2">
                                <div className="truncate text-sm font-medium">
                                  {tool.name}
                                </div>
                                {usedToolSet.has(tool.name) ? (
                                  <Badge variant="secondary">已使用</Badge>
                                ) : null}
                              </div>
                              {tool.description ? (
                                <div
                                  className="mt-1 truncate text-xs text-muted-foreground"
                                  title={tool.description}
                                >
                                  {tool.description}
                                </div>
                              ) : null}
                            </div>
                            <Switch
                              checked={active}
                              className="shrink-0"
                              disabled={loading || saving}
                              id={switchId}
                              onCheckedChange={(checked) => toggleTool(tool.name, checked)}
                              size="sm"
                            />
                          </label>
                        );
                      })}
                    </div>
                  </section>
                ))}
              </div>
            )}
          </ScrollArea>
        </div>

        <DialogFooter className="border-t px-5 py-4 sm:items-center sm:justify-between">
          <div className="flex items-center gap-2 text-xs text-muted-foreground">
            {loading || saving ? <Spinner /> : null}
            <span>
              {hasDraftChanges
                ? "更改将在提交后用于当前对话和新对话。"
                : "默认激活列表会用于新对话。"}
            </span>
          </div>
          <div className="flex flex-col-reverse gap-2 sm:flex-row">
            <DialogClose asChild>
              <Button disabled={saving} type="button" variant="outline">
                关闭
              </Button>
            </DialogClose>
            <Button
              disabled={loading || saving || !hasDraftChanges}
              onClick={() => void submitActiveToolNames()}
              type="button"
            >
              {saving ? "提交中" : "提交"}
            </Button>
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function ToolActivationDialogSkeleton() {
  return (
    <div className="flex flex-col gap-4">
      {Array.from({ length: 3 }, (_, groupIndex) => (
        <section className="flex flex-col gap-2" key={groupIndex}>
          <div className="flex items-center justify-between gap-2">
            <Skeleton className="h-4 w-36" />
            <Skeleton className="h-5 w-10 rounded-full" />
          </div>
          <div className="overflow-hidden rounded-md border">
            {Array.from({ length: 3 }, (_, rowIndex) => (
              <div
                className="flex items-start gap-3 border-b p-3 last:border-b-0"
                key={rowIndex}
              >
                <Skeleton className="h-[14px] w-6 rounded-full" />
                <div className="flex flex-1 flex-col gap-2">
                  <Skeleton className="h-4 w-44" />
                  <Skeleton className="h-3 w-full" />
                  <Skeleton className="h-3 w-2/3" />
                </div>
              </div>
            ))}
          </div>
        </section>
      ))}
    </div>
  );
}

type ToolCatalogGroup = {
  id: string;
  label: string;
  tools: ToolCatalogEntry[];
};

function isToggleableCatalogTool(tool: ToolCatalogEntry) {
  if (!tool.enabled || isBridgeCatalogTool(tool)) return false;
  if (tool.source === "plugin" || tool.source === "mcp") return true;
  if (
    ["mcp", "plugin", "agent", "automation", "admin", "browser"].includes(
      tool.category ?? "",
    )
  ) {
    return true;
  }
  return (tool.toolsets ?? []).some(
    (toolset) =>
      toolset === "mcp" ||
      toolset === "plugin" ||
      toolset === "admin" ||
      toolset === "browser" ||
      toolset.startsWith("mcp:") ||
      toolset.startsWith("plugin:"),
  );
}

function isBridgeCatalogTool(tool: ToolCatalogEntry) {
  return ["tool_resolve", "tool_search", "tool_list", "tool_detail", "tool_call"].includes(
    tool.name,
  );
}

function groupToolCatalogEntries(
  tools: ToolCatalogEntry[],
  pluginLabels: Record<string, string>,
) {
  const groups = new Map<string, ToolCatalogGroup>();
  for (const tool of tools) {
    const id = `${tool.source}:${tool.sourceId || tool.namespace || tool.category || "other"}`;
    const label =
      (tool.sourceId && pluginLabels[tool.sourceId]) ||
      tool.namespace ||
      tool.sourceId ||
      toolCategoryLabel(tool);
    const group = groups.get(id) ?? { id, label, tools: [] };
    group.tools.push(tool);
    groups.set(id, group);
  }
  return [...groups.values()];
}

function toolCategoryLabel(tool: ToolCatalogEntry) {
  if (tool.source === "mcp") return "MCP";
  if (tool.source === "plugin") return "插件";
  if (tool.category) return tool.category;
  return "工具";
}

function pluginLabelsById(items: PluginListItem[]) {
  return Object.fromEntries(
    items.map((item) => [
      item.plugin.id,
      item.plugin.manifest.displayName ||
        item.plugin.manifest.name ||
        item.plugin.id,
    ]),
  );
}

function normalizeToolNames(names: string[]) {
  return [...new Set(names.map((name) => name.trim()).filter(Boolean))].toSorted();
}

function toolActivationSwitchId(toolName: string) {
  return `tool-activation-${encodeURIComponent(toolName)}`;
}

function toolNameListsEqual(left: string[], right: string[]) {
  const normalizedLeft = normalizeToolNames(left);
  const normalizedRight = normalizeToolNames(right);
  return (
    normalizedLeft.length === normalizedRight.length &&
    normalizedLeft.every((name, index) => name === normalizedRight[index])
  );
}

function usedToolNamesFromTurns(turns: ConversationTurn[]) {
  return normalizeToolNames(
    turns.flatMap((turn) => turn.toolCalls.map((toolCall) => toolCall.name || "")),
  );
}

function ProjectColumnShell({
  bottomPanel,
  leftSidebar,
  main,
  mainTopBar,
  onRightOpenChange,
  rightOpen,
  rightSidebar,
  showTerminalButton,
  topBar,
}: {
  bottomPanel: (height: number) => React.ReactNode;
  leftSidebar: React.ReactNode;
  main: React.ReactNode;
  mainTopBar?: React.ReactNode;
  onRightOpenChange: (open: boolean) => void;
  rightOpen: boolean;
  rightSidebar: React.ReactNode;
  showTerminalButton?: boolean;
  topBar: (controls: {
    leftSidebarState: "expanded" | "collapsed";
    onToggleLeftSidebar: () => void;
  }) => React.ReactNode;
}) {
  const panelLayout = useProjectPreferencesStore((state) => state.panelLayout);
  const setPanelLayout = useProjectPreferencesStore(
    (state) => state.setPanelLayout,
  );
  const rightTopbarActionsWidth =
    rightOpen && showTerminalButton
      ? "120px"
      : showTerminalButton
        ? "84px"
        : rightOpen
          ? "84px"
          : "40px";

  return (
    <SidebarProvider
      className="h-dvh !min-h-0 overflow-hidden bg-background text-foreground"
      style={
        {
          "--project-bottom-panel-height": `${panelLayout.bottomHeight}px`,
          "--project-left-sidebar-width": `${panelLayout.leftWidth}px`,
          "--project-right-sidebar-width": `${panelLayout.rightWidth}px`,
          "--project-right-topbar-actions-width": rightTopbarActionsWidth,
          "--sidebar-width": "var(--project-left-sidebar-width)",
          "--sidebar-width-icon": "52px",
        } as React.CSSProperties
      }
    >
      <ProjectColumnShellContent
        bottomPanel={bottomPanel}
        leftSidebar={leftSidebar}
        main={main}
        mainTopBar={mainTopBar}
        onRightOpenChange={onRightOpenChange}
        panelLayout={panelLayout}
        rightOpen={rightOpen}
        rightSidebar={rightSidebar}
        setPanelLayout={setPanelLayout}
        showTerminalButton={showTerminalButton}
        topBar={topBar}
      />
    </SidebarProvider>
  );
}

function ProjectColumnShellContent({
  bottomPanel,
  leftSidebar,
  main,
  mainTopBar,
  onRightOpenChange,
  panelLayout,
  rightOpen,
  rightSidebar,
  setPanelLayout,
  showTerminalButton,
  topBar,
}: {
  bottomPanel: (height: number) => React.ReactNode;
  leftSidebar: React.ReactNode;
  main: React.ReactNode;
  mainTopBar?: React.ReactNode;
  onRightOpenChange: (open: boolean) => void;
  panelLayout: ProjectPanelLayout;
  rightOpen: boolean;
  rightSidebar: React.ReactNode;
  setPanelLayout: (layout: ProjectPanelLayout) => void;
  showTerminalButton?: boolean;
  topBar: (controls: {
    leftSidebarState: "expanded" | "collapsed";
    onToggleLeftSidebar: () => void;
  }) => React.ReactNode;
}) {
  const { state: leftSidebarState, toggleSidebar: toggleLeftSidebar } =
    useSidebar();
  const rootRef = useRef<HTMLDivElement | null>(null);
  const panelLayoutRef = useRef(panelLayout);
  const [isRightSidebarMaximized, setRightSidebarMaximized] = useState(false);
  const rightTopbarActionsWidth =
    rightOpen && showTerminalButton
      ? "120px"
      : showTerminalButton
        ? "84px"
        : rightOpen
          ? "84px"
          : "40px";

  useEffect(() => {
    panelLayoutRef.current = panelLayout;
  }, [panelLayout]);

  const commitResizedPanelLayout = useCallback(
    (nextLayout: ProjectPanelLayout) => {
      panelLayoutRef.current = nextLayout;
      setPanelLayout(nextLayout);
    },
    [setPanelLayout],
  );

  const updatePanelLayoutVariable = useCallback(
    (key: keyof ProjectPanelLayout, value: number) => {
      const root = rootRef.current;
      if (!root) return;
      const cssName =
        key === "leftWidth"
          ? "--project-left-sidebar-width"
          : key === "rightWidth"
            ? "--project-right-sidebar-width"
            : "--project-bottom-panel-height";
      root.style.setProperty(cssName, `${Math.round(value)}px`);
    },
    [],
  );

  const applyRightSidebarRuntimeWidth = useCallback(
    (maximized: boolean) => {
      const root = rootRef.current;
      if (!root) return;
      const nextWidth = maximized
        ? getRightSidebarMaximizedWidth(root)
        : panelLayoutRef.current.rightWidth;
      updatePanelLayoutVariable("rightWidth", nextWidth);
    },
    [updatePanelLayoutVariable],
  );

  const toggleRightSidebarMaximized = useCallback(() => {
    setRightSidebarMaximized((current) => {
      const next = !current;
      applyRightSidebarRuntimeWidth(next);
      return next;
    });
  }, [applyRightSidebarRuntimeWidth]);

  useLayoutEffect(() => {
    if (!rightOpen && isRightSidebarMaximized) {
      setRightSidebarMaximized(false);
      updatePanelLayoutVariable("rightWidth", panelLayout.rightWidth);
      return;
    }

    applyRightSidebarRuntimeWidth(rightOpen && isRightSidebarMaximized);
  }, [
    applyRightSidebarRuntimeWidth,
    isRightSidebarMaximized,
    panelLayout.leftWidth,
    panelLayout.rightWidth,
    rightOpen,
    updatePanelLayoutVariable,
  ]);

  useEffect(() => {
    if (!rightOpen || !isRightSidebarMaximized) return;

    function handleResize() {
      applyRightSidebarRuntimeWidth(true);
    }

    window.addEventListener("resize", handleResize);
    return () => window.removeEventListener("resize", handleResize);
  }, [applyRightSidebarRuntimeWidth, isRightSidebarMaximized, rightOpen]);

  return (
    <div
      className="relative flex h-full min-h-0 min-w-0 flex-1 overflow-hidden [&_[data-slot=sidebar-container]]:[transition-duration:var(--project-panel-transition-duration,200ms)] [&_[data-slot=sidebar-gap]]:[transition-duration:var(--project-panel-transition-duration,200ms)]"
      ref={rootRef}
      style={
        {
          "--project-bottom-panel-height": `${panelLayout.bottomHeight}px`,
          "--project-left-sidebar-width": `${panelLayout.leftWidth}px`,
          "--project-right-sidebar-width": `${panelLayout.rightWidth}px`,
          "--project-right-topbar-actions-width": rightTopbarActionsWidth,
          "--sidebar-width": "var(--project-left-sidebar-width)",
        } as React.CSSProperties
      }
    >
      <Sidebar className="z-40" collapsible="offcanvas">
        {leftSidebar}
      </Sidebar>
      {leftSidebarState === "expanded" ? (
        <ProjectResizeHandle
          ariaLabel="调整左侧栏宽度"
          className="absolute inset-y-0 z-40 -ml-px hidden md:flex"
          orientation="vertical"
          onResizeStart={(event) => {
            const root = rootRef.current;
            if (!root) return;
            startProjectPanelResize({
              commitResizedPanelLayout,
              event,
              key: "leftWidth",
              leftOpen: true,
              panelLayout: panelLayoutRef.current,
              rightOpen,
              root,
              updatePanelLayoutVariable,
            });
          }}
          style={{ left: "var(--project-left-sidebar-width)" }}
        />
      ) : null}
      <SidebarProvider
        className="!contents"
        onOpenChange={onRightOpenChange}
        open={rightOpen}
        style={
          {
            "--sidebar-width": "var(--project-right-sidebar-width)",
          } as React.CSSProperties
        }
      >
        <div className="pointer-events-none absolute inset-x-0 top-0 z-50 h-9">
          <div className="pointer-events-none h-full">
            {topBar({
              leftSidebarState,
              onToggleLeftSidebar: toggleLeftSidebar,
            })}
          </div>
        </div>
        <ProjectFloatingRightTopBarActions
          isRightSidebarMaximized={isRightSidebarMaximized}
          onToggleRightSidebarMaximized={toggleRightSidebarMaximized}
          rightOpen={rightOpen}
          showTerminalButton={showTerminalButton}
        />
        <ProjectColumnWorkspace
          bottomPanel={bottomPanel}
          leftSidebarState={leftSidebarState}
          main={main}
          mainTopBar={mainTopBar}
          onResizeStart={(event, key) => {
            const root = rootRef.current;
            if (!root) return;
            startProjectPanelResize({
              commitResizedPanelLayout,
              event,
              key,
              leftOpen: leftSidebarState === "expanded",
              panelLayout: panelLayoutRef.current,
              rightOpen,
              root,
              updatePanelLayoutVariable,
            });
          }}
          panelLayout={panelLayout}
          rightSidebarMaximized={isRightSidebarMaximized}
          rightSidebar={rightSidebar}
        />
      </SidebarProvider>
    </div>
  );
}

function ProjectColumnWorkspace({
  bottomPanel,
  leftSidebarState,
  main,
  mainTopBar,
  onResizeStart,
  panelLayout,
  rightSidebarMaximized,
  rightSidebar,
}: {
  bottomPanel: (height: number) => React.ReactNode;
  leftSidebarState: "expanded" | "collapsed";
  main: React.ReactNode;
  mainTopBar?: React.ReactNode;
  onResizeStart: (
    event: React.PointerEvent<HTMLButtonElement>,
    key: "bottomHeight" | "rightWidth",
  ) => void;
  panelLayout: ProjectPanelLayout;
  rightSidebarMaximized: boolean;
  rightSidebar: React.ReactNode;
}) {
  const { state: bottomPanelState } = useTerminalDock();
  const { state: rightSidebarState } = useSidebar();
  const bottomOpen = bottomPanelState === "expanded";
  const rightOpen = rightSidebarState === "expanded";
  const isMac = window.aivo?.platform === "darwin";
  const leftCompactWidth = isMac ? 202 : 148;
  const mainTopBarLeftOffset =
    leftSidebarState === "collapsed"
      ? `calc(${leftCompactWidth}px - var(--sidebar-width-icon, 52px))`
      : undefined;

  return (
    <SidebarInset className="h-full min-h-0 min-w-0 overflow-hidden">
      <div className="flex h-full min-h-0 flex-col overflow-hidden">
        <div
          className="relative flex min-h-0 flex-1 overflow-hidden"
          data-project-workspace-content
        >
          <main className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
            {mainTopBar ? (
              <div
                className="z-50 h-9 shrink-0 transition-[margin-left] duration-[var(--project-panel-transition-duration,200ms)] ease-linear"
                style={{ marginLeft: mainTopBarLeftOffset }}
              >
                {mainTopBar}
              </div>
            ) : null}
            <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
              {main}
            </div>
          </main>
          {rightOpen && !rightSidebarMaximized ? (
            <ProjectResizeHandle
              ariaLabel="调整右侧栏宽度"
              className="absolute inset-y-0 z-40 -mr-px hidden md:flex"
              orientation="vertical"
              onResizeStart={(event) => onResizeStart(event, "rightWidth")}
              style={{ right: "var(--project-right-sidebar-width)" }}
            />
          ) : null}
          <Sidebar
            className="!absolute !inset-y-0 !z-[70] !h-auto bg-background"
            collapsible="offcanvas"
            side="right"
            style={
              {
                "--sidebar-width": "var(--project-right-sidebar-width)",
              } as React.CSSProperties
            }
          >
            {rightSidebar}
          </Sidebar>
        </div>
        {bottomOpen ? (
          <ProjectResizeHandle
            ariaLabel="调整底部栏高度"
            orientation="horizontal"
            onResizeStart={(event) => onResizeStart(event, "bottomHeight")}
          />
        ) : null}
        <TerminalDockPanel height="var(--project-bottom-panel-height)">
          {bottomPanel(panelLayout.bottomHeight)}
        </TerminalDockPanel>
      </div>
    </SidebarInset>
  );
}

function ProjectResizeHandle({
  ariaLabel,
  className,
  onResizeStart,
  orientation,
  style,
}: {
  ariaLabel: string;
  className?: string;
  onResizeStart: (event: React.PointerEvent<HTMLButtonElement>) => void;
  orientation: "horizontal" | "vertical";
  style?: React.CSSProperties;
}) {
  return (
    <button
      aria-label={ariaLabel}
      className={cn(
        "group/resize relative shrink-0 bg-transparent outline-none ring-offset-background transition-colors focus-visible:ring-0",
        orientation === "vertical"
          ? "flex h-full w-px cursor-col-resize items-center justify-center after:absolute after:inset-y-0 after:left-1/2 after:w-2 after:-translate-x-1/2"
          : "flex h-px w-full cursor-row-resize items-center justify-center after:absolute after:left-0 after:top-1/2 after:h-2 after:w-full after:-translate-y-1/2",
        className,
      )}
      onPointerDown={onResizeStart}
      style={style}
      type="button"
    />
  );
}

function getRightSidebarMaximizedWidth(root: HTMLDivElement) {
  const workspace = root.querySelector<HTMLElement>(
    "[data-project-workspace-content]",
  );
  const rect = (workspace ?? root).getBoundingClientRect();
  return Math.max(PROJECT_RIGHT_SIDEBAR_MIN_WIDTH, Math.round(rect.width));
}

function startProjectPanelResize({
  commitResizedPanelLayout,
  event,
  key,
  leftOpen,
  panelLayout,
  rightOpen,
  root,
  updatePanelLayoutVariable,
}: {
  commitResizedPanelLayout: (layout: ProjectPanelLayout) => void;
  event: React.PointerEvent<HTMLButtonElement>;
  key: keyof ProjectPanelLayout;
  leftOpen: boolean;
  panelLayout: ProjectPanelLayout;
  rightOpen: boolean;
  root: HTMLDivElement;
  updatePanelLayoutVariable: (
    key: keyof ProjectPanelLayout,
    value: number,
  ) => void;
}) {
  event.preventDefault();
  event.currentTarget.setPointerCapture(event.pointerId);

  const rect = root.getBoundingClientRect();
  const startX = event.clientX;
  const startY = event.clientY;
  const startLayout = panelLayout;
  const leftVisible = leftOpen ? startLayout.leftWidth : 0;
  const rightVisible = rightOpen ? startLayout.rightWidth : 0;
  let latestValue = startLayout[key];
  let latestClientX = startX;
  let latestClientY = startY;
  let hasMoved = false;
  let frame = 0;
  const previousCursor = document.body.style.cursor;
  const previousUserSelect = document.body.style.userSelect;

  function clampLayoutValue(clientX: number, clientY: number) {
    if (key === "leftWidth") {
      const maxWidth = rect.width - rightVisible - PROJECT_MAIN_MIN_WIDTH;
      return clampNumber(
        clientX - rect.left,
        PROJECT_LEFT_SIDEBAR_MIN_WIDTH,
        maxWidth,
      );
    }

    if (key === "rightWidth") {
      const maxWidth = rect.width - leftVisible - PROJECT_MAIN_MIN_WIDTH;
      return clampNumber(
        rect.right - clientX,
        PROJECT_RIGHT_SIDEBAR_MIN_WIDTH,
        maxWidth,
      );
    }

    return clampNumber(
      rect.bottom - clientY,
      PROJECT_BOTTOM_PANEL_MIN_HEIGHT,
      rect.height - PROJECT_UPPER_MIN_HEIGHT,
    );
  }

  function scheduleUpdate() {
    if (frame) return;
    frame = window.requestAnimationFrame(() => {
      frame = 0;
      latestValue = clampLayoutValue(latestClientX, latestClientY);
      updatePanelLayoutVariable(key, latestValue);
    });
  }

  function handlePointerMove(moveEvent: PointerEvent) {
    latestClientX = moveEvent.clientX;
    latestClientY = moveEvent.clientY;
    hasMoved =
      hasMoved ||
      Math.abs(latestClientX - startX) > 1 ||
      Math.abs(latestClientY - startY) > 1;
    scheduleUpdate();
  }

  function handlePointerUp() {
    if (frame) {
      window.cancelAnimationFrame(frame);
      frame = 0;
    }
    latestValue = clampLayoutValue(latestClientX, latestClientY);
    updatePanelLayoutVariable(key, latestValue);
    if (hasMoved) {
      commitResizedPanelLayout({
        ...startLayout,
        [key]: Math.round(latestValue),
      });
    }
    document.body.style.cursor = previousCursor;
    document.body.style.userSelect = previousUserSelect;
    root.style.removeProperty("--project-panel-transition-duration");
    window.removeEventListener("pointermove", handlePointerMove);
    window.removeEventListener("pointerup", handlePointerUp);
    window.removeEventListener("pointercancel", handlePointerUp);
  }

  document.body.style.cursor =
    key === "bottomHeight" ? "row-resize" : "col-resize";
  document.body.style.userSelect = "none";
  root.style.setProperty("--project-panel-transition-duration", "0ms");
  window.addEventListener("pointermove", handlePointerMove);
  window.addEventListener("pointerup", handlePointerUp);
  window.addEventListener("pointercancel", handlePointerUp);
}

function TodoFloatingStatus({ todoItems }: { todoItems: TodoItem[] }) {
  const todos = todoItems;
  if (todos.length === 0) return null;

  const runningIndex = todos.findIndex((todo) => todo.status === "in_progress");
  const openIndex = todos.findIndex((todo) => !isTodoDone(todo.status));
  const currentIndex =
    runningIndex >= 0
      ? runningIndex
      : openIndex >= 0
        ? openIndex
        : todos.length - 1;
  const currentTodo = todos[currentIndex] ?? todos[0];

  return (
    <HoverCard openDelay={100}>
      <HoverCardTrigger asChild>
        <Button type="button">
          {todoStatusIcon(currentTodo.status)}
          <span>
            第 {currentIndex + 1} / {todos.length} 步
          </span>
        </Button>
      </HoverCardTrigger>
      <HoverCardContent side="top">
        <div className="flex flex-col gap-2">
          <div className="font-medium">计划列表</div>
          <Separator />
          <div className="flex flex-col gap-2">
            {todos.map((todo) => (
              <div className="flex min-w-0 items-start gap-2" key={todo.id}>
                {todoStatusIcon(todo.status)}
                <div className="min-w-0 flex-1">
                  <div className="truncate font-medium">
                    {todo.title || "待办"}
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      </HoverCardContent>
    </HoverCard>
  );
}

function QuestionRequestDock({
  dockPinnedSummary,
  request,
}: {
  dockPinnedSummary: boolean;
  request?: QuestionRequest;
}) {
  const [answers, setAnswers] = useState<string[][]>(() =>
    (request?.questions ?? []).map((_, index) =>
      request?.answers?.[index] ? [...request.answers[index]] : [],
    ),
  );
  const [customAnswers, setCustomAnswers] = useState<string[]>(() =>
    initialQuestionCustomAnswers(request),
  );
  const [currentQuestionIndex, setCurrentQuestionIndex] = useState(0);
  const [activeAnswerIndex, setActiveAnswerIndex] = useState(0);
  const [busy, setBusy] = useState<"idle" | "submitting" | "rejecting">("idle");
  const customInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    setAnswers(
      (request?.questions ?? []).map((_, index) =>
        request?.answers?.[index] ? [...request.answers[index]] : [],
      ),
    );
    setCustomAnswers(initialQuestionCustomAnswers(request));
    setCurrentQuestionIndex(0);
    setActiveAnswerIndex(0);
    setBusy("idle");
  }, [request]);

  useEffect(() => {
    if (!request?.id) return;
    const onKeyDown = (event: KeyboardEvent) => {
      const target = event.target;
      const isTextInput =
        target instanceof HTMLInputElement ||
        target instanceof HTMLTextAreaElement ||
        (target instanceof HTMLElement && target.isContentEditable);

      if (event.key === "Escape") {
        event.preventDefault();
        void reject();
        return;
      }

      if (event.key === "ArrowDown" || event.key === "ArrowUp") {
        event.preventDefault();
        if (isTextInput && target instanceof HTMLElement) target.blur();
        setActiveAnswerIndex((current) => {
          const direction = event.key === "ArrowDown" ? 1 : -1;
          return (current + direction + answerItemCount) % answerItemCount;
        });
        return;
      }

      if (isTextInput) {
        if (event.key === "Enter") {
          event.preventDefault();
          void handlePrimaryAction();
        }
        return;
      }

      if (event.key === "Enter") {
        event.preventDefault();
        if (question.multiple) {
          void handlePrimaryAction();
        } else {
          selectActiveAnswer();
        }
        return;
      }

      if (event.key === " " && question.multiple) {
        event.preventDefault();
        selectActiveAnswer();
      }
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  });

  if (!request || request.questions.length === 0) return null;

  const currentRequest = request;
  const isBusy = busy !== "idle";
  const questionCount = request.questions.length;
  const questionIndex = Math.min(currentQuestionIndex, questionCount - 1);
  const question = request.questions[questionIndex]!;
  const selected = answers[questionIndex] ?? [];
  const optionCount = question.options?.length ?? 0;
  const customAnswerIndex = optionCount;
  const answerItemCount = optionCount + 1;
  const activeIndex = Math.min(activeAnswerIndex, customAnswerIndex);
  const hasNextQuestion = questionIndex < questionCount - 1;
  const primaryActionContinues = hasNextQuestion;

  function setVisibleQuestionIndex(index: number) {
    setCurrentQuestionIndex(index);
    setActiveAnswerIndex(0);
  }

  function setQuestionAnswer(index: number, value: string, multiple: boolean) {
    setCustomAnswers((current) => {
      if (!current[index]) return current;
      const next = [...current];
      next[index] = "";
      return next;
    });
    setAnswers((current) => {
      const next = [...current];
      const existing = next[index] ?? [];
      if (multiple) {
        next[index] = existing.includes(value)
          ? existing.filter((item) => item !== value)
          : [...existing, value];
      } else {
        next[index] = [value];
      }
      return next;
    });
    if (!multiple && index < questionCount - 1) {
      setVisibleQuestionIndex(index + 1);
    }
  }

  function selectActiveAnswer() {
    if (isBusy) return;
    const activeOption = question.options?.[activeIndex];
    if (activeOption) {
      setQuestionAnswer(
        questionIndex,
        activeOption.label,
        Boolean(question.multiple),
      );
      return;
    }
    customInputRef.current?.focus();
  }

  function setCustomAnswer(index: number, value: string) {
    setCustomAnswers((current) => {
      const next = [...current];
      next[index] = value;
      return next;
    });
    const trimmed = value.trim();
    setAnswers((current) => {
      const next = [...current];
      next[index] = trimmed ? [trimmed] : [];
      return next;
    });
  }

  async function submit() {
    if (isBusy) return;
    setBusy("submitting");
    try {
      await replyQuestionRequest(
        currentRequest.id,
        currentRequest.questions.map((_, index) => answers[index] ?? []),
      );
    } catch (err) {
      setBusy("idle");
      toast.error(err instanceof Error ? err.message : String(err));
    }
  }

  async function handlePrimaryAction() {
    if (isBusy) return;
    if (primaryActionContinues) {
      setVisibleQuestionIndex(questionIndex + 1);
      return;
    }
    await submit();
  }

  async function reject() {
    if (isBusy) return;
    setBusy("rejecting");
    try {
      await rejectQuestionRequest(currentRequest.id, "Dismissed by user");
    } catch (err) {
      setBusy("idle");
      toast.error(err instanceof Error ? err.message : String(err));
    }
  }

  return (
    <div
      className={cn(
        "absolute bottom-4 left-1/2 z-30 w-[calc(100%-2rem)] max-w-[960px] -translate-x-1/2 transition-[margin,transform] duration-500 ease-[cubic-bezier(0.22,1,0.36,1)] sm:bottom-6 sm:w-[calc(100%-48px)]",
        dockPinnedSummary && "min-[1050px]:-ml-40",
      )}
      data-assistant-hover-ignore="true"
    >
      <Card className="gap-2 py-2 [--card-spacing:--spacing(2.5)]" size="sm">
        <CardHeader className="gap-0.5">
          <CardTitle>{question.question || "需要你的选择"}</CardTitle>
          {questionCount > 1 ? (
            <CardAction className="flex items-center gap-2">
              <Button
                aria-label="上一个问题"
                disabled={isBusy || questionIndex === 0}
                onClick={() => setVisibleQuestionIndex(Math.max(0, questionIndex - 1))}
                size="icon-sm"
                type="button"
                variant="ghost"
              >
                <ArrowLeft />
              </Button>
              <CardDescription>
                {questionIndex + 1} / {questionCount}
              </CardDescription>
              <Button
                aria-label="下一个问题"
                disabled={isBusy || questionIndex >= questionCount - 1}
                onClick={() =>
                  setVisibleQuestionIndex(Math.min(questionCount - 1, questionIndex + 1))
                }
                size="icon-sm"
                type="button"
                variant="ghost"
              >
                <ArrowRight />
              </Button>
            </CardAction>
          ) : null}
          {question.header ? <CardDescription>{question.header}</CardDescription> : null}
        </CardHeader>
        <CardContent>
          <ScrollArea className="max-h-[min(58vh,460px)]">
            <ItemGroup className="!gap-1" data-size="xs">
              {(question.options ?? []).map((option, optionIndex) => {
                const picked = selected.includes(option.label);
                const active = activeIndex === optionIndex;
                return (
                  <Item
                    asChild
                    key={`${option.label}:${optionIndex}`}
                    variant={active ? "muted" : "default"}
                  >
                    <button
                      disabled={isBusy}
                      onMouseEnter={() => setActiveAnswerIndex(optionIndex)}
                      onClick={() =>
                        setQuestionAnswer(
                          questionIndex,
                          option.label,
                          Boolean(question.multiple),
                        )
                      }
                      type="button"
                    >
                      <ItemMedia
                        className={cn(
                          "size-5 !translate-y-0 rounded-full border text-xs font-medium leading-none !self-center",
                          picked
                            ? "border-primary bg-primary text-primary-foreground"
                            : "border-border text-muted-foreground",
                        )}
                      >
                        {optionIndex + 1}
                      </ItemMedia>
                      <ItemContent>
                        <ItemTitle>{option.label}</ItemTitle>
                        {option.description ? (
                          <ItemDescription>{option.description}</ItemDescription>
                        ) : null}
                      </ItemContent>
                    </button>
                  </Item>
                );
              })}
              <Item
                asChild
                variant={activeIndex === customAnswerIndex ? "muted" : "default"}
              >
                <label onMouseEnter={() => setActiveAnswerIndex(customAnswerIndex)}>
                  <ItemMedia
                    className={cn(
                      "size-5 !translate-y-0 !self-center",
                      customAnswers[questionIndex]
                        ? "text-primary"
                        : "border-border text-muted-foreground",
                    )}
                    variant="icon"
                  >
                    <Pencil />
                  </ItemMedia>
                  <ItemContent>
                    <Input
                      ref={customInputRef}
                      disabled={isBusy}
                      onFocus={() => setActiveAnswerIndex(customAnswerIndex)}
                      onChange={(event) =>
                        setCustomAnswer(questionIndex, event.target.value)
                      }
                      placeholder="否，请告知 Aivo 如何调整"
                      value={customAnswers[questionIndex] ?? ""}
                    />
                  </ItemContent>
                </label>
              </Item>
            </ItemGroup>
          </ScrollArea>
        </CardContent>
        <CardFooter className="justify-end gap-2">
          <Button
            disabled={isBusy}
            onClick={() => void reject()}
            type="button"
            variant="ghost"
          >
            忽略
            <Kbd>ESC</Kbd>
          </Button>
          <Button
            disabled={isBusy}
            onClick={() => void handlePrimaryAction()}
            type="button"
          >
            {primaryActionContinues
              ? "继续"
              : busy === "submitting"
                ? "提交中"
                : "提交"}
            {primaryActionContinues ? <ArrowRight /> : <CornerDownLeft />}
          </Button>
        </CardFooter>
      </Card>
    </div>
  );
}

function initialQuestionCustomAnswers(request?: QuestionRequest) {
  return (request?.questions ?? []).map((question, index) => {
    const optionLabels = new Set((question.options ?? []).map((option) => option.label));
    return (
      request?.answers?.[index]?.find((answer) => !optionLabels.has(answer)) ??
      ""
    );
  });
}

function PermissionApprovalDock({
  dockPinnedSummary,
  permissions,
}: {
  dockPinnedSummary: boolean;
  permissions: PermissionRequest[];
}) {
  if (permissions.length === 0) return null;

  return (
    <div
      className={cn(
        "absolute bottom-4 left-1/2 z-20 w-[calc(100%-2rem)] max-w-[760px] -translate-x-1/2 transition-[margin,transform] duration-500 ease-[cubic-bezier(0.22,1,0.36,1)] sm:bottom-6 sm:w-[calc(100%-48px)]",
        dockPinnedSummary && "min-[1050px]:-ml-40",
      )}
      data-assistant-hover-ignore="true"
    >
      <div className="overflow-hidden rounded-2xl border border-border/80 bg-popover text-popover-foreground shadow-2xl shadow-foreground/15 ring-1 ring-foreground/5">
        <div className="flex items-center gap-3 border-b border-border/70 px-4 py-3">
          <span className="flex size-8 shrink-0 items-center justify-center rounded-full bg-primary text-primary-foreground">
            <ShieldCheck className="size-4" />
          </span>
          <div className="min-w-0 flex-1">
            <div className="truncate text-sm font-semibold">等待权限审批</div>
            <div className="truncate text-xs text-muted-foreground">
              审批后任务会继续执行；拒绝会停止这次工具调用。
            </div>
          </div>
          {permissions.length > 1 ? (
            <span className="rounded-full bg-primary/10 px-2 py-1 text-xs  text-primary">
              {permissions.length} 项
            </span>
          ) : null}
        </div>
        <div className="max-h-[min(54vh,420px)] overflow-auto">
          <div className="flex flex-col">
            {permissions.map((permission, index) => (
              <PermissionRequestCard
                compact={permissions.length > 1}
                index={index}
                key={permission.id}
                permission={permission}
              />
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}

function PermissionRequestCard({
  compact,
  index,
  permission,
}: {
  compact: boolean;
  index: number;
  permission: PermissionRequest;
}) {
  const [action, setAction] = useState<PermissionActionState>("idle");
  const [remember, setRemember] = useState(false);
  const files = permissionFiles(permission);
  const command = permissionCommand(permission);
  const title = command
    ? permission.toolName === "run_tests"
      ? "批准测试命令"
      : "批准命令执行"
    : writePermissionToolNames.has(permission.toolName)
      ? "批准文件修改"
      : `批准 ${permission.toolName}`;
  const target = command
    ? command.command
    : permission.paths?.length
      ? permission.paths.join(", ")
      : permission.action;
  const permissionModeLabel = permissionAgentMode(permission);
  const permissionToolsetLabel = permissionToolsets(permission).join(", ");
  const isBusy = action === "approving" || action === "denying";

  async function approve() {
    if (action === "approving" || action === "approved") return;
    setAction("approving");
    try {
      await approvePermissionRequest(permission.id, remember);
      setAction("approved");
    } catch {
      setAction("idle");
    }
  }

  async function deny() {
    if (action === "denying" || action === "denied") return;
    setAction("denying");
    try {
      await denyPermissionRequest(permission.id, remember, "Denied by user");
      setAction("denied");
    } catch {
      setAction("idle");
    }
  }

  return (
    <section className="border-b border-border/70 p-4 text-popover-foreground last:border-b-0">
      <div className="flex min-w-0 items-start gap-3">
        <span className="mt-0.5 flex size-6 shrink-0 items-center justify-center rounded-full bg-primary/10 text-xs font-semibold text-primary">
          {index + 1}
        </span>
        <div className="min-w-0 flex-1">
          <div className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1">
            <h3 className="min-w-0 truncate text-sm font-semibold">{title}</h3>
            <span className="rounded-full bg-muted px-2 py-0.5 text-[11px]  text-muted-foreground">
              {permission.toolName}
            </span>
          </div>
          <div className="mt-1 min-w-0 truncate text-xs text-muted-foreground">
            {target}
          </div>
          <div className="mt-1 min-w-0 truncate text-[11px] text-muted-foreground">
            mode: {permissionModeLabel || "assistant"}
            {permissionToolsetLabel
              ? ` · toolsets: ${permissionToolsetLabel}`
              : ""}
          </div>
        </div>
      </div>
      {files.length > 0 ? (
        <div
          className={cn(
            "mt-3 grid gap-1.5 text-xs",
            compact && files.length > 2 && "max-h-20 overflow-auto pr-1",
          )}
        >
          {files.map((file) => (
            <div
              className="flex min-w-0 items-center justify-between gap-3 rounded-md bg-muted/70 px-2 py-1.5"
              key={`${file.type}:${file.path}:${file.movePath}`}
            >
              <span className="min-w-0 truncate font-mono text-[11px]">
                <span className="mr-2 inline-flex min-w-4 justify-center rounded bg-background px-1 font-sans font-semibold text-muted-foreground">
                  {file.typeLabel}
                </span>
                {file.movePath ? `${file.path} -> ${file.movePath}` : file.path}
              </span>
              <span className="shrink-0 font-mono text-[11px] text-muted-foreground">
                +{file.additions} -{file.deletions}
              </span>
            </div>
          ))}
        </div>
      ) : null}
      {command ? (
        <div className="mt-3 grid gap-2 rounded-md bg-muted/70 p-3 text-xs">
          <div className="min-w-0">
            <div className="mb-1 text-[11px] font-semibold text-muted-foreground">
              命令
            </div>
            <pre className="max-h-24 overflow-auto whitespace-pre-wrap break-words font-mono text-[11px] leading-relaxed text-foreground">
              {command.command}
            </pre>
          </div>
          <div className="grid gap-1 text-[11px] text-muted-foreground sm:grid-cols-2">
            <span className="min-w-0 truncate">cwd: {command.cwd || "."}</span>
            <span className="min-w-0 truncate">
              risk: {command.riskLevel || "unknown"}
            </span>
            <span className="min-w-0 truncate">
              category: {command.category || "unknown"}
            </span>
            <span className="min-w-0 truncate">
              network: {command.networkHint || "deny"}
            </span>
          </div>
        </div>
      ) : null}
      <div className="mt-3 flex flex-col gap-2 border-t border-border/70 pt-3 sm:flex-row sm:items-center sm:justify-between">
        <label className="flex min-w-0 items-center gap-2 text-xs text-muted-foreground">
          <input
            checked={remember}
            className="size-3.5 accent-primary"
            onChange={(event) => setRemember(event.target.checked)}
            type="checkbox"
          />
          <span className="truncate">
            {command ? "记住此命令和 cwd" : "记住这类权限"}
          </span>
        </label>
        <div className="flex shrink-0 items-center gap-2">
          <Button
            className="h-8 px-3 text-xs"
            disabled={action !== "idle"}
            onClick={deny}
            size="sm"
            type="button"
            variant="outline"
          >
            {action === "denying"
              ? "拒绝中"
              : action === "denied"
                ? "已拒绝"
                : "拒绝"}
          </Button>
          <Button
            className="h-8 gap-1.5 px-3 text-xs"
            disabled={action !== "idle"}
            onClick={approve}
            size="sm"
            type="button"
          >
            {isBusy ? null : <Check className="size-3.5" />}
            {action === "approving"
              ? "批准中"
              : action === "approved"
                ? "已批准"
                : "批准并继续"}
          </Button>
        </div>
      </div>
    </section>
  );
}

type PermissionCommandInfo = {
  command: string;
  cwd?: string;
  riskLevel?: string;
  category?: string;
  networkHint?: string;
};

function permissionCommand(
  permission: PermissionRequest,
): PermissionCommandInfo | null {
  const args = permission.arguments;
  if (!args) return null;
  const command =
    typeof args.command === "string"
      ? args.command
      : typeof args.normalizedCommand === "string"
        ? args.normalizedCommand
        : "";
  if (!command.trim()) return null;
  return {
    command,
    cwd: typeof args.cwd === "string" ? args.cwd : undefined,
    riskLevel: typeof args.riskLevel === "string" ? args.riskLevel : undefined,
    category: typeof args.category === "string" ? args.category : undefined,
    networkHint:
      typeof args.networkHint === "string" ? args.networkHint : undefined,
  };
}

function permissionAgentMode(permission: PermissionRequest) {
  const mode = permission.arguments?.agentMode;
  return typeof mode === "string" ? mode : "";
}

function permissionToolsets(permission: PermissionRequest) {
  const toolsets = permission.arguments?.toolsets;
  if (!Array.isArray(toolsets)) return [];
  return toolsets.filter(
    (toolset): toolset is string => typeof toolset === "string",
  );
}

type PermissionFileInfo = {
  path: string;
  movePath?: string;
  type: string;
  typeLabel: string;
  additions: number;
  deletions: number;
  baseHash?: string;
  currentHash?: string;
  stale?: boolean;
};

function permissionFiles(permission: PermissionRequest): PermissionFileInfo[] {
  const rawFiles = permission.arguments?.files;
  if (!Array.isArray(rawFiles)) return [];
  return rawFiles.flatMap((file) => {
    if (!file || typeof file !== "object") return [];
    const record = file as Record<string, unknown>;
    const path = typeof record.path === "string" ? record.path : "";
    if (!path) return [];
    const type = typeof record.type === "string" ? record.type : "update";
    return [
      {
        path,
        movePath:
          typeof record.movePath === "string" ? record.movePath : undefined,
        type,
        typeLabel: permissionFileTypeLabel(type),
        additions: typeof record.additions === "number" ? record.additions : 0,
        deletions: typeof record.deletions === "number" ? record.deletions : 0,
        baseHash:
          typeof record.baseHash === "string" ? record.baseHash : undefined,
        currentHash:
          typeof record.currentHash === "string"
            ? record.currentHash
            : undefined,
        stale: record.stale === true,
      },
    ];
  });
}

function permissionFileTypeLabel(type: string) {
  switch (type) {
    case "add":
      return "A";
    case "delete":
      return "D";
    case "move":
      return "R";
    default:
      return "M";
  }
}

function parseTime(value?: string) {
  if (!value) return new Date();
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? new Date() : date;
}

function getTodoPlanKey(todoItems: TodoItem[]) {
  return todoItems
    .map((todo) =>
      [todo.id, todo.title, todo.status, todo.timeUpdated].join("\u0001"),
    )
    .join("\u0002");
}

function isTodoDone(status: string) {
  return status === "done" || status === "completed";
}

function todoStatusIcon(status: string) {
  const iconClassName = "size-4";
  const icon = isTodoDone(status) ? (
      <Check className={iconClassName} />
  ) : status === "in_progress" ? (
      <Spinner className={iconClassName} />
  ) : (
      <Circle className={iconClassName} />
  );
  return (
    <span className="flex size-5 shrink-0 items-center justify-center">
      {icon}
    </span>
  );
}

function toolActivityTabIsClosed(
  tab: ToolActivityTab,
  closedItemIds: Set<string>,
) {
  return toolActivityCloseKeys(tab).some((key) => closedItemIds.has(key));
}

function toolActivityCloseKeys(tab: ToolActivityTab) {
  if (tab.kind === "command") {
    const toolCallIds = [
      tab.toolCallId,
      ...tab.entries.map((entry) => entry.toolCallId),
    ].filter(Boolean);
    return [...new Set(toolCallIds.map(toolActivityToolCallKey))];
  }
  return [toolActivityTabKey(tab.id)];
}

function toolActivityToolCallKey(toolCallId: string) {
  return `tool-call:${toolCallId}`;
}

function toolActivityTabKey(tabId: string) {
  return `tab:${tabId}`;
}

function relativeTime(value?: string) {
  const date = parseTime(value);
  const elapsedSeconds = Math.max(
    0,
    Math.floor((Date.now() - date.getTime()) / 1000),
  );
  if (elapsedSeconds < 60) return "刚刚";
  const elapsedMinutes = Math.floor(elapsedSeconds / 60);
  if (elapsedMinutes < 60) return `${elapsedMinutes} 分`;
  const elapsedHours = Math.floor(elapsedMinutes / 60);
  if (elapsedHours < 24) return `${elapsedHours} 小时`;
  const elapsedDays = Math.floor(elapsedHours / 24);
  if (elapsedDays < 7) return `${elapsedDays} 天`;
  return `${Math.floor(elapsedDays / 7)} 周`;
}

function hasRunningTurn(turns: ConversationTurn[]) {
  return turns.some((turn) => !turn.responseCompletedAt && !turn.stopped);
}

function turnsFromEvents(
  events: domain.SessionEvent[],
  toolCalls: domain.ToolCall[] = [],
  runtimeTurns: domain.Turn[] = [],
): ConversationTurn[] {
  const messageEvents = events.filter(isConversationMessageEvent);
  const runtimeTurnByUserEventId = new Map(
    runtimeTurns
      .filter((turn) => turn.userEventId)
      .map((turn) => [turn.userEventId, turn]),
  );
  const runtimeTurnById = new Map(runtimeTurns.map((turn) => [turn.id, turn]));
  const turns: ConversationTurn[] = [];
  let current: ConversationTurn | null = null;
  const toolCallsByTurnId = groupToolCallsByTurnId(toolCalls);
  const getTurnToolCalls = (turnId?: string) =>
    turnId ? toolCallsByTurnId.get(turnId) ?? [] : [];

  for (const event of messageEvents) {
    if (event.role === "user" || event.type === "user_message") {
      if (current) {
        turns.push(finalizeSupersededOpenTurn(current, runtimeTurnById));
      }
      const submittedAt = parseTime(event.timeCreated);
      const runtimeTurn = runtimeTurnByUserEventId.get(event.id);
      const currentToolCalls = getTurnToolCalls(runtimeTurn?.id);
      const attachments = conversationAttachmentsFromEvent(event);
      current = {
        activityVisible: currentToolCalls.length > 0,
        assistantPreambles: [],
        attachments,
        id: event.id,
        prompt:
          attachments.length > 0
            ? stripSessionAttachmentSummary(event.content ?? "")
            : event.content ?? "",
        preToolText: "",
        responseCompletedAt: null,
        responseText: "",
        responseVisible: false,
        startedAt: submittedAt.getTime(),
        stopped: false,
        submittedAt,
        thinkingSeconds: 0,
        toolCalls: currentToolCalls,
        turnId: runtimeTurn?.id,
        userEventId: event.id,
      };
      continue;
    }

    if (!current) continue;
    if (event.type === "error") {
      const completedAt = parseTime(event.timeCreated);
      current = {
        ...current,
        activityVisible:
          current.activityVisible ||
          getTurnToolCalls(event.turnId).length > 0,
        responseCompletedAt: completedAt,
        responseText: event.content ?? "请求失败。",
        responseVisible: true,
        thinkingSeconds: Math.max(
          0,
          Math.floor(
            (completedAt.getTime() - current.submittedAt.getTime()) / 1000,
          ),
        ),
        toolCalls: getTurnToolCalls(event.turnId),
        turnId: event.turnId,
      };
      turns.push(current);
      current = null;
      continue;
    }
    if (isBeforeToolAssistantEvent(event)) {
      current = {
        ...current,
        activityVisible: true,
        assistantPreambles: appendAssistantPreamblePart(
          current.assistantPreambles,
          {
            id: event.id,
            text: event.content ?? "",
            timeCreated: event.timeCreated,
          },
        ),
        preToolText: appendAssistantText(
          current.preToolText,
          event.content ?? "",
        ),
        toolCalls: getTurnToolCalls(event.turnId),
        turnId: event.turnId,
      };
      continue;
    }
    const completedAt = parseTime(event.timeCreated);
    current = {
      ...current,
      activityVisible:
        current.activityVisible ||
        getTurnToolCalls(event.turnId).length > 0,
      responseCompletedAt: completedAt,
      responseText: appendAssistantText(
        current.responseText,
        event.content ?? "",
      ),
      responseVisible: true,
      thinkingSeconds: Math.max(
        0,
        Math.floor(
          (completedAt.getTime() - current.submittedAt.getTime()) / 1000,
        ),
      ),
      toolCalls: getTurnToolCalls(event.turnId),
      turnId: event.turnId,
      assistantEventId: event.id,
    };
    turns.push(current);
    current = null;
  }

  if (current)
    turns.push(finalizeOpenTurnFromRuntime(current, runtimeTurnById));
  return attachSystemNotesToTurns(turns, events);
}

function conversationAttachmentsFromEvent(
  event: domain.SessionEvent,
): ConversationUserAttachment[] {
  const payload = recordFromUnknown(event.payload);
  const rawAttachments = Array.isArray(payload?.attachments)
    ? payload.attachments
    : [];
  const attachments: ConversationUserAttachment[] = [];
  for (const rawAttachment of rawAttachments) {
    const attachment = recordFromUnknown(rawAttachment);
    const name = stringFromUnknown(attachment?.name);
    const mimeType = stringFromUnknown(attachment?.mimeType);
    const kind = stringFromUnknown(attachment?.kind);
    if (!name || !mimeType || (kind !== "image" && kind !== "file")) {
      continue;
    }
    const data = stringFromUnknown(attachment?.data);
    attachments.push({
      id: stringFromUnknown(attachment?.id) || crypto.randomUUID(),
      name,
      mimeType,
      kind,
      previewUrl:
        kind === "image" && data ? `data:${mimeType};base64,${data}` : undefined,
      size: numberFromUnknown(attachment?.size),
    });
  }
  return attachments;
}

function attachSystemNotesToTurns(
  turns: ConversationTurn[],
  events: domain.SessionEvent[],
): ConversationTurn[] {
  const notesByTurnId = new Map<string, domain.SessionEvent[]>();
  for (const event of events) {
    if (
      (event.visibility && event.visibility !== "normal") ||
      event.type !== "system_note" ||
      !event.turnId ||
      !event.content?.trim()
    ) {
      continue;
    }
    const notes = notesByTurnId.get(event.turnId) ?? [];
    notes.push(event);
    notesByTurnId.set(event.turnId, notes);
  }
  if (notesByTurnId.size === 0) return turns;
  return turns.map((turn) => {
    if (!turn.turnId) return turn;
    const notes = notesByTurnId.get(turn.turnId);
    if (!notes?.length) return turn;
    return {
      ...turn,
      systemNotes: notes.map((note) => ({
        id: note.id,
        content: note.content ?? "",
        timeCreated: note.timeCreated,
      })),
    };
  });
}

function finalizeSupersededOpenTurn(
  turn: ConversationTurn,
  runtimeTurnById: Map<string, domain.Turn>,
) {
  const finalizedTurn = finalizeOpenTurnFromRuntime(turn, runtimeTurnById);
  if (finalizedTurn !== turn) return finalizedTurn;

  return {
    ...turn,
    stopped: true,
    thinkingSeconds: getTurnElapsedSeconds(turn),
  };
}

function finalizeOpenTurnFromRuntime(
  turn: ConversationTurn,
  runtimeTurnById: Map<string, domain.Turn>,
) {
  const runtimeTurn = turn.turnId
    ? runtimeTurnById.get(turn.turnId)
    : undefined;
  if (!runtimeTurn || runtimeTurn.status === "running") return turn;

  const completedAt = parseTime(
    runtimeTurn.timeCompleted || runtimeTurn.timeUpdated,
  );
  const thinkingSeconds = Math.max(
    0,
    Math.floor((completedAt.getTime() - turn.submittedAt.getTime()) / 1000),
  );

  if (runtimeTurn.status === "cancelled") {
    return {
      ...turn,
      stopped: true,
      thinkingSeconds,
    };
  }

  if (runtimeTurn.status === "failed") {
    return {
      ...turn,
      responseCompletedAt: completedAt,
      responseText: runtimeTurn.error || "请求失败。",
      responseVisible: true,
      thinkingSeconds,
    };
  }

  if (runtimeTurn.status === "completed") {
    return {
      ...turn,
      responseCompletedAt: completedAt,
      responseVisible: true,
      thinkingSeconds,
    };
  }

  return turn;
}

function isBeforeToolAssistantEvent(event: domain.SessionEvent) {
  return (
    event.type === "assistant_message" &&
    event.payload?.["phase"] === "before_tool"
  );
}

function appendAssistantText(current: string, next: string) {
  const trimmedNext = next.trim();
  if (!trimmedNext) return current;
  if (!current.trim()) return next;
  return `${current.trimEnd()}\n\n${trimmedNext}`;
}

function appendAssistantPreamblePart(
  current: ConversationAssistantTextPart[] | undefined,
  next: ConversationAssistantTextPart,
) {
  const trimmedText = next.text.trim();
  if (!trimmedText) return current ?? [];
  const parts = current ?? [];
  const existingIndex = parts.findIndex((part) => part.id === next.id);
  if (existingIndex >= 0) {
    const existing = parts[existingIndex];
    const updated = {
      ...existing,
      text: next.text,
      timeCreated: next.timeCreated ?? existing.timeCreated,
    };
    if (
      existing.text === updated.text &&
      existing.timeCreated === updated.timeCreated
    ) {
      return parts;
    }
    return parts.map((part, index) =>
      index === existingIndex ? updated : part,
    );
  }
  return [...parts, next];
}

function toolCallsForTurn(toolCalls: domain.ToolCall[], turnId?: string) {
  if (!turnId) return [];
  return toolCalls
    .filter((toolCall) => toolCall.turnId === turnId)
    .toSorted((a, b) => {
      const timeDelta =
        parseTime(a.timeCreated).getTime() - parseTime(b.timeCreated).getTime();
      if (timeDelta !== 0) return timeDelta;
      return a.id.localeCompare(b.id);
    });
}

function groupToolCallsByTurnId(toolCalls: domain.ToolCall[]) {
  const callsByTurnId = new Map<string, domain.ToolCall[]>();
  for (const toolCall of toolCalls) {
    if (!toolCall.turnId) continue;
    const calls = callsByTurnId.get(toolCall.turnId) ?? [];
    calls.push(toolCall);
    callsByTurnId.set(toolCall.turnId, calls);
  }
  for (const [turnId, calls] of callsByTurnId) {
    callsByTurnId.set(
      turnId,
      calls.toSorted((a, b) => {
        const timeDelta =
          parseTime(a.timeCreated).getTime() -
          parseTime(b.timeCreated).getTime();
        if (timeDelta !== 0) return timeDelta;
        return a.id.localeCompare(b.id);
      }),
    );
  }
  return callsByTurnId;
}

function mergeSingleToolCall(
  turns: ConversationTurn[],
  toolCall: domain.ToolCall,
) {
  if (!toolCall.id) return turns;
  let changed = false;
  const lastRunningTurnIndex = turns.findLastIndex(
    (turn) => !turn.stopped && !turn.responseCompletedAt,
  );
  const nextTurns = turns.map((turn, index) => {
    if (toolCall.turnId) {
      if (turn.turnId !== toolCall.turnId) return turn;
    } else if (index !== lastRunningTurnIndex) {
      return turn;
    }
    const nextToolCalls = mergeToolCallLists(turn.toolCalls, [toolCall]);
    const nextActivityVisible =
      turn.activityVisible || nextToolCalls.length > 0;
    if (
      sameToolCalls(turn.toolCalls, nextToolCalls) &&
      turn.activityVisible === nextActivityVisible
    ) {
      return turn;
    }
    changed = true;
    return {
      ...turn,
      activityVisible: nextActivityVisible,
      toolCalls: nextToolCalls,
      turnId: turn.turnId || toolCall.turnId,
    };
  });
  return changed ? nextTurns : turns;
}

function moveOpenResponseTextToAssistantPreambleBeforeTool(
  turns: ConversationTurn[],
  toolCall: domain.ToolCall,
) {
  let changed = false;
  const lastRunningTurnIndex = turns.findLastIndex(
    (turn) => !turn.stopped && !turn.responseCompletedAt,
  );
  const nextTurns = turns.map((turn, index) => {
    if (turn.stopped || turn.responseCompletedAt || !turn.responseText.trim()) {
      return turn;
    }
    if (toolCall.turnId) {
      if (turn.turnId !== toolCall.turnId) return turn;
    } else if (index !== lastRunningTurnIndex) {
      return turn;
    }
    changed = true;
    return {
      ...turn,
      activityVisible: true,
      assistantPreambles: appendAssistantPreamblePart(
        turn.assistantPreambles,
        {
          id: `live-preamble:${toolCall.id}`,
          text: turn.responseText,
          timeCreated: toolCall.timeCreated || new Date().toISOString(),
        },
      ),
      preToolText: appendAssistantText(turn.preToolText, turn.responseText),
      responseText: "",
      responseVisible: false,
      turnId: turn.turnId || toolCall.turnId,
    };
  });
  return changed ? nextTurns : turns;
}

function mergeRuntimeTurn(turns: ConversationTurn[], runtimeTurn: domain.Turn) {
  if (!runtimeTurn.id) return turns;
  let changed = false;
  const nextTurns = turns.map((turn) => {
    if (turn.turnId !== runtimeTurn.id) return turn;
    const finalized = finalizeOpenTurnFromRuntime(
      {
        ...turn,
        turnId: runtimeTurn.id,
      },
      new Map([[runtimeTurn.id, runtimeTurn]]),
    );
    if (finalized === turn) return turn;
    changed = true;
    return finalized;
  });
  return changed ? nextTurns : turns;
}

function updatePermissionPauseState(
  turns: ConversationTurn[],
  permissions: PermissionRequest[],
  now: number,
) {
  let changed = false;
  const pendingPermissions = permissions.filter(
    (permission) => permission.status === "pending",
  );

  const nextTurns = turns.map((turn) => {
    if (turn.stopped || turn.responseCompletedAt) {
      if (!turn.pauseStartedAt) return turn;
      changed = true;
      return {
        ...turn,
        pauseStartedAt: null,
        pausedMilliseconds:
          (turn.pausedMilliseconds ?? 0) +
          Math.max(0, now - turn.pauseStartedAt),
      };
    }

    const shouldPause = pendingPermissions.some((permission) =>
      permissionMatchesTurn(permission, turn),
    );
    if (shouldPause) {
      if (turn.pauseStartedAt && turn.activityVisible) return turn;
      changed = true;
      return {
        ...turn,
        activityVisible: true,
        pauseStartedAt: turn.pauseStartedAt ?? now,
        thinkingSeconds: getTurnElapsedSeconds(turn, now),
      };
    }

    if (!turn.pauseStartedAt) return turn;
    changed = true;
    const pausedMilliseconds =
      (turn.pausedMilliseconds ?? 0) + Math.max(0, now - turn.pauseStartedAt);
    return {
      ...turn,
      pausedMilliseconds,
      pauseStartedAt: null,
      thinkingSeconds: getTurnElapsedSeconds(
        {
          pausedMilliseconds,
          pauseStartedAt: null,
          startedAt: turn.startedAt,
        },
        now,
      ),
    };
  });

  return changed ? nextTurns : turns;
}

function mergePendingPermissionToolCalls(
  turns: ConversationTurn[],
  permissions: PermissionRequest[],
) {
  const pendingPermissions = permissions.filter(
    (permission) => permission.status === "pending",
  );
  if (pendingPermissions.length === 0) return turns;

  let changed = false;
  const nextTurns = turns.map((turn, index) => {
    if (turn.stopped || turn.responseCompletedAt) return turn;
    const matchingPermissions = pendingPermissions.filter((permission) =>
      permissionMatchesTurnOrLastRunning(permission, turn, index, turns),
    );
    if (matchingPermissions.length === 0) return turn;

    const missingPermissionToolCalls = matchingPermissions
      .filter((permission) => {
        const toolCallId =
          permission.toolCallId || `permission:${permission.id}`;
        return !turn.toolCalls.some((toolCall) => toolCall.id === toolCallId);
      })
      .map(permissionToolCall);

    if (missingPermissionToolCalls.length === 0 && turn.activityVisible) {
      return turn;
    }

    changed = true;
    return {
      ...turn,
      activityVisible: true,
      toolCalls:
        missingPermissionToolCalls.length === 0
          ? turn.toolCalls
          : mergeToolCallLists(turn.toolCalls, missingPermissionToolCalls),
    };
  });

  return changed ? nextTurns : turns;
}

function permissionMatchesTurnOrLastRunning(
  permission: PermissionRequest,
  turn: ConversationTurn,
  index: number,
  turns: ConversationTurn[],
) {
  if (permissionMatchesTurn(permission, turn)) return true;
  if (permission.turnId) return false;
  return (
    index ===
    turns.findLastIndex(
      (candidate) => !candidate.stopped && !candidate.responseCompletedAt,
    )
  );
}

function permissionToolCall(permission: PermissionRequest): domain.ToolCall {
  const now =
    permission.timeUpdated ||
    permission.timeCreated ||
    new Date().toISOString();
  return {
    id: permission.toolCallId || `permission:${permission.id}`,
    sessionId: permission.sessionId || "",
    turnId: permission.turnId || "",
    name: permission.toolName,
    arguments: permission.arguments,
    status: "pending_approval",
    resultSummary: "等待权限审批",
    result: { pendingApprovalId: permission.id },
    timeCreated: permission.timeCreated || now,
    timeUpdated: now,
  } as domain.ToolCall;
}

function permissionMatchesTurn(
  permission: PermissionRequest,
  turn: ConversationTurn,
) {
  if (permission.turnId) return permission.turnId === turn.turnId;
  return Boolean(turn.turnId);
}

function isConversationMessageEvent(event: domain.SessionEvent) {
  if (event.visibility && event.visibility !== "normal") return false;
  return (
    event.type === "user_message" ||
    event.type === "assistant_message" ||
    event.type === "error"
  );
}

function applyPendingTurnMetadata(
  turns: ConversationTurn[],
  options: { pendingTurnId?: string; pendingStartedAt?: number },
) {
  if (!options.pendingTurnId || !options.pendingStartedAt || turns.length === 0)
    return turns;
  const pendingTurnId = options.pendingTurnId;
  const pendingStartedAt = options.pendingStartedAt;
  const lastIndex = turns.length - 1;
  return turns.map((turn, index) => {
    if (index !== lastIndex || !turn.responseVisible) return turn;
    return {
      ...turn,
      id: turn.id || pendingTurnId,
      startedAt: pendingStartedAt,
      thinkingSeconds: getTurnElapsedSeconds({
        pausedMilliseconds: turn.pausedMilliseconds,
        pauseStartedAt: turn.pauseStartedAt,
        startedAt: pendingStartedAt,
      }),
    };
  });
}

function mergeTurnPauseMetadata(
  nextTurns: ConversationTurn[],
  currentTurns: ConversationTurn[],
) {
  if (nextTurns.length === 0 || currentTurns.length === 0) return nextTurns;
  const currentByKey = new Map<string, ConversationTurn>();
  for (const turn of currentTurns) {
    for (const key of turnIdentityKeys(turn)) {
      currentByKey.set(key, turn);
    }
  }

  const now = Date.now();
  let changed = false;
  const mergedTurns = nextTurns.map((turn, index) => {
    const current =
      turnIdentityKeys(turn)
        .map((key) => currentByKey.get(key))
        .find(Boolean) ??
      fallbackCurrentTurnForMerge(turn, index, currentTurns);
    if (!current) return turn;

    let nextTurn = turn;

    if (current.toolCalls.length > 0) {
      const mergedToolCalls = mergeToolCallLists(
        current.toolCalls,
        turn.toolCalls,
      );
      if (!sameToolCalls(turn.toolCalls, mergedToolCalls)) {
        changed = true;
        nextTurn = {
          ...nextTurn,
          activityVisible: true,
          toolCalls: mergedToolCalls,
          turnId: nextTurn.turnId || current.turnId,
        };
      }
    }

    if (!nextTurn.preToolText.trim() && current.preToolText.trim()) {
      changed = true;
      nextTurn = {
        ...nextTurn,
        activityVisible: true,
        preToolText: current.preToolText,
      };
    }

    if (
      (nextTurn.attachments?.length ?? 0) === 0 &&
      (current.attachments?.length ?? 0) > 0
    ) {
      changed = true;
      nextTurn = {
        ...nextTurn,
        attachments: current.attachments,
        prompt: stripSessionAttachmentSummary(nextTurn.prompt),
      };
    }

    if (
      (nextTurn.assistantPreambles?.length ?? 0) === 0 &&
      (current.assistantPreambles?.length ?? 0) > 0
    ) {
      changed = true;
      nextTurn = {
        ...nextTurn,
        activityVisible: true,
        assistantPreambles: current.assistantPreambles,
      };
    }

    if (current.activityVisible && !nextTurn.activityVisible) {
      changed = true;
      nextTurn = {
        ...nextTurn,
        activityVisible: true,
      };
    }

    const pausedMilliseconds =
      (current.pausedMilliseconds ?? 0) +
      (current.pauseStartedAt ? Math.max(0, now - current.pauseStartedAt) : 0);
    if (pausedMilliseconds <= 0) return nextTurn;

    changed = true;
    return {
      ...nextTurn,
      pausedMilliseconds,
      pauseStartedAt: null,
      thinkingSeconds: Math.max(
        0,
        nextTurn.thinkingSeconds - Math.floor(pausedMilliseconds / 1000),
      ),
    };
  });

  return changed ? mergedTurns : nextTurns;
}

function mergeToolCallLists(
  currentToolCalls: domain.ToolCall[],
  nextToolCalls: domain.ToolCall[],
) {
  if (currentToolCalls.length === 0) return dedupeDelegateToolCalls(nextToolCalls);
  if (nextToolCalls.length === 0) return currentToolCalls;

  const callsById = new Map<string, domain.ToolCall>();
  for (const toolCall of currentToolCalls) {
    callsById.set(toolCall.id, toolCall);
  }
  for (const toolCall of nextToolCalls) {
    callsById.set(toolCall.id, toolCall);
  }

  return dedupeDelegateToolCalls([...callsById.values()]).toSorted((a, b) => {
    const timeDelta =
      parseTime(a.timeCreated).getTime() - parseTime(b.timeCreated).getTime();
    if (timeDelta !== 0) return timeDelta;
    return a.id.localeCompare(b.id);
  });
}

function mergePreservedTurnAttachments(
  pendingTurnId: string | undefined,
  currentTurns: ConversationTurn[],
) {
  if (!pendingTurnId) return undefined;
  return currentTurns.find((turn) => turn.id === pendingTurnId)?.attachments;
}

function stripSessionAttachmentSummary(text: string) {
  return text.replace(/\n{2,}附件:\n(?:- .+(?:\n|$))+$/u, "").trimEnd();
}

function dedupeDelegateToolCalls(toolCalls: domain.ToolCall[]) {
  const output: domain.ToolCall[] = [];
  const indexByKey = new Map<string, number>();

  for (const toolCall of toolCalls) {
    const key = delegateToolCallIdentityKey(toolCall);
    if (!key) {
      output.push(toolCall);
      continue;
    }

    const existingIndex = indexByKey.get(key);
    if (existingIndex === undefined) {
      indexByKey.set(key, output.length);
      output.push(toolCall);
      continue;
    }

    output[existingIndex] = preferredDelegateToolCall(
      output[existingIndex],
      toolCall,
    );
  }

  return output;
}

function delegateToolCallIdentityKey(toolCall: domain.ToolCall) {
  if (!isDelegateTaskToolName(toolCall.name)) return "";
  const run = delegateToolCallStructuredRun(toolCall);
  const runId = stringFromUnknown(run?.id);
  const sessionId = stringFromUnknown(run?.sessionId);
  const metadata = recordFromUnknown(run?.metadata);
  const toolCallId = stringFromUnknown(metadata?.toolCallId);
  if (sessionId) return `delegate:session:${sessionId}`;
  if (runId) return `delegate:run:${runId}`;
  if (toolCallId) return `delegate:tool-call:${toolCallId}`;
  return "";
}

function isDelegateTaskToolName(name: string) {
  return name === "agent_delegate_task";
}

function delegateToolCallStructuredRun(toolCall: domain.ToolCall) {
  const structured = recordFromUnknown(toolCall.result?.structured);
  return recordFromUnknown(structured?.result);
}

function preferredDelegateToolCall(
  current: domain.ToolCall,
  next: domain.ToolCall,
) {
  if (!delegateToolCallStructuredRun(current) && delegateToolCallStructuredRun(next)) {
    return next;
  }
  if (current.status === "running" && next.status !== "running") return next;
  if (!current.result && next.result) return next;
  if (parseTime(next.timeUpdated).getTime() > parseTime(current.timeUpdated).getTime()) {
    return next;
  }
  return current;
}

function recordFromUnknown(value: unknown): Record<string, unknown> | undefined {
  if (!value || typeof value !== "object" || Array.isArray(value)) return undefined;
  return value as Record<string, unknown>;
}

function stringFromUnknown(value: unknown) {
  return typeof value === "string" ? value.trim() : "";
}

function numberFromUnknown(value: unknown) {
  return typeof value === "number" && Number.isFinite(value)
    ? value
    : undefined;
}

function fallbackCurrentTurnForMerge(
  turn: ConversationTurn,
  index: number,
  currentTurns: ConversationTurn[],
) {
  const currentAtIndex = currentTurns[index];
  if (currentAtIndex && turnsCanShareLiveMetadata(turn, currentAtIndex)) {
    return currentAtIndex;
  }

  return currentTurns.find((currentTurn) =>
    turnsCanShareLiveMetadata(turn, currentTurn),
  );
}

function turnsCanShareLiveMetadata(
  nextTurn: ConversationTurn,
  currentTurn: ConversationTurn,
) {
  if (currentTurn.stopped || currentTurn.responseCompletedAt) return false;
  if (nextTurn.stopped || nextTurn.responseCompletedAt) return false;
  return true;
}

function turnIdentityKeys(turn: ConversationTurn) {
  return [
    turn.turnId && `turn:${turn.turnId}`,
    turn.userEventId && `user-event:${turn.userEventId}`,
    turn.id && `id:${turn.id}`,
  ].filter((key): key is string => Boolean(key));
}

function samePermissionRequests(
  a: PermissionRequest[],
  b: PermissionRequest[],
) {
  if (a.length !== b.length) return false;
  return a.every((permission, index) => {
    const other = b[index];
    return (
      other &&
      permission.id === other.id &&
      permission.status === other.status &&
      permission.timeUpdated === other.timeUpdated &&
      permission.turnId === other.turnId &&
      permission.toolCallId === other.toolCallId &&
      permission.toolName === other.toolName &&
      permission.action === other.action
    );
  });
}

function upsertPermissionRequest(
  requests: PermissionRequest[],
  request: PermissionRequest,
) {
  const existingIndex = requests.findIndex((item) => item.id === request.id);
  if (existingIndex === -1) return [request, ...requests];
  const next = requests.slice();
  next[existingIndex] = request;
  return next;
}

function sameQuestionRequests(a: QuestionRequest[], b: QuestionRequest[]) {
  if (a.length !== b.length) return false;
  return a.every((request, index) => {
    const other = b[index];
    return (
      other &&
      request.id === other.id &&
      request.status === other.status &&
      request.timeUpdated === other.timeUpdated &&
      request.turnId === other.turnId &&
      request.toolCallId === other.toolCallId &&
      request.questions.length === other.questions.length
    );
  });
}

function upsertQuestionRequest(
  requests: QuestionRequest[],
  request: QuestionRequest,
) {
  const existingIndex = requests.findIndex((item) => item.id === request.id);
  if (existingIndex === -1) return [request, ...requests];
  const next = requests.slice();
  next[existingIndex] = request;
  return next;
}

function upsertSession(sessions: domain.Session[], session: domain.Session) {
  const existingIndex = sessions.findIndex((item) => item.id === session.id);
  if (existingIndex === -1) {
    return [session, ...sessions];
  }
  const next = sessions.slice();
  next[existingIndex] = session;
  return next;
}

function upsertRecentProject(
  projects: domain.AssistantProject[],
  project: domain.AssistantProject,
) {
  return [
    project,
    ...projects.filter((item) => item.rootPath !== project.rootPath),
  ].slice(0, 20);
}

function buildProjectConversationGroups(
  sessions: domain.Session[],
  projects: domain.AssistantProject[],
  selectedProjectPath: string,
): ProjectConversationGroup[] {
  const projectsByPathKey = new Map<string, domain.AssistantProject>();
  const orderedProjectPathKeys: string[] = [];
  const conversationsByPathKey = new Map<string, domain.Session[]>();

  const addProject = (project: domain.AssistantProject) => {
    const pathKey = normalizeProjectPathKey(project.rootPath);
    if (!pathKey || !projectIsUserSelectable(project)) return;
    if (!projectsByPathKey.has(pathKey)) {
      projectsByPathKey.set(pathKey, project);
      orderedProjectPathKeys.push(pathKey);
    }
  };

  if (selectedProjectPath) {
    addProject(assistantProjectFromPath(selectedProjectPath));
  }

  for (const project of projects) {
    addProject(project);
  }

  for (const session of sessions) {
    const projectPath = sessionSidebarProjectPath(session);
    const pathKey = normalizeProjectPathKey(projectPath);
    if (!pathKey || !projectsByPathKey.has(pathKey)) continue;
    conversationsByPathKey.set(pathKey, [
      ...(conversationsByPathKey.get(pathKey) ?? []),
      session,
    ]);
  }

  return orderedProjectPathKeys
    .map((pathKey) => {
      const project = projectsByPathKey.get(pathKey);
      if (!project) return null;
      return {
        conversations: conversationsByPathKey.get(pathKey) ?? [],
        project,
        projectPath: project.rootPath,
      } satisfies ProjectConversationGroup;
    })
    .filter((group): group is ProjectConversationGroup => Boolean(group));
}

function isSessionGroupedUnderProject(
  session: domain.Session,
  projectGroups: ProjectConversationGroup[],
) {
  const pathKey = normalizeProjectPathKey(sessionSidebarProjectPath(session));
  if (!pathKey) return false;
  return projectGroups.some(
    (group) => normalizeProjectPathKey(group.projectPath) === pathKey,
  );
}

function sessionSidebarProjectPath(session: domain.Session) {
  const projectPath = session.projectPath?.trim() ?? "";
  if (!projectPath || isManagedWorkspacePath(projectPath)) return "";
  return projectPath;
}

function normalizeProjectPathKey(projectPath: string) {
  return projectPath.trim().replace(/[\\/]+$/, "");
}

function assistantProjectFromPath(rootPath: string): domain.AssistantProject {
  const now = new Date().toISOString();
  return {
    id: rootPath,
    name: projectNameFromPath(rootPath),
    rootPath,
    gitAvailable: false,
    timeOpened: now,
    timeUpdated: now,
  };
}

function projectPickerLabel(
  project: domain.AssistantProject | null,
  projectPath: string,
) {
  if (project?.name) return project.name;
  if (projectPath) return projectNameFromPath(projectPath);
  return "项目选择";
}

function projectNameFromPath(rootPath: string) {
  const trimmed = rootPath.trim().replace(/[\\/]+$/, "");
  const parts = trimmed.split(/[\\/]/).filter(Boolean);
  return parts.at(-1) || trimmed || "Project";
}

function projectIsUserSelectable(project: domain.AssistantProject) {
  return !isManagedWorkspacePath(project.rootPath);
}

function isManagedWorkspacePath(rootPath: string) {
  const parts = rootPath.trim().split(/[\\/]/).filter(Boolean);
  const rootIndex = parts.lastIndexOf("Aivo Workspaces");
  if (rootIndex < 0) return false;
  const datePart = parts[rootIndex + 1] ?? "";
  const workspacePart = parts[rootIndex + 2] ?? "";
  return /^\d{4}-\d{2}-\d{2}$/.test(datePart) && isManagedWorkspaceSlug(workspacePart);
}

function isManagedWorkspaceSlug(value: string) {
  return (
    value === "session" ||
    value.startsWith("session-") ||
    /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}(?:-\d+)?$/.test(
      value,
    )
  );
}

function normalizeSessionUpdatedPayload(payloads: unknown[]) {
  const payload = normalizeRecordPayload(payloads);
  return {
    sessionId: typeof payload?.sessionId === "string" ? payload.sessionId : "",
    session: normalizeSessionObject(payload?.session),
  };
}

function normalizeSessionObject(value: unknown): domain.Session | null {
  if (!value || typeof value !== "object" || Array.isArray(value)) return null;
  const record = value as Record<string, unknown>;
  if (typeof record.id === "string") return record as unknown as domain.Session;
  return null;
}

function normalizeTurnUpdatedPayload(payloads: unknown[]) {
  const payload = normalizeRecordPayload(payloads);
  return normalizeTurnObject(payload?.turn);
}

function normalizeTurnObject(value: unknown): domain.Turn | null {
  if (!value || typeof value !== "object" || Array.isArray(value)) return null;
  const record = value as Record<string, unknown>;
  if (typeof record.id === "string" && typeof record.sessionId === "string") {
    return record as unknown as domain.Turn;
  }
  return null;
}

function normalizeToolCallUpdatedPayload(payloads: unknown[]) {
  const payload = normalizeRecordPayload(payloads);
  return normalizeToolCallObject(payload?.toolCall);
}

function normalizeToolCallObject(value: unknown): domain.ToolCall | null {
  if (!value || typeof value !== "object" || Array.isArray(value)) return null;
  const record = value as Record<string, unknown>;
  if (typeof record.id === "string" && typeof record.sessionId === "string") {
    return record as unknown as domain.ToolCall;
  }
  return null;
}

function normalizeShellOutputPayload(payloads: unknown[]): ShellOutputPayload {
  const payload = normalizeRecordPayload(payloads);
  return {
    sessionId: typeof payload?.sessionId === "string" ? payload.sessionId : "",
    turnId: typeof payload?.turnId === "string" ? payload.turnId : "",
    toolCallId:
      typeof payload?.toolCallId === "string" ? payload.toolCallId : "",
    processRef:
      typeof payload?.processRef === "string" ? payload.processRef : "",
    stream: typeof payload?.stream === "string" ? payload.stream : "",
    chunk: typeof payload?.chunk === "string" ? payload.chunk : "",
    sequence:
      typeof payload?.sequence === "number" ? payload.sequence : undefined,
    timeCreated:
      typeof payload?.timeCreated === "string" ? payload.timeCreated : "",
  };
}

function normalizePermissionEventPayload(
  payloads: unknown[],
): PermissionRequest | null {
  const payload = normalizeRecordPayload(payloads);
  return (
    normalizePermissionRequestObject(payload?.permission) ??
    normalizePermissionRequestObject(payload)
  );
}

function normalizeQuestionEventPayload(
  payloads: unknown[],
): QuestionRequest | null {
  const payload = normalizeRecordPayload(payloads);
  return (
    normalizeQuestionRequestObject(payload?.question) ??
    normalizeQuestionRequestObject(payload)
  );
}

function normalizeTodoItemsUpdatedPayload(
  payloads: unknown[],
): { sessionId: string; projectPath: string; items: TodoItem[] } | null {
  const payload = normalizeRecordPayload(payloads);
  if (!payload || typeof payload.sessionId !== "string") return null;
  if (!Array.isArray(payload.items)) return null;
  return {
    sessionId: payload.sessionId,
    projectPath:
      typeof payload.projectPath === "string" ? payload.projectPath : "",
    items: payload.items as TodoItem[],
  };
}

function normalizeRecordPayload(
  payloads: unknown[],
): Record<string, unknown> | null {
  const first = payloads[0];
  if (first && typeof first === "object" && !Array.isArray(first)) {
    return first as Record<string, unknown>;
  }
  if (Array.isArray(first) && first[0] && typeof first[0] === "object") {
    return first[0] as Record<string, unknown>;
  }
  return null;
}

function normalizePermissionRequestObject(
  value: unknown,
): PermissionRequest | null {
  if (!value || typeof value !== "object") return null;
  if (Array.isArray(value)) return normalizePermissionRequestObject(value[0]);
  const record = value as Record<string, unknown>;
  if (typeof record.id === "string" && typeof record.toolName === "string") {
    return record as PermissionRequest;
  }
  return (
    normalizePermissionRequestObject(record.data) ??
    normalizePermissionRequestObject(record.properties)
  );
}

function normalizeQuestionRequestObject(value: unknown): QuestionRequest | null {
  if (!value || typeof value !== "object") return null;
  if (Array.isArray(value)) return normalizeQuestionRequestObject(value[0]);
  const record = value as Record<string, unknown>;
  if (typeof record.id === "string" && Array.isArray(record.questions)) {
    return record as QuestionRequest;
  }
  return (
    normalizeQuestionRequestObject(record.data) ??
    normalizeQuestionRequestObject(record.properties)
  );
}

function normalizeAssistantDeltaPayload(payloads: unknown[]) {
  const first = payloads[0];
  const payload =
    normalizeAssistantDeltaObject(first) ??
    normalizeAssistantDeltaObject(payloads);
  return {
    sessionId: payload?.sessionId ?? "",
    turnId: payload?.turnId ?? "",
    delta: payload?.delta ?? "",
  };
}

function normalizeAssistantDeltaObject(
  value: unknown,
): { sessionId?: string; turnId?: string; delta?: string } | null {
  if (!value || typeof value !== "object") return null;
  if (Array.isArray(value)) return normalizeAssistantDeltaObject(value[0]);
  const record = value as Record<string, unknown>;
  if (
    typeof record.sessionId === "string" ||
    typeof record.delta === "string"
  ) {
    return {
      sessionId:
        typeof record.sessionId === "string" ? record.sessionId : undefined,
      turnId: typeof record.turnId === "string" ? record.turnId : undefined,
      delta: typeof record.delta === "string" ? record.delta : undefined,
    };
  }
  return (
    normalizeAssistantDeltaObject(record.data) ??
    normalizeAssistantDeltaObject(record.properties)
  );
}

function normalizeProviderAuthUpdatedPayload(payloads: unknown[]) {
  const payload = normalizeRecordPayload(payloads);
  const status = payload?.status;
  if (!status || typeof status !== "object" || Array.isArray(status))
    return null;
  const statusRecord = status as Record<string, unknown>;
  if (typeof statusRecord.providerId !== "string") return null;
  return statusRecord as unknown as domain.ProviderAuthStatus;
}

function getActiveProvider(
  config: domain.AppConfig | null,
  providers: ProviderInfo[],
  selectedProviderId = "",
) {
  const providerId =
    selectedProviderId ||
    config?.provider?.id ||
    config?.defaultModel?.providerId ||
    providers[0]?.id ||
    "";
  return (
    providers.find((provider) => provider.id === providerId) ??
    providers[0] ??
    null
  );
}

function getConnectedModelProviders(
  config: domain.AppConfig | null,
  providers: ProviderInfo[],
  connectedProviders: ProviderInfo[],
) {
  if (connectedProviders.length > 0) return connectedProviders;
  const configuredProviderIds = new Set(
    [config?.provider?.id, config?.defaultModel?.providerId].filter(Boolean),
  );
  return providers.filter(
    (provider) => provider.connected || configuredProviderIds.has(provider.id),
  );
}

function getModelOptions(
  provider: ProviderInfo | null,
  catalogModels: ModelInfo[],
) {
  if (!provider) return [];
  const providerModels = provider.models?.length
    ? provider.models
    : catalogModels.filter((model) => model.providerId === provider.id);
  const seen = new Set<string>();
  return providerModels.filter((model) => {
    if (!model.id || seen.has(model.id)) return false;
    seen.add(model.id);
    return true;
  });
}

function getAllModelOptions(
  providers: ProviderInfo[],
  catalogModels: ModelInfo[],
): ModelOption[] {
  const out: ModelOption[] = [];
  const seen = new Set<string>();
  for (const provider of providers) {
    const models = getModelOptions(provider, catalogModels);
    for (const model of models) {
      const normalizedId = normalizeModelId(provider.id, model.id);
      const key = modelOptionKey(provider.id, normalizedId);
      if (seen.has(key)) continue;
      seen.add(key);
      out.push({
        ...model,
        id: normalizedId,
        providerId: provider.id,
        providerName: provider.name,
      });
    }
  }
  return out;
}

function getDefaultModelId(
  config: domain.AppConfig | null,
  provider: ProviderInfo | null,
  modelOptions: ModelInfo[],
) {
  if (!provider) return "";
  if (
    config?.defaultModel?.providerId === provider.id &&
    config.defaultModel.modelId
  ) {
    const modelId = normalizeModelId(provider.id, config.defaultModel.modelId);
    if (
      modelOptions.length === 0 ||
      modelOptions.some((model) => model.id === modelId)
    )
      return modelId;
  }
  if (config?.provider?.id === provider.id && config.provider.model) {
    const modelId = normalizeModelId(provider.id, config.provider.model);
    if (
      modelOptions.length === 0 ||
      modelOptions.some((model) => model.id === modelId)
    )
      return modelId;
  }
  return provider.defaultModelId || modelOptions[0]?.id || "";
}

function getModelLabel(modelOptions: ModelInfo[], modelId: string) {
  const normalizedModelId = normalizeModelId(
    modelOptions[0]?.providerId,
    modelId,
  );
  return (
    modelOptions.find((model) => model.id === normalizedModelId)?.name ||
    normalizedModelId
  );
}

function normalizeModelId(providerId: string | undefined, modelId: string) {
  if (providerId === "openai" && modelId === "gpt-5-codex") return "gpt-5.5";
  return modelId;
}

function normalizeReasoningEffort(effort: string | undefined) {
  if (
    effort === "low" ||
    effort === "medium" ||
    effort === "high" ||
    effort === "ultra"
  )
    return effort;
  if (effort === "低") return "low";
  if (effort === "中") return "medium";
  if (effort === "高") return "high";
  if (effort === "超高") return "ultra";
  return "medium";
}

function reasoningEffortLabel(effort: string) {
  switch (normalizeReasoningEffort(effort)) {
    case "low":
      return "低";
    case "high":
      return "高";
    case "ultra":
      return "超高";
    default:
      return "中";
  }
}

function normalizeServiceTier(serviceTier: string | undefined) {
  if (serviceTier === "priority" || serviceTier === "fast") return "priority";
  return "default";
}

function serviceTierLabel(serviceTier: string) {
  return normalizeServiceTier(serviceTier) === "priority" ? "快速" : "标准";
}

function normalizePermissionMode(mode: string | undefined): PermissionMode {
  if (
    mode === "request_approval" ||
    mode === "auto_approve" ||
    mode === "full_access"
  ) {
    return mode;
  }
  return "request_approval";
}

function providerSupportsServiceTier(providerId: string | undefined) {
  return providerId === "openai";
}

const MAX_COMPOSER_ATTACHMENT_BYTES = 50 * 1024 * 1024;

function modelOptionKey(providerId: string, modelId: string) {
  return `${providerId}:${normalizeModelId(providerId, modelId)}`;
}

function groupModelOptionsByProvider(models: ModelOption[]) {
  const groups: Array<{
    providerId: string;
    providerName: string;
    models: ModelOption[];
  }> = [];
  const indexes = new Map<string, number>();
  for (const model of models) {
    let index = indexes.get(model.providerId);
    if (index === undefined) {
      index = groups.length;
      indexes.set(model.providerId, index);
      groups.push({
        providerId: model.providerId,
        providerName: model.providerName,
        models: [],
      });
    }
    groups[index].models.push(model);
  }
  return groups;
}

function dragEventHasFiles(event: React.DragEvent<HTMLElement>) {
  return Array.from(event.dataTransfer.types).includes("Files");
}

async function readComposerAttachmentFiles(
  files: File[],
  modelRef: domain.ModelRef | null | undefined,
  modelInfo: ModelInfo | undefined,
) {
  const attachments: ComposerAttachment[] = [];
  const rejections: string[] = [];
  for (const file of files) {
    const mimeType = file.type || mimeTypeFromName(file.name);
    const kind = mimeType.startsWith("image/") ? "image" : "file";
    if (file.size > MAX_COMPOSER_ATTACHMENT_BYTES) {
      rejections.push(`${file.name} 超过 50 MB，不能作为模型附件发送。`);
      continue;
    }
    if (!modelSupportsAttachment(modelRef, modelInfo, kind, mimeType)) {
      rejections.push(
        `当前模型不支持${attachmentKindLabel(kind, mimeType)}：${file.name}`,
      );
      continue;
    }
    try {
      const data = await readFileAsBase64(file);
      attachments.push({
        id: crypto.randomUUID(),
        name: file.name || defaultAttachmentName(kind, mimeType),
        mimeType,
        size: file.size,
        kind,
        data,
        previewUrl: kind === "image" ? `data:${mimeType};base64,${data}` : undefined,
      });
    } catch {
      rejections.push(`${file.name} 读取失败。`);
    }
  }
  return { attachments, rejections };
}

function readFileAsBase64(file: File) {
  return new Promise<string>((resolve, reject) => {
    const reader = new FileReader();
    reader.onerror = () => reject(reader.error);
    reader.onload = () => {
      const result = typeof reader.result === "string" ? reader.result : "";
      resolve(result.includes(",") ? result.split(",").at(-1) || "" : result);
    };
    reader.readAsDataURL(file);
  });
}

function modelSupportsAttachment(
  modelRef: domain.ModelRef | null | undefined,
  modelInfo: ModelInfo | undefined,
  kind: ComposerAttachment["kind"],
  mimeType: string,
) {
  const capabilities = new Set(
    [...(modelInfo?.capabilities ?? []), ...(modelInfo?.modalities ?? [])].map(
      (item) => item.toLowerCase(),
    ),
  );
  if (kind === "image" && hasAnyCapability(capabilities, ["vision", "image", "image-input", "multimodal"])) {
    return true;
  }
  if (kind === "file" && hasAnyCapability(capabilities, ["file", "file-input", "document", "pdf", "multimodal"])) {
    return true;
  }
  const providerId = normalizeProviderIdForCapability(modelRef?.providerId);
  const modelId = modelRef?.modelId.toLowerCase() ?? "";
  if (!providerId || !modelId) return false;
  if (providerId === "openai" || providerId === "azure-openai") {
    return isLatestOpenAIMultimodalModel(modelId);
  }
  if (providerId === "anthropic" || providerId === "claude-code") {
    if (!isCurrentClaudeModel(modelId)) return false;
    return kind === "image" || mimeType === "application/pdf";
  }
  if (providerId === "google" || providerId === "gemini" || providerId === "google-vertex") {
    return modelId.includes("gemini-");
  }
  return false;
}

function hasAnyCapability(capabilities: Set<string>, values: string[]) {
  return values.some((value) => capabilities.has(value));
}

function normalizeProviderIdForCapability(providerId: string | undefined) {
  const normalized = providerId?.toLowerCase().trim() ?? "";
  if (normalized === "claude") return "anthropic";
  if (normalized === "vertex") return "google-vertex";
  return normalized;
}

function isLatestOpenAIMultimodalModel(modelId: string) {
  return /(^|[/:-])(gpt-5|gpt-4o|gpt-4\.1|o3|o4)/.test(modelId);
}

function isCurrentClaudeModel(modelId: string) {
  return modelId.includes("claude-3") || modelId.includes("claude-4") || modelId.includes("claude-5") || modelId.includes("sonnet") || modelId.includes("opus") || modelId.includes("haiku") || modelId.includes("fable") || modelId.includes("mythos");
}

function attachmentKindLabel(kind: ComposerAttachment["kind"], mimeType: string) {
  if (kind === "image") return "图片输入";
  if (mimeType === "application/pdf") return "PDF 文件输入";
  return "文件输入";
}

function mimeTypeFromName(name: string) {
  const extension = name.toLowerCase().split(".").pop() ?? "";
  switch (extension) {
    case "png":
      return "image/png";
    case "jpg":
    case "jpeg":
      return "image/jpeg";
    case "gif":
      return "image/gif";
    case "webp":
      return "image/webp";
    case "pdf":
      return "application/pdf";
    case "txt":
    case "md":
      return "text/plain";
    case "json":
      return "application/json";
    case "csv":
      return "text/csv";
    default:
      return "application/octet-stream";
  }
}

function defaultAttachmentName(kind: ComposerAttachment["kind"], mimeType: string) {
  if (kind === "image") return `pasted-image.${mimeType.split("/").at(-1) || "png"}`;
  return "attachment";
}

function composerAttachmentToConversationAttachment(
  attachment: ComposerAttachment,
): ConversationUserAttachment {
  return {
    id: attachment.id,
    name: attachment.name,
    mimeType: attachment.mimeType,
    kind: attachment.kind,
    previewUrl: attachment.previewUrl,
    size: attachment.size,
  };
}

function formatAttachmentMeta(attachment: ComposerAttachment) {
  const type = attachment.kind === "image" ? "图片" : readableAttachmentType(attachment.mimeType);
  return `${type} · ${formatBytes(attachment.size)}`;
}

function readableAttachmentType(mimeType: string) {
  if (mimeType === "application/pdf") return "PDF";
  if (mimeType.includes("spreadsheet") || mimeType.includes("csv")) return "表格";
  if (mimeType.includes("presentation")) return "演示文稿";
  if (mimeType.includes("wordprocessing") || mimeType.includes("document")) return "文档";
  if (mimeType.startsWith("text/") || mimeType.includes("json")) return "文本";
  return "文件";
}

function formatBytes(size: number) {
  if (!Number.isFinite(size) || size <= 0) return "0 B";
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
  return `${(size / (1024 * 1024)).toFixed(1)} MB`;
}

function formatAttachmentOnlyPrompt(attachments: ComposerAttachment[]) {
  if (attachments.length === 0) return "附件";
  if (attachments.length === 1) return `附件：${attachments[0].name}`;
  return `附件：${attachments.length} 个文件`;
}

function formatModelTriggerLabel(modelLabel: string) {
  return modelLabel
    .replace(/^GPT-/i, "")
    .replace(/^Claude\s+/i, "")
    .replace(/^Gemini\s+/i, "");
}

function compactModelLabel(modelLabel: string) {
  return modelLabel.toLowerCase();
}

function SubagentSessionActionBar({
  agentRun,
  onBack,
  onCancel,
  onHeightChange,
}: {
  agentRun?: AgentRun;
  onBack: () => void;
  onCancel?: () => void;
  onHeightChange: (height: number) => void;
}) {
  const barHeight = 72;

  useLayoutEffect(() => {
    onHeightChange(barHeight);
  }, [onHeightChange]);

  const status = agentRun?.status || "";
  const modeLabel = agentModeLabel(agentRun?.mode || "assistant");
  const statusLabel = agentRunStatusLabel(status);

  return (
    <div
      className="flex min-w-0 items-center justify-between gap-3 rounded-2xl border border-border bg-card px-4 shadow-lg shadow-foreground/5"
      style={{ height: barHeight }}
    >
      <div className="flex min-w-0 items-center gap-3">
        <Button onClick={onBack} type="button" variant="outline">
          <ArrowLeft />
          返回父会话
        </Button>
        <div className="hidden min-w-0 flex-col sm:flex">
          <div className="truncate text-sm font-medium text-foreground">
            子代理 · {modeLabel}
          </div>
          <div className="truncate text-xs text-muted-foreground">
            {agentRun?.prompt || "只读子代理会话"}
          </div>
        </div>
      </div>
      <div className="flex shrink-0 items-center gap-2">
        {statusLabel ? (
          <span className={cn("text-xs", agentRunStatusClass(status))}>
            {statusLabel}
          </span>
        ) : null}
        {onCancel ? (
          <Button onClick={onCancel} type="button" variant="destructive">
            <X />
            取消运行
          </Button>
        ) : null}
      </div>
    </div>
  );
}

function agentRunStatusLabel(status: string) {
  switch (status) {
    case "running":
      return "运行中";
    case "completed":
    case "success":
      return "已完成";
    case "failed":
      return "失败";
    case "cancelled":
      return "已取消";
    case "pending_approval":
      return "等待批准";
    default:
      return status;
  }
}

function agentRunStatusClass(status: string) {
  if (status === "completed" || status === "success") {
    return "text-emerald-600 dark:text-emerald-400";
  }
  if (status === "failed" || status === "cancelled") {
    return "text-destructive";
  }
  return "text-muted-foreground";
}

function PromptComposer({
  agentMode,
  agentModes,
  allModelOptions,
  modelId,
  modelLabel,
  modelOptions,
  onAddAttachments,
  onAgentModeSelect,
  onExtraHeightChange,
  onHeightChange,
  onModelSelect,
  onPermissionModeSelect,
  onPromptChange,
  onProjectAdd,
  onProjectClear,
  onProjectSelect,
  onReasoningEffortSelect,
  onRemoveAttachment,
  onServiceTierSelect,
  onSubmit,
  pending,
  permissionMode,
  prompt,
  project,
  projectPath,
  projects,
  attachments,
  reasoningEffort,
  serviceTier,
  showProjectPicker,
  showServiceTier,
}: {
  agentMode: AgentModeId;
  agentModes: AgentModeDefinition[];
  allModelOptions: ModelOption[];
  modelId: string;
  modelLabel: string;
  modelOptions: ModelInfo[];
  onAddAttachments: (files: FileList | null) => void;
  onAgentModeSelect: (mode: AgentModeId) => void;
  onExtraHeightChange: (height: number) => void;
  onHeightChange: (height: number) => void;
  onModelSelect: (option: ModelOption) => void;
  onPermissionModeSelect: (mode: PermissionMode) => void;
  onPromptChange: (prompt: string) => void;
  onProjectAdd: () => void;
  onProjectClear: () => void;
  onProjectSelect: (project: domain.AssistantProject) => void;
  onReasoningEffortSelect: (reasoningEffort: string) => void;
  onRemoveAttachment: (id: string) => void;
  onServiceTierSelect: (serviceTier: string) => void;
  onSubmit: () => void;
  pending: boolean;
  permissionMode: PermissionMode;
  prompt: string;
  project: domain.AssistantProject | null;
  projectPath: string;
  projects: domain.AssistantProject[];
  attachments: ComposerAttachment[];
  reasoningEffort: string;
  serviceTier: string;
  showProjectPicker: boolean;
  showServiceTier: boolean;
}) {
  const rootRef = useRef<HTMLDivElement>(null);
  const composerCardRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const minTextareaHeight = 0;
  const maxTextareaHeight = 300;
  const [compactToolbar, setCompactToolbar] = useState(false);
  const textareaHeights = useAutoTextareaHeight(
    prompt,
    minTextareaHeight,
    maxTextareaHeight,
    textareaRef,
  );
  useLayoutEffect(() => {
    const rootElement = rootRef.current;
    const cardElement = composerCardRef.current;
    if (!rootElement || !cardElement) return;
    const updateHeight = () => {
      const rootHeight = Math.ceil(rootElement.getBoundingClientRect().height);
      const cardHeight = Math.ceil(cardElement.getBoundingClientRect().height);
      const cardWidth = Math.ceil(cardElement.getBoundingClientRect().width);
      onHeightChange(cardHeight);
      onExtraHeightChange(Math.max(0, rootHeight - cardHeight));
      setCompactToolbar(cardWidth < 560);
    };
    updateHeight();
    const resizeObserver = new ResizeObserver(updateHeight);
    resizeObserver.observe(rootElement);
    resizeObserver.observe(cardElement);
    return () => resizeObserver.disconnect();
  }, [onExtraHeightChange, onHeightChange, showProjectPicker]);

  return (
    <div className="flex min-w-0 flex-col" ref={rootRef}>
      <div
        className="relative z-10 flex min-w-0 flex-col overflow-hidden rounded-2xl border border-border bg-card px-5 py-3 shadow-lg shadow-foreground/5"
        ref={composerCardRef}
      >
        {attachments.length > 0 ? (
          <AttachmentGroup>
            {attachments.map((attachment) => (
              <Attachment
                className={cn(
                  "bg-background/70",
                  attachment.kind === "image"
                    ? "size-24 overflow-hidden p-0"
                    : "w-28",
                )}
                key={attachment.id}
                orientation="vertical"
                size="sm"
              >
                {attachment.kind === "image" ? (
                  attachment.previewUrl ? (
                    <img
                      alt=""
                      className="absolute inset-0 size-full object-cover"
                      src={attachment.previewUrl}
                    />
                  ) : (
                    <div className="absolute inset-0 flex items-center justify-center bg-muted text-muted-foreground">
                      <Image />
                    </div>
                  )
                ) : (
                  <AttachmentMedia className="h-16" variant="icon">
                    <File />
                  </AttachmentMedia>
                )}
                {attachment.kind === "file" ? (
                  <AttachmentContent>
                    <AttachmentTitle>{attachment.name}</AttachmentTitle>
                    <AttachmentDescription>
                      {formatAttachmentMeta(attachment)}
                    </AttachmentDescription>
                  </AttachmentContent>
                ) : null}
                <AttachmentActions className="group-data-[orientation=vertical]/attachment:right-1 group-data-[orientation=vertical]/attachment:top-1">
                  <AttachmentAction
                    aria-label={`移除附件 ${attachment.name}`}
                    className="rounded-full bg-background/90 shadow-sm"
                    onClick={() => onRemoveAttachment(attachment.id)}
                    type="button"
                  >
                    <X />
                  </AttachmentAction>
                </AttachmentActions>
              </Attachment>
            ))}
          </AttachmentGroup>
        ) : null}
        <ScrollArea
          className=" [&_[data-slot=scroll-area-scrollbar]]:mr-2 [&_[data-slot=scroll-area-scrollbar]]:mt-2"
          style={
            textareaHeights.viewport
              ? { height: textareaHeights.viewport }
              : undefined
          }
        >
          <textarea
            aria-label="任务描述"
            className="block w-full resize-none overflow-hidden bg-transparent text-sm  leading-normal text-foreground outline-none placeholder:text-muted-foreground"
            onChange={(event) => onPromptChange(event.target.value)}
            onPaste={(event) => {
              const files = event.clipboardData.files;
              if (!files.length) return;
              event.preventDefault();
              onAddAttachments(files);
            }}
            onKeyDown={(event) => {
              if (
                event.key === "Enter" &&
                !event.shiftKey &&
                !event.nativeEvent.isComposing
              ) {
                event.preventDefault();
                onSubmit();
              }
            }}
            placeholder="随心输入"
            ref={textareaRef}
            rows={2}
            style={
              textareaHeights.content
                ? { height: textareaHeights.content }
                : undefined
            }
            value={prompt}
          />
        </ScrollArea>

        <div className="flex min-w-0 items-center justify-between gap-2 sm:gap-2.5">
          <div className="flex h-9 min-w-0 items-center gap-1.5 sm:gap-3">
            <input
              className="hidden"
              multiple
              onChange={(event) => {
                onAddAttachments(event.currentTarget.files);
                event.currentTarget.value = "";
              }}
              ref={fileInputRef}
              type="file"
            />
            <Button
              aria-label="添加文件"
              className="rounded-full"
              onClick={() => fileInputRef.current?.click()}
              size="icon-sm"
              type="button"
              variant="ghost"
            >
              <Plus />
            </Button>

            <PermissionModeMenu
              compact={compactToolbar}
              mode={permissionMode}
              onModeSelect={onPermissionModeSelect}
            />
            <AgentModeMenu
              compact={compactToolbar}
              mode={agentMode}
              modes={agentModes}
              onModeSelect={onAgentModeSelect}
            />
          </div>

          <div className="flex h-9 min-w-0 items-center gap-1.5 sm:gap-2.5">
            <ModelSettingsMenu
              allModelOptions={allModelOptions}
              compact={compactToolbar}
              modelId={modelId}
              modelLabel={modelLabel}
              modelOptions={modelOptions}
              onModelSelect={onModelSelect}
              onReasoningEffortSelect={onReasoningEffortSelect}
              reasoningEffort={reasoningEffort}
              onServiceTierSelect={onServiceTierSelect}
              serviceTier={serviceTier}
              showServiceTier={showServiceTier}
            />

            <Button
              aria-label="语音输入"
              className="rounded-full"
              size="icon-sm"
              type="button"
              variant="ghost"
            >
              <Mic />
            </Button>

            <Button
              aria-label={pending ? "停止" : "发送"}
              className="rounded-full"
              disabled={!pending && !prompt.trim() && attachments.length === 0}
              onClick={onSubmit}
              size="icon"
              type="button"
              variant="default"
            >
              {pending ? <Pause /> : <ArrowUp />}
            </Button>
          </div>
        </div>
      </div>
      {showProjectPicker ? (
        <div className="-mt-5 rounded-2xl bg-card p-2 pt-7 shadow-lg shadow-foreground/5">
          <ProjectPicker
            onAddProject={onProjectAdd}
            onProjectClear={onProjectClear}
            onProjectSelect={onProjectSelect}
            project={project}
            projectPath={projectPath}
            projects={projects}
          />
        </div>
      ) : null}
    </div>
  );
}

function ProjectPicker({
  onAddProject,
  onProjectClear,
  onProjectSelect,
  project,
  projectPath,
  projects,
}: {
  onAddProject: () => void;
  onProjectClear: () => void;
  onProjectSelect: (project: domain.AssistantProject) => void;
  project: domain.AssistantProject | null;
  projectPath: string;
  projects: domain.AssistantProject[];
}) {
  const [open, setOpen] = useState(false);
  const label = projectPickerLabel(project, projectPath);
  const hasCurrentProject = Boolean(projectPath);

  return (
    <Popover onOpenChange={setOpen} open={open}>
      <div className="group/project-picker relative inline-flex">
        <PopoverTrigger asChild>
          <Button type="button" variant="ghost">
            <FileText
              className={cn(
                "transition-opacity",
                hasCurrentProject &&
                  "group-hover/project-picker:opacity-0",
              )}
            />
            <span className="truncate">{label}</span>
          </Button>
        </PopoverTrigger>
        {hasCurrentProject ? (
          <button
            aria-label="清除当前项目选择"
            className="absolute left-1.5 top-1/2 z-10 inline-flex size-4 -translate-y-1/2 items-center justify-center rounded-sm text-muted-foreground opacity-0 transition-opacity hover:bg-muted hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/30 group-hover/project-picker:opacity-100"
            onClick={(event) => {
              event.preventDefault();
              event.stopPropagation();
              onProjectClear();
              setOpen(false);
            }}
            title="清除当前项目选择"
            type="button"
          >
            <X className="size-3.5" aria-hidden="true" />
          </button>
        ) : null}
      </div>
      <PopoverContent align="start">
        <Command>
          <CommandInput placeholder="搜索项目" />
          <CommandList>
            <CommandEmpty>没有找到项目</CommandEmpty>
            {projects.length > 0 ? (
              <CommandGroup>
                {projects.map((item) => (
                  <CommandItem
                    data-checked={item.rootPath === projectPath}
                    key={item.rootPath}
                    onSelect={() => {
                      onProjectSelect(item);
                      setOpen(false);
                    }}
                    value={`${item.name} ${item.rootPath}`}
                  >
                    <FileText />
                    <span className="min-w-0 flex-1 truncate">
                      {item.name || projectNameFromPath(item.rootPath)}
                    </span>
                  </CommandItem>
                ))}
              </CommandGroup>
            ) : null}
            <CommandSeparator />
            <CommandGroup>
              <CommandItem
                onSelect={() => {
                  onAddProject();
                  setOpen(false);
                }}
                value="New project 添加项目"
              >
                <Plus />
                <span>New project</span>
              </CommandItem>
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}

const permissionModeOptions: Array<{
  mode: PermissionMode;
  label: string;
  description: string;
}> = [
  {
    mode: "request_approval",
    label: "请求批准",
    description: "编辑外部文件和使用互联网时始终询问",
  },
  {
    mode: "auto_approve",
    label: "替我批准",
    description: "仅对检测到的风险操作请求批准",
  },
  {
    mode: "full_access",
    label: "完全访问权限",
    description: "可不受限制地访问互联网和您电脑上的任何文件",
  },
];

const fallbackAgentModes: AgentModeDefinition[] = [
  {
    id: "code",
    displayName: "Code",
    description: "默认编码主代理，按权限使用完整工具",
    prompt: "",
    toolsets: [],
  },
  {
    id: "assistant",
    displayName: "Assistant",
    description: "通用对话，必要时可编码",
    prompt: "",
    toolsets: [],
  },
  {
    id: "build",
    displayName: "Build",
    description: "实现代码、编辑文件、运行验证",
    prompt: "",
    toolsets: [],
  },
  {
    id: "explore",
    displayName: "Explore",
    description: "只读探索代码与方案",
    prompt: "",
    toolsets: [],
  },
  {
    id: "plan",
    displayName: "Plan",
    description: "只规划，不修改",
    prompt: "",
    toolsets: [],
  },
  {
    id: "planner",
    displayName: "Planner",
    description: "Plan 兼容模式",
    prompt: "",
    toolsets: [],
    hidden: true,
  },
  {
    id: "review",
    displayName: "Review",
    description: "只读审查代码风险",
    prompt: "",
    toolsets: [],
  },
  {
    id: "debug",
    displayName: "Debug",
    description: "诊断问题，可运行验证但不改文件",
    prompt: "",
    toolsets: [],
  },
  {
    id: "summary",
    displayName: "Summary",
    description: "隐藏总结模式",
    prompt: "",
    toolsets: [],
    hidden: true,
  },
  {
    id: "title",
    displayName: "Title",
    description: "隐藏标题模式",
    prompt: "",
    toolsets: [],
    hidden: true,
  },
];

function AgentModeMenu({
  compact,
  mode,
  modes,
  onModeSelect,
}: {
  compact: boolean;
  mode: AgentModeId;
  modes: AgentModeDefinition[];
  onModeSelect: (mode: AgentModeId) => void;
}) {
  const options = modes.length > 0 ? modes : fallbackAgentModes;
  const visibleOptions = options.filter((option) => !option.hidden);
  const selectedMode = normalizeAgentMode(mode);
  const selectedOption =
    options.find((option) => option.id === selectedMode) ??
    visibleOptions[0] ??
    options[0];

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          className="rounded-full font-semibold"
          size="sm"
          type="button"
          variant="ghost"
        >
          <Bot />
          <span className={cn(compact && "hidden")}>
            {agentModeShortLabel(selectedOption)}
          </span>
          <ChevronDown data-icon="inline-end" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent
        align="start"
        className="flex w-72 max-w-[calc(100vw-2rem)] flex-col gap-1"
        side="top"
      >
        <DropdownMenuLabel>
          <span>Agent 模式</span>
        </DropdownMenuLabel>
        {visibleOptions.map((option) => {
          const selected = option.id === selectedMode;
          return (
            <DropdownMenuItem
              className={cn(selected && "bg-accent")}
              key={option.id}
              onSelect={() => onModeSelect(option.id)}
            >
              <Bot className="text-foreground" />
              <span className="min-w-0 flex-1">
                <span className="block font-semibold text-foreground">
                  {agentModeShortLabel(option)}
                </span>
                <span className="block truncate text-muted-foreground">
                  {option.description}
                </span>
              </span>
              {selected ? <Check className="text-foreground" /> : null}
            </DropdownMenuItem>
          );
        })}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function agentModeShortLabel(mode: AgentModeDefinition) {
  switch (mode.id) {
    case "code":
      return "代码";
    case "assistant":
      return "助手";
    case "build":
      return "构建";
    case "explore":
      return "探索";
    case "plan":
    case "planner":
      return "规划";
    case "review":
      return "审查";
    case "debug":
      return "调试";
    case "summary":
      return "总结";
    case "title":
      return "标题";
    default:
      return mode.displayName;
  }
}

function agentModeLabel(mode: AgentModeId | string) {
  const definition =
    fallbackAgentModes.find((item) => item.id === mode) ??
    fallbackAgentModes[0];
  return agentModeShortLabel(definition);
}

function normalizeAgentMode(
  mode: AgentModeId | string | undefined,
): AgentModeId {
  switch (mode) {
    case "code":
    case "assistant":
    case "build":
    case "explore":
    case "plan":
    case "review":
    case "debug":
    case "summary":
    case "title":
	case "scheduler_worker":
      return mode;
    case "planner":
      return "plan";
    default:
      return "code";
  }
}

function PermissionModeMenu({
  compact,
  mode,
  onModeSelect,
}: {
  compact: boolean;
  mode: PermissionMode;
  onModeSelect: (mode: PermissionMode) => void;
}) {
  const selectedMode = normalizePermissionMode(mode);
  const selectedOption =
    permissionModeOptions.find((option) => option.mode === selectedMode) ??
    permissionModeOptions[0];

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          className="rounded-full font-semibold text-primary"
          size="sm"
          type="button"
          variant="ghost"
        >
          {permissionModeIcon(selectedMode)}
          <span className={cn(compact && "hidden")}>
            {selectedOption.label.replace("权限", "")}
          </span>
          <ChevronDown data-icon="inline-end" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent
        align="start"
        className="flex w-80 max-w-[calc(100vw-2rem)] flex-col gap-1"
        side="top"
      >
        <DropdownMenuLabel>
          <span>应如何批准 Codex 操作？</span>
        </DropdownMenuLabel>
        {permissionModeOptions.map((option) => {
          const selected = option.mode === selectedMode;
          return (
            <DropdownMenuItem
              className={cn(selected && "bg-accent")}
              key={option.mode}
              onSelect={() => onModeSelect(option.mode)}
            >
              <span className="text-foreground">
                {permissionModeIcon(option.mode)}
              </span>
              <span className="min-w-0 flex-1">
                <span className="block font-semibold text-foreground">
                  {option.label}
                </span>
                <span className="block truncate text-muted-foreground">
                  {option.description}
                </span>
              </span>
              {selected ? <Check className="text-foreground" /> : null}
            </DropdownMenuItem>
          );
        })}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function permissionModeIcon(mode: PermissionMode, className?: string) {
  switch (normalizePermissionMode(mode)) {
    case "auto_approve":
      return <Bot className={className} data-icon="inline-start" />;
    case "full_access":
      return <ShieldAlert className={className} data-icon="inline-start" />;
    default:
      return <Hand className={className} data-icon="inline-start" />;
  }
}

function ModelSettingsMenu({
  compact,
  allModelOptions,
  modelId,
  modelLabel,
  modelOptions,
  onModelSelect,
  onReasoningEffortSelect,
  onServiceTierSelect,
  reasoningEffort,
  serviceTier,
  showServiceTier,
}: {
  compact: boolean;
  allModelOptions: ModelOption[];
  modelId: string;
  modelLabel: string;
  modelOptions: ModelInfo[];
  onModelSelect: (option: ModelOption) => void;
  onReasoningEffortSelect: (reasoningEffort: string) => void;
  onServiceTierSelect: (serviceTier: string) => void;
  reasoningEffort: string;
  serviceTier: string;
  showServiceTier: boolean;
}) {
  const [query, setQuery] = useState("");
  const [menuOpen, setMenuOpen] = useState(false);
  const [connectDialogOpen, setConnectDialogOpen] = useState(false);
  const { catalog, reload, setCatalog, setConfig, setError } = useAppConfig();
  const currentProviderId = modelOptions[0]?.providerId || "";
  const normalizedQuery = query.trim().toLowerCase();
  const filteredModels = allModelOptions.filter((model) => {
    if (!normalizedQuery) return true;
    return `${model.providerName} ${model.name} ${model.id}`
      .toLowerCase()
      .includes(normalizedQuery);
  });
  const groupedModels = groupModelOptionsByProvider(filteredModels);
  const activeModelKey = modelOptionKey(currentProviderId, modelId);

  return (
    <>
      <DropdownMenu open={menuOpen} onOpenChange={setMenuOpen}>
        <DropdownMenuTrigger asChild>
          <Button
            className="rounded-full"
            size="sm"
            type="button"
            variant="ghost"
          >
            <span>
              {compact ? compactModelLabel(modelLabel) : modelLabel}
              {!compact ? ` ${reasoningEffortLabel(reasoningEffort)}` : ""}
              {!compact && showServiceTier
                ? ` ${serviceTierLabel(serviceTier)}`
                : ""}
            </span>
            <ChevronDown
              className="text-muted-foreground"
              data-icon="inline-end"
            />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuLabel>推理</DropdownMenuLabel>
          {[
            { value: "low", label: "低" },
            { value: "medium", label: "中" },
            { value: "high", label: "高" },
            { value: "ultra", label: "超高" },
          ].map((level) => (
            <DropdownMenuItem
              key={level.value}
              onSelect={(event: Event) => {
                event.preventDefault();
                onReasoningEffortSelect(level.value);
              }}
            >
              <span>{level.label}</span>
              {level.value === normalizeReasoningEffort(reasoningEffort) && (
                <Check className="ml-auto" />
              )}
            </DropdownMenuItem>
          ))}
          <DropdownMenuSeparator />
          <DropdownMenuSub>
            <DropdownMenuSubTrigger>
              <span>{getModelLabel(modelOptions, modelId) || modelLabel}</span>
            </DropdownMenuSubTrigger>
            <DropdownMenuSubContent>
              <div className="flex items-center gap-1 px-1 py-1">
                <div className="relative min-w-0 flex-1">
                  <Search className="pointer-events-none absolute left-2 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
                  <Input
                    className="pl-8"
                    onChange={(event) => setQuery(event.target.value)}
                    onKeyDown={(event) => event.stopPropagation()}
                    placeholder="搜索模型"
                    value={query}
                  />
                </div>
                <Button
                  aria-label="添加提供商"
                  className="shrink-0"
                  onClick={(event) => {
                    event.preventDefault();
                    event.stopPropagation();
                    setMenuOpen(false);
                    setConnectDialogOpen(true);
                  }}
                  onMouseDown={(event) => event.preventDefault()}
                  size="icon-sm"
                  type="button"
                  variant="ghost"
                >
                  <Plus />
                </Button>
              </div>
              <ScrollArea>
                {groupedModels.map((group) => (
                  <div key={group.providerId} className="py-1">
                    <DropdownMenuLabel>{group.providerName}</DropdownMenuLabel>
                    {group.models.map((model) => (
                      <DropdownMenuItem
                        key={modelOptionKey(model.providerId, model.id)}
                        onSelect={() => onModelSelect(model)}
                      >
                        <span className="min-w-0 truncate">
                          {model.name || model.id}
                        </span>
                        {modelOptionKey(model.providerId, model.id) ===
                        activeModelKey ? (
                          <Check className="ml-auto" />
                        ) : null}
                      </DropdownMenuItem>
                    ))}
                  </div>
                ))}
                {groupedModels.length === 0 ? (
                  <div className="px-2 py-6 text-center text-sm text-muted-foreground">
                    没有匹配的模型
                  </div>
                ) : null}
              </ScrollArea>
            </DropdownMenuSubContent>
          </DropdownMenuSub>
          {showServiceTier ? (
            <DropdownMenuSub>
              <DropdownMenuSubTrigger>速度</DropdownMenuSubTrigger>
              <DropdownMenuSubContent>
                {[
                  { value: "default", label: "标准" },
                  { value: "priority", label: "快速" },
                ].map((tier) => (
                  <DropdownMenuItem
                    key={tier.value}
                    onSelect={(event: Event) => {
                      event.preventDefault();
                      onServiceTierSelect(tier.value);
                    }}
                  >
                    <span>{tier.label}</span>
                    {tier.value === normalizeServiceTier(serviceTier) && (
                      <Check className="ml-auto" />
                    )}
                  </DropdownMenuItem>
                ))}
              </DropdownMenuSubContent>
            </DropdownMenuSub>
          ) : null}
        </DropdownMenuContent>
      </DropdownMenu>
      <ProviderConnectDialog
        catalogProviders={catalog?.providers ?? []}
        onConnected={async (option) => {
          if (option) onModelSelect(option);
          await reload();
        }}
        onOpenChange={setConnectDialogOpen}
        open={connectDialogOpen}
        setCatalog={setCatalog}
        setConfig={setConfig}
        setError={setError}
      />
    </>
  );
}

function ProviderConnectDialog({
  catalogProviders,
  onConnected,
  onOpenChange,
  open,
  setCatalog,
  setConfig,
  setError,
}: {
  catalogProviders: ProviderInfo[];
  onConnected: (option: ModelOption | null) => Promise<void>;
  onOpenChange: (open: boolean) => void;
  open: boolean;
  setCatalog: (catalog: CatalogState) => void;
  setConfig: (config: domain.AppConfig) => void;
  setError: (error: string) => void;
}) {
  const [query, setQuery] = useState("");
  const [selectedProvider, setSelectedProvider] =
    useState<ProviderPickOption | null>(null);
  const [providerDialogStep, setProviderDialogStep] =
    useState<ProviderDialogStep>("details");
  const [authMode, setAuthMode] = useState<ProviderAuthMode>("api-key");
  const [oauthStarted, setOauthStarted] = useState(false);
  const [oauthStartResult, setOauthStartResult] =
    useState<domain.ProviderAuthStartResult | null>(null);
  const [oauthStatus, setOauthStatus] =
    useState<domain.ProviderAuthStatus | null>(null);
  const [callbackInput, setCallbackInput] = useState("");
  const [apiKey, setApiKey] = useState("");
  const [customProviderForm, setCustomProviderForm] =
    useState<CustomProviderForm>(() => emptyCustomProviderForm());
  const [selectedModelId, setSelectedModelId] = useState("");
  const [authSuccessMessage, setAuthSuccessMessage] = useState("");
  const authSuccessNotifiedRef = useRef(false);
  const [submitting, setSubmitting] = useState(false);
  const [localError, setLocalError] = useState("");
  const providerOptions = useMemo(
    () => providerPickerOptions(catalogProviders),
    [catalogProviders],
  );
  const normalizedQuery = query.trim().toLowerCase();
  const filteredProviders = normalizedQuery
    ? providerOptions.filter((provider) =>
        `${provider.name} ${provider.id} ${provider.type}`
          .toLowerCase()
          .includes(normalizedQuery),
      )
    : providerOptions;

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
      const status = normalizeProviderAuthUpdatedPayload(payloads);
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
  }, [markOpenAIAuthorized]);

  function resetDialog() {
    setQuery("");
    setSelectedProvider(null);
    setProviderDialogStep("details");
    setAuthMode("api-key");
    setOauthStarted(false);
    setOauthStartResult(null);
    setOauthStatus(null);
    setCallbackInput("");
    setApiKey("");
    setCustomProviderForm(emptyCustomProviderForm());
    setSelectedModelId("");
    setAuthSuccessMessage("");
    authSuccessNotifiedRef.current = false;
    setSubmitting(false);
    setLocalError("");
  }

  function selectProvider(provider: ProviderPickOption) {
    setSelectedProvider(provider);
    const nextAuthMode = provider.id === "openai" ? "oauth-browser" : "api-key";
    setAuthMode(nextAuthMode);
    setProviderDialogStep(provider.id === "openai" ? "options" : "details");
    setOauthStarted(false);
    setOauthStartResult(null);
    setOauthStatus(null);
    setCallbackInput("");
    setApiKey("");
    setAuthSuccessMessage("");
    authSuccessNotifiedRef.current = false;
    setCustomProviderForm(customProviderFormFor(provider));
    setSelectedModelId(
      catalogDefaultModelForProvider(catalogProviders, provider.id) ||
        defaultModelForProvider(provider.id),
    );
    setLocalError("");
    onOpenChange(false);
  }

  function closeProviderDetails() {
    resetDialog();
  }

  function selectOpenAIAuthMode(nextMode: ProviderAuthMode) {
    setOauthStarted(false);
    setOauthStartResult(null);
    setOauthStatus(null);
    setCallbackInput("");
    setApiKey("");
    setAuthSuccessMessage("");
    setAuthMode(nextMode);
    setProviderDialogStep("details");
  }

  function resetAuthMode(nextMode: ProviderAuthMode) {
    setOauthStarted(false);
    setOauthStartResult(null);
    setOauthStatus(null);
    setCallbackInput("");
    setApiKey("");
    setAuthSuccessMessage("");
    setAuthMode(nextMode);
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

  async function submitProvider() {
    if (!selectedProvider) return;
    const isCustom = selectedProvider.id === "custom-api";
    const customModel = customProviderForm.models.find((model) =>
      model.name.trim(),
    );
    const providerId = isCustom
      ? customProviderForm.providerId.trim()
      : selectedProvider.id;
    const providerName = isCustom
      ? customProviderForm.displayName.trim()
      : selectedProvider.name;
    const baseUrl = isCustom
      ? customProviderForm.baseUrl.trim()
      : selectedProvider.baseUrl ||
        providerBaseURLDefaults[selectedProvider.id] ||
        "";
    const modelId = isCustom
      ? customModel?.name.trim()
      : selectedModelId ||
        catalogDefaultModelForProvider(catalogProviders, selectedProvider.id) ||
        defaultModelForProvider(selectedProvider.id);
    if (!providerId) {
      setLocalError("请输入 Provider ID。");
      return;
    }
    if (selectedProvider.id === "openai" && authMode !== "api-key") {
      setSubmitting(true);
      setLocalError("");
      try {
        const oauthReady = await startOrCheckOpenAIOAuth();
        if (!oauthReady) return;
      } catch (err) {
        const message = err instanceof Error ? err.message : String(err);
        setLocalError(message);
        setError(message);
        return;
      } finally {
        setSubmitting(false);
      }
    } else if (isCustom && !customProviderFormIsValid(customProviderForm)) {
      setLocalError("请填写完整的自定义 Provider 信息。");
      return;
    } else if (!isCustom && !apiKey.trim()) {
      setLocalError("请输入 API Key。");
      return;
    }
    setSubmitting(true);
    setLocalError("");
    setError("");
    const input = {
      providerId,
      name: providerName || providerId,
      type: isCustom
        ? customProviderForm.protocol
        : selectedProvider.type ||
          providerProtocolForProvider(selectedProvider.id),
      baseUrl,
      apiKey: isCustom
        ? customProviderForm.apiKey.trim()
        : authMode === "api-key"
          ? apiKey.trim()
          : undefined,
      modelId: modelId || "default",
      method: selectedProvider.id === "openai" ? authMode : "api-key",
      headers: isCustom
        ? headersFromRows(customProviderForm.headers)
        : undefined,
    } as domain.ProviderConnectInput;
    try {
      let modelToSelect = input.modelId || "default";
      if (hasAppBridge()) {
        try {
          const refreshedCatalog = await refreshProviderModels(input);
          setCatalog(refreshedCatalog);
          modelToSelect =
            defaultModelFromCatalog(refreshedCatalog, providerId) ||
            modelToSelect;
          input.modelId = modelToSelect;
        } catch {
          // Connecting with the existing/default model is still useful if refresh fails.
        }
        const nextCatalog = await connectProvider(input);
        setCatalog(nextCatalog);
        const option = modelOptionFromCatalog(
          nextCatalog,
          providerId,
          input.modelId || modelToSelect,
        );
        resetDialog();
        await onConnected(option);
        return;
      }
      const next = connectPreviewProvider(input);
      setCatalog(next.catalog);
      setConfig(next.config);
      const option = modelOptionFromCatalog(
        next.catalog,
        providerId,
        modelToSelect,
      );
      resetDialog();
      await onConnected(option);
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      setLocalError(message);
      setError(message);
    } finally {
      setSubmitting(false);
    }
  }

  const activeProviderModels =
    selectedProvider && selectedProvider.id !== "custom-api"
      ? (catalogProviders.find(
          (provider) => provider.id === selectedProvider.id,
        )?.models ?? selectedProvider.models)
      : [];
  const activeProviderModelValue =
    selectedModelId ||
    (selectedProvider
      ? catalogDefaultModelForProvider(catalogProviders, selectedProvider.id) ||
        activeProviderModels[0]?.id ||
        ""
      : "");
  const oauthReady = oauthStatus?.status === "success";
  const showModelSelect =
    activeProviderModels.length > 0 &&
    Boolean(
      selectedProvider &&
      (selectedProvider.id !== "openai" ||
        authMode === "api-key" ||
        oauthReady),
    );
  const submitDisabled =
    submitting ||
    !selectedProvider ||
    (selectedProvider.id === "custom-api"
      ? !customProviderFormIsValid(customProviderForm)
      : authMode === "api-key"
        ? !apiKey.trim()
        : false);

  return (
    <>
      <Dialog
        open={open}
        onOpenChange={(nextOpen: boolean) => {
          onOpenChange(nextOpen);
          if (!nextOpen && !selectedProvider) resetDialog();
        }}
      >
        <DialogContent className="sm:max-w-lg">
          <div className="flex flex-col gap-4">
            <DialogHeader>
              <DialogTitle>选择提供商</DialogTitle>
            </DialogHeader>
            <div className="relative">
              <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                aria-label="搜索提供商"
                className="pl-9"
                onChange={(event) => setQuery(event.target.value)}
                placeholder="搜索 provider"
                value={query}
              />
            </div>
            <ScrollArea className="max-h-[min(50vh,24rem)] pr-2 [&_[data-slot=scroll-area-viewport]]:!h-auto [&_[data-slot=scroll-area-viewport]]:max-h-[min(50vh,24rem)]">
              <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
                {filteredProviders.map((provider) => (
                  <button
                    className="flex  items-center gap-2 rounded-lg border bg-background p-2 text-left  transition-colors hover:bg-muted"
                    key={provider.id}
                    onClick={() => selectProvider(provider)}
                    type="button"
                  >
                    <ProviderPickIcon provider={provider} />
                    <span className="min-w-0 truncate">{provider.name}</span>
                  </button>
                ))}
              </div>
              {filteredProviders.length === 0 ? (
                <div className="px-2 py-8 text-center text-sm text-muted-foreground">
                  没有可添加的 provider
                </div>
              ) : null}
            </ScrollArea>
          </div>
        </DialogContent>
      </Dialog>
      <ProviderConnectionDialogs
        activeProvider={selectedProvider}
        apiKey={apiKey}
        authMode={authMode}
        authSuccessMessage={authSuccessMessage}
        callbackInput={callbackInput}
        customProviderForm={customProviderForm}
        error={localError}
        models={activeProviderModels}
        oauthReady={oauthReady}
        oauthStarted={oauthStarted}
        oauthStartResult={oauthStartResult}
        oauthStatus={oauthStatus}
        onApiKeyChange={setApiKey}
        onCallbackInputChange={setCallbackInput}
        onClose={closeProviderDetails}
        onCustomProviderFormChange={setCustomProviderForm}
        onProviderDialogStepChange={setProviderDialogStep}
        onResetAuthMode={resetAuthMode}
        onSelectOpenAIAuthMode={selectOpenAIAuthMode}
        onSelectedModelIdChange={setSelectedModelId}
        onSubmit={submitProvider}
        providerDialogStep={providerDialogStep}
        saving={submitting}
        selectedModelId={activeProviderModelValue}
        showModelSelect={showModelSelect}
        submitDisabled={submitDisabled}
      />
    </>
  );
}

function ProviderPickIcon({ provider }: { provider: ProviderPickOption }) {
  if (provider.iconSrc) {
    return <img alt="" className="size-7 shrink-0" src={provider.iconSrc} />;
  }
  return (
    <span className="grid size-7 shrink-0 place-items-center rounded-full bg-muted text-sm font-semibold text-muted-foreground">
      {provider.name.slice(0, 1)}
    </span>
  );
}

function providerPickerOptions(catalogProviders: ProviderInfo[]) {
  const options = new Map<string, ProviderPickOption>();
  for (const [path, iconSrc] of Object.entries(providerIconModules)) {
    const id =
      path
        .split("/")
        .pop()
        ?.replace(/\.svg$/, "") ?? "";
    if (!id) continue;
    options.set(id, {
      id,
      name: providerDisplayName(id),
      type: providerProtocolForProvider(id),
      baseUrl: providerBaseURLDefaults[id],
      defaultModelId: defaultModelForProvider(id),
      iconSrc,
      models: [],
    });
  }
  for (const provider of catalogProviders) {
    const existing = options.get(provider.id);
    options.set(provider.id, {
      id: provider.id,
      name: provider.name || existing?.name || providerDisplayName(provider.id),
      type:
        provider.type ||
        existing?.type ||
        providerProtocolForProvider(provider.id),
      baseUrl:
        provider.baseUrl ||
        existing?.baseUrl ||
        providerBaseURLDefaults[provider.id],
      defaultModelId:
        provider.defaultModelId ||
        existing?.defaultModelId ||
        defaultModelForProvider(provider.id),
      iconSrc: existing?.iconSrc,
      models: provider.models ?? existing?.models ?? [],
    });
  }
  return Array.from(options.values()).sort((first, second) => {
    if (first.id === "custom-api") return -1;
    if (second.id === "custom-api") return 1;
    return first.name.localeCompare(second.name);
  });
}

function emptyCustomProviderForm(): CustomProviderForm {
  return {
    providerId: "",
    displayName: "",
    protocol: "openai-compatible",
    baseUrl: "",
    apiKey: "",
    models: [emptyCustomRow()],
    headers: [emptyCustomRow()],
  };
}

function customProviderFormFor(
  provider: ProviderPickOption,
): CustomProviderForm {
  if (provider.id !== "custom-api") return emptyCustomProviderForm();
  return {
    ...emptyCustomProviderForm(),
    protocol: providerProtocolForProvider(provider.id),
    baseUrl: provider.baseUrl || providerBaseURLDefaults[provider.id] || "",
    models: [
      {
        ...emptyCustomRow(),
        name: provider.defaultModelId || "custom-profile",
      },
    ],
  };
}

function emptyCustomRow(): CustomProviderRow {
  return { id: crypto.randomUUID(), name: "", value: "" };
}

function customProviderFormIsValid(form: CustomProviderForm) {
  return Boolean(
    form.providerId.trim() &&
    form.displayName.trim() &&
    form.baseUrl.trim() &&
    form.models.some((model) => model.name.trim()),
  );
}

function headersFromRows(rows: CustomProviderRow[]) {
  return Object.fromEntries(
    rows
      .map((row) => [row.name.trim(), row.value.trim()] as const)
      .filter(([name, value]) => name && value),
  );
}

function catalogDefaultModelForProvider(
  catalogProviders: ProviderInfo[],
  providerId: string,
) {
  const provider = catalogProviders.find((item) => item.id === providerId);
  if (!provider) return "";
  if (provider.defaultModelId) return provider.defaultModelId;
  return (
    provider.models.find((model) => model.recommended)?.id ||
    provider.models[0]?.id ||
    ""
  );
}

function providerProtocolForProvider(providerId: string) {
  return providerProtocolDefaults[providerId] ?? "openai-compatible";
}

function defaultModelForProvider(providerId: string) {
  if (providerId === "openai") return "gpt-5.5";
  if (providerId === "anthropic" || providerId === "claude-code")
    return "claude-sonnet-4";
  if (providerId === "google" || providerId === "gemini")
    return "gemini-2.5-pro";
  if (providerId === "openrouter") return "openai/gpt-5-codex";
  return "custom-profile";
}

function providerDisplayName(providerId: string) {
  const knownNames: Record<string, string> = {
    "302ai": "302.AI",
    aihubmix: "AIHubMix",
    "alibaba-cn": "Alibaba CN",
    "alibaba-coding-plan-cn": "Alibaba Coding Plan CN",
    "alibaba-coding-plan": "Alibaba Coding Plan",
    alibaba: "Alibaba",
    anthropic: "Anthropic",
    "amazon-bedrock": "Amazon Bedrock",
    "azure-cognitive-services": "Azure Cognitive Services",
    "cloudflare-ai-gateway": "Cloudflare AI Gateway",
    "cloudflare-workers-ai": "Cloudflare Workers AI",
    "fireworks-ai": "Fireworks AI",
    "github-copilot": "GitHub Copilot",
    "github-models": "GitHub Models",
    "google-vertex-anthropic": "Google Vertex Anthropic",
    "google-vertex": "Google Vertex",
    google: "Google",
    "io-net": "io.net",
    lmstudio: "LM Studio",
    "minimax-cn-coding-plan": "MiniMax CN Coding Plan",
    "minimax-cn": "MiniMax CN",
    "minimax-coding-plan": "MiniMax Coding Plan",
    minimax: "MiniMax",
    "moonshotai-cn": "Moonshot AI CN",
    moonshotai: "Moonshot AI",
    "nano-gpt": "Nano GPT",
    "novita-ai": "Novita AI",
    "ollama-cloud": "Ollama Cloud",
    "opencode-go": "opencode Go",
    opencode: "opencode",
    openai: "OpenAI",
    openrouter: "OpenRouter",
    "perplexity-agent": "Perplexity Agent",
    "qiniu-ai": "Qiniu AI",
    "siliconflow-cn": "SiliconFlow CN",
    siliconflow: "SiliconFlow",
    stackit: "STACKIT",
    synthetic: "Synthetic",
    "tencent-coding-plan": "Tencent Coding Plan",
    togetherai: "Together AI",
    v0: "v0",
    wandb: "Weights & Biases",
  };
  if (knownNames[providerId]) return knownNames[providerId];
  return providerId
    .split("-")
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}

function defaultModelFromCatalog(catalog: CatalogState, providerId: string) {
  const provider = catalog.providers.find((item) => item.id === providerId);
  if (!provider) return "";
  return (
    provider.defaultModelId ||
    provider.models.find((model) => model.recommended)?.id ||
    provider.models[0]?.id ||
    ""
  );
}

function modelOptionFromCatalog(
  catalog: CatalogState,
  providerId: string,
  modelId: string,
) {
  const provider = catalog.providers.find((item) => item.id === providerId);
  if (!provider) return null;
  const model =
    provider.models.find((item) => item.id === modelId) ?? provider.models[0];
  if (!model) return null;
  return {
    ...model,
    id: normalizeModelId(provider.id, model.id),
    providerId: provider.id,
    providerName: provider.name,
  };
}

async function openExternalURL(url: string) {
  await BrowserOpenURL(url);
}

function delay(ms: number) {
  return new Promise((resolve) => window.setTimeout(resolve, ms));
}

function useAutoTextareaHeight(
  value: string,
  minHeight: number,
  maxHeight: number,
  textareaRef: React.RefObject<HTMLTextAreaElement | null>,
) {
  const [height, setHeight] = useState({
    content: 0,
    viewport: 0,
  });

  useLayoutEffect(() => {
    const textarea = textareaRef.current;
    if (!textarea) return;

    textarea.style.height = "";
    const baseHeight = textarea.offsetHeight;
    textarea.style.height = "0px";
    const measuredHeight = Math.max(textarea.scrollHeight, minHeight);
    const contentHeight = measuredHeight > baseHeight ? measuredHeight : 0;
    const viewportHeight =
      contentHeight > 0 ? Math.min(contentHeight, maxHeight) : 0;

    textarea.style.height = contentHeight > 0 ? `${contentHeight}px` : "";
    setHeight((current) => {
      if (
        current.content === contentHeight &&
        current.viewport === viewportHeight
      )
        return current;
      return {
        content: contentHeight,
        viewport: viewportHeight,
      };
    });
  }, [maxHeight, minHeight, textareaRef, value]);

  return height;
}

function ConversationSidebar({
  activeConversationId,
  activeProjectPage,
  conversations,
  onNewConversation,
  onNewProjectConversation,
  onArchiveConversation,
  onHideProject,
  onSelectConversation,
  onTogglePinnedConversation,
  pendingPermissionCountsBySessionId,
  pinnedConversationIds,
  projectGroups,
  runningConversationIds,
  selectedProjectPath,
  topBar,
}: {
  activeConversationId: string;
  activeProjectPage: "chat" | "plugins";
  conversations: domain.Session[];
  onNewConversation: () => void;
  onNewProjectConversation: (projectPath: string) => void;
  onArchiveConversation: (sessionId: string) => void;
  onHideProject: (projectPath: string) => void;
  onSelectConversation: (session: domain.Session) => void;
  onTogglePinnedConversation: (sessionId: string) => void;
  pendingPermissionCountsBySessionId: Record<string, number>;
  pinnedConversationIds: string[];
  projectGroups: ProjectConversationGroup[];
  runningConversationIds: string[];
  selectedProjectPath: string;
  topBar: React.ReactNode;
}) {
  const [conversationsCollapsed, setConversationsCollapsed] = useState(false);
  const [projectsCollapsed, setProjectsCollapsed] = useState(false);
  const [collapsedProjectPaths, setCollapsedProjectPaths] = useState<string[]>(
    [],
  );
  const pinnedConversationIdSet = useMemo(
    () => new Set(pinnedConversationIds),
    [pinnedConversationIds],
  );
  const runningConversationIdSet = useMemo(
    () => new Set(runningConversationIds),
    [runningConversationIds],
  );
  const pinnedConversations = conversations.filter((conversation) =>
    pinnedConversationIdSet.has(conversation.id),
  );
  const regularConversations = conversations.filter(
    (conversation) => !pinnedConversationIdSet.has(conversation.id),
  );
  const collapsedProjectPathSet = useMemo(
    () => new Set(collapsedProjectPaths),
    [collapsedProjectPaths],
  );

  function toggleProjectCollapsed(projectPath: string) {
    setCollapsedProjectPaths((currentPaths) =>
      currentPaths.includes(projectPath)
        ? currentPaths.filter((path) => path !== projectPath)
        : [projectPath, ...currentPaths],
    );
  }

  return (
    <>
      <SidebarHeader className="h-9 shrink-0 p-0 text-base">
        {topBar}
      </SidebarHeader>

      <SidebarContent className="h-0 overflow-hidden px-0 pb-0">
        <SidebarGroup className="shrink-0 p-0 px-1.5 group-data-[collapsible=icon]:px-2">
          <SidebarMenu className="gap-2">
            <SidebarMenuItem>
              <SidebarMenuButton
                className="h-7 gap-2.5 px-3 text-sm text-sidebar-foreground group-data-[collapsible=icon]:justify-center group-data-[collapsible=icon]:px-2"
                onClick={onNewConversation}
                tooltip="新对话"
                type="button"
              >
                <SquarePen className="size-3.5!" />
                <span>新对话</span>
              </SidebarMenuButton>
            </SidebarMenuItem>
            <SidebarMenuItem>
              <SidebarMenuButton
                asChild
                className="h-7 gap-2.5 px-3 text-sm text-sidebar-foreground group-data-[collapsible=icon]:justify-center group-data-[collapsible=icon]:px-2"
                isActive={activeProjectPage === "plugins"}
                tooltip="插件"
              >
                <Link to="/projects/plugins">
                  <Plug className="size-3.5!" />
                  <span>插件</span>
                </Link>
              </SidebarMenuButton>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarGroup>

        {pinnedConversations.length > 0 && (
          <SidebarGroup className="mt-3 shrink-0 p-0">
            <SidebarGroupLabel className="mx-1.5 flex h-6 items-center px-3 text-sm font-semibold text-muted-foreground group-data-[collapsible=icon]:mx-2">
              <span className="min-w-0 truncate">置顶</span>
            </SidebarGroupLabel>
            <SidebarGroupContent className="group-data-[collapsible=icon]:hidden">
              <SidebarMenu className="min-w-0 gap-0.5 px-3">
                {pinnedConversations.map((conversation) => (
                  <ConversationSidebarItem
                    activeConversationId={activeConversationId}
                    conversation={conversation}
                    isPinned
                    key={conversation.id}
                    onArchiveConversation={onArchiveConversation}
                    onSelectConversation={onSelectConversation}
                    onTogglePinnedConversation={onTogglePinnedConversation}
                    pendingPermissionCount={
                      pendingPermissionCountsBySessionId[conversation.id] ?? 0
                    }
                    isRunning={runningConversationIdSet.has(conversation.id)}
                  />
                ))}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        )}

        <ScrollArea className="min-h-0 flex-1 [&_[data-slot=scroll-area-viewport]]:overflow-x-hidden [&_[data-slot=scroll-area-viewport]>div]:!block [&_[data-slot=scroll-area-viewport]>div]:!min-w-0">
          {projectGroups.length > 0 && (
            <SidebarGroup className="mt-3 shrink-0 p-0">
              <SidebarSectionHeader
                collapsed={projectsCollapsed}
                label="项目"
                moreLabel="更多项目操作"
                onToggle={() =>
                  setProjectsCollapsed((collapsed) => !collapsed)
                }
              />
              <SidebarGroupContent
                aria-hidden={projectsCollapsed}
                className={cn(
                  "grid origin-top overflow-hidden transition-[grid-template-rows,opacity,transform] duration-200 ease-out group-data-[collapsible=icon]:hidden",
                  projectsCollapsed
                    ? "pointer-events-none grid-rows-[0fr] -translate-y-1 opacity-0"
                    : "grid-rows-[1fr] translate-y-0 opacity-100",
                )}
              >
                <div className="min-h-0 overflow-hidden">
                  <SidebarMenu className="min-w-0 gap-1 px-1.5">
                    {projectGroups.map((group) => {
                      const projectCollapsed = collapsedProjectPathSet.has(
                        group.projectPath,
                      );
                      const projectActive =
                        group.conversations.some(
                          (conversation) =>
                            conversation.id === activeConversationId,
                        ) ||
                        (!activeConversationId &&
                          normalizeProjectPathKey(selectedProjectPath) ===
                            normalizeProjectPathKey(group.projectPath));
                      return (
                        <ProjectSidebarItem
                          activeConversationId={activeConversationId}
                          collapsed={projectCollapsed}
                          group={group}
                          isActive={projectActive}
                          isPinnedConversation={(sessionId) =>
                            pinnedConversationIdSet.has(sessionId)
                          }
                          isRunningConversation={(sessionId) =>
                            runningConversationIdSet.has(sessionId)
                          }
                          key={group.projectPath}
                          onArchiveConversation={onArchiveConversation}
                          onHideProject={onHideProject}
                          onNewProjectConversation={onNewProjectConversation}
                          onSelectConversation={onSelectConversation}
                          onToggleCollapsed={() =>
                            toggleProjectCollapsed(group.projectPath)
                          }
                          onTogglePinnedConversation={onTogglePinnedConversation}
                          pendingPermissionCountsBySessionId={
                            pendingPermissionCountsBySessionId
                          }
                        />
                      );
                    })}
                  </SidebarMenu>
                </div>
              </SidebarGroupContent>
            </SidebarGroup>
          )}

          <SidebarGroup className="mt-3 shrink-0 p-0">
            <SidebarSectionHeader
              collapsed={conversationsCollapsed}
              label="对话"
              moreLabel="更多对话操作"
              onToggle={() =>
                setConversationsCollapsed((collapsed) => !collapsed)
              }
            />
            <SidebarGroupContent
              aria-hidden={conversationsCollapsed}
              className={cn(
                "grid origin-top overflow-hidden transition-[grid-template-rows,opacity,transform] duration-200 ease-out group-data-[collapsible=icon]:hidden",
                conversationsCollapsed
                  ? "pointer-events-none grid-rows-[0fr] -translate-y-1 opacity-0"
                  : "grid-rows-[1fr] translate-y-0 opacity-100",
              )}
            >
              <div className="min-h-0 overflow-hidden">
                <SidebarMenu className="min-w-0 gap-0.5 px-3">
                  {regularConversations.map((conversation) => (
                    <ConversationSidebarItem
                      activeConversationId={activeConversationId}
                      conversation={conversation}
                      isPinned={false}
                      key={conversation.id}
                      onArchiveConversation={onArchiveConversation}
                      onSelectConversation={onSelectConversation}
                      onTogglePinnedConversation={onTogglePinnedConversation}
                      pendingPermissionCount={
                        pendingPermissionCountsBySessionId[conversation.id] ?? 0
                      }
                      isRunning={runningConversationIdSet.has(conversation.id)}
                    />
                  ))}
                </SidebarMenu>
              </div>
            </SidebarGroupContent>
          </SidebarGroup>
        </ScrollArea>
      </SidebarContent>

        <SidebarFooter className="flex-row items-center justify-between px-1.5 pb-5 pt-3 group-data-[collapsible=icon]:px-2">
          <SidebarMenu className="min-w-0 flex-1">
            <SidebarMenuItem>
              <Button
                asChild
                size={"lg"}
                variant="ghost"
              >
                <Link to="/settings">
                  <Settings />
                  <span>设置</span>
                </Link>
              </Button>
            </SidebarMenuItem>
          </SidebarMenu>
          <Button
            aria-label="移动设备"
            className={cn(
              "text-muted-foreground group-data-[collapsible=icon]:hidden",
            )}
            size="icon"
            type="button"
            variant="ghost"
          >
            <Smartphone />
          </Button>
        </SidebarFooter>
    </>
  );
}

function SidebarSectionHeader({
  collapsed,
  label,
  moreLabel,
  onToggle,
}: {
  collapsed: boolean;
  label: string;
  moreLabel: string;
  onToggle: () => void;
}) {
  return (
    <SidebarGroupLabel
      aria-expanded={!collapsed}
      className="group/sidebar-section-heading mx-1.5 flex h-6 cursor-pointer items-center gap-1 px-3 text-sm font-semibold text-muted-foreground group-data-[collapsible=icon]:mx-2"
      onClick={onToggle}
      role="button"
      tabIndex={0}
      onKeyDown={(event) => {
        if (event.key === "Enter" || event.key === " ") {
          event.preventDefault();
          onToggle();
        }
      }}
    >
      <span className="min-w-0 truncate">{label}</span>
      <button
        aria-label={collapsed ? `展开${label}` : `收起${label}`}
        className="inline-flex size-5 shrink-0 items-center justify-center rounded-md opacity-0 transition-opacity hover:bg-sidebar-accent hover:text-sidebar-accent-foreground group-hover/sidebar-section-heading:opacity-100 group-focus-within/sidebar-section-heading:opacity-100"
        onClick={(event) => {
          event.stopPropagation();
          onToggle();
        }}
        type="button"
      >
        <ChevronDown
          className={cn(
            "size-4 transition-transform duration-200",
            collapsed && "-rotate-90",
          )}
        />
      </button>
      <span className="min-w-0 flex-1" />
      <button
        aria-label={moreLabel}
        className="inline-flex size-5 shrink-0 items-center justify-center rounded-md opacity-0 transition-opacity hover:bg-sidebar-accent hover:text-sidebar-accent-foreground group-hover/sidebar-section-heading:opacity-100 group-focus-within/sidebar-section-heading:opacity-100"
        onClick={(event) => event.stopPropagation()}
        type="button"
      >
        <Ellipsis className="size-4" />
      </button>
    </SidebarGroupLabel>
  );
}

function ProjectSidebarItem({
  activeConversationId,
  collapsed,
  group,
  isActive,
  isPinnedConversation,
  isRunningConversation,
  onArchiveConversation,
  onHideProject,
  onNewProjectConversation,
  onSelectConversation,
  onToggleCollapsed,
  onTogglePinnedConversation,
  pendingPermissionCountsBySessionId,
}: {
  activeConversationId: string;
  collapsed: boolean;
  group: ProjectConversationGroup;
  isActive: boolean;
  isPinnedConversation: (sessionId: string) => boolean;
  isRunningConversation: (sessionId: string) => boolean;
  onArchiveConversation: (sessionId: string) => void;
  onHideProject: (projectPath: string) => void;
  onNewProjectConversation: (projectPath: string) => void;
  onSelectConversation: (session: domain.Session) => void;
  onToggleCollapsed: () => void;
  onTogglePinnedConversation: (sessionId: string) => void;
  pendingPermissionCountsBySessionId: Record<string, number>;
}) {
  const projectName =
    group.project.name || projectNameFromPath(group.projectPath);
  const conversationCount = group.conversations.length;

  return (
    <SidebarMenuItem className="min-w-0">
      <div className="group/project-row relative min-w-0">
        <SidebarMenuButton
          aria-expanded={!collapsed}
          className="min-w-0 pr-14 text-sidebar-foreground"
          isActive={isActive}
          onClick={onToggleCollapsed}
          title={group.projectPath}
          type="button"
        >
          <FileText />
          <span className="flex min-w-0 items-center gap-1">
            <span className="min-w-0 truncate">{projectName}</span>
            <span className="relative inline-flex size-4 shrink-0 items-center justify-center text-muted-foreground">
              {conversationCount > 0 ? (
                <span className="text-xs leading-none transition-opacity duration-200 group-hover/project-row:opacity-0 group-focus-within/project-row:opacity-0">
                  {conversationCount}
                </span>
              ) : null}
              <ChevronDown
                className={cn(
                  "absolute size-4 opacity-0 transition-[opacity,transform] duration-200 group-hover/project-row:opacity-100 group-focus-within/project-row:opacity-100",
                  collapsed && "-rotate-90",
                )}
              />
            </span>
          </span>
        </SidebarMenuButton>
        <span className="pointer-events-none absolute right-1 top-1/2 flex -translate-y-1/2 items-center gap-0.5 opacity-0 transition-opacity group-hover/project-row:pointer-events-auto group-hover/project-row:opacity-100 group-focus-within/project-row:pointer-events-auto group-focus-within/project-row:opacity-100">
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <button
                aria-label="更多项目操作"
                className="inline-flex size-6 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground"
                onClick={(event) => event.stopPropagation()}
                type="button"
              >
                <Ellipsis className="size-4" />
              </button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuLabel className="max-w-64 truncate">
                {projectName}
              </DropdownMenuLabel>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                onSelect={() => onNewProjectConversation(group.projectPath)}
              >
                <Pencil />
                新项目对话
              </DropdownMenuItem>
              <DropdownMenuItem
                onSelect={() => {
                  void window.aivo?.openPath(group.projectPath).catch((error) => {
                    toast.error(
                      error instanceof Error ? error.message : "打开项目失败",
                    );
                  });
                }}
              >
                <FolderOpen />
                打开目录
              </DropdownMenuItem>
              <DropdownMenuItem
                onSelect={() => {
                  void navigator.clipboard
                    ?.writeText(group.projectPath)
                    .then(() => toast.success("已复制项目路径"))
                    .catch(() => toast.error("复制项目路径失败"));
                }}
              >
                <Copy />
                复制路径
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem onSelect={() => onHideProject(group.projectPath)}>
                <X />
                从侧边栏移除
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
          <button
            aria-label="打开项目新对话"
            className="inline-flex size-6 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground"
            onClick={(event) => {
              event.preventDefault();
              event.stopPropagation();
              onNewProjectConversation(group.projectPath);
            }}
            type="button"
          >
            <Pencil className="size-4" />
          </button>
        </span>
      </div>
      {!collapsed && group.conversations.length > 0 ? (
        <SidebarMenu className="mt-1 min-w-0 gap-0.5 px-3">
          {group.conversations.map((conversation) => (
            <ConversationSidebarItem
              activeConversationId={activeConversationId}
              conversation={conversation}
              isPinned={isPinnedConversation(conversation.id)}
              isRunning={isRunningConversation(conversation.id)}
              key={conversation.id}
              onArchiveConversation={onArchiveConversation}
              onSelectConversation={onSelectConversation}
              onTogglePinnedConversation={onTogglePinnedConversation}
              pendingPermissionCount={
                pendingPermissionCountsBySessionId[conversation.id] ?? 0
              }
            />
          ))}
        </SidebarMenu>
      ) : null}
    </SidebarMenuItem>
  );
}

function ConversationSidebarItem({
  activeConversationId,
  conversation,
  isPinned,
  onArchiveConversation,
  onSelectConversation,
  onTogglePinnedConversation,
  pendingPermissionCount,
  isRunning,
}: {
  activeConversationId: string;
  conversation: domain.Session;
  isPinned: boolean;
  onArchiveConversation: (sessionId: string) => void;
  onSelectConversation: (session: domain.Session) => void;
  onTogglePinnedConversation: (sessionId: string) => void;
  pendingPermissionCount: number;
  isRunning: boolean;
}) {
  const isActive = conversation.id === activeConversationId;
  const hasPendingPermission = pendingPermissionCount > 0;

  return (
    <SidebarMenuItem
      className="group/conversation-item relative min-w-0 "
      key={conversation.id}
    >
      <SidebarMenuButton
        aria-current={isActive ? "page" : undefined}
        className="min-w-0 justify-between gap-2 rounded-md px-1.5 text-sidebar-foreground transition-colors"
        isActive={isActive}
        onClick={() => onSelectConversation(conversation)}
        type="button"
      >
        <AnimatedTitle
          className="min-w-0 flex-1 text-xs leading-5"
          value={conversation.title}
        />
        {hasPendingPermission ? (
          <span className="shrink-0 rounded-full bg-primary px-2 py-0.5 text-[11px] font-semibold leading-none text-primary-foreground shadow-sm shadow-primary/20">
            {pendingPermissionCount > 1
              ? `待批准 ${pendingPermissionCount}`
              : "待批准"}
          </span>
        ) : isRunning ? (
          <span className="shrink-0 transition-opacity group-hover/conversation-item:opacity-0 group-focus-within/conversation-item:opacity-0">
            <Spinner
              className={cn(
                "size-3.5 text-muted-foreground",
                isActive && "text-sidebar-accent-foreground/70",
              )}
            />
          </span>
        ) : (
          <span
            className={cn(
              "shrink-0 text-xs  text-muted-foreground transition-[opacity,color] group-hover/conversation-item:opacity-0 group-focus-within/conversation-item:opacity-0",
              isActive && "text-sidebar-accent-foreground/70",
            )}
          >
            {relativeTime(conversation.timeUpdated || conversation.timeCreated)}
          </span>
        )}
      </SidebarMenuButton>
      {!hasPendingPermission && (
        <span className="pointer-events-none absolute right-2 top-1/2 flex -translate-y-1/2 items-center gap-0.5 opacity-0 transition-opacity group-hover/conversation-item:pointer-events-auto group-hover/conversation-item:opacity-100 group-focus-within/conversation-item:pointer-events-auto group-focus-within/conversation-item:opacity-100">
          <button
            aria-label={isPinned ? "取消置顶" : "置顶对话"}
            className={cn(
              "inline-flex size-5 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground",
              isPinned &&
                "bg-sidebar-accent text-muted-foreground/80 hover:text-muted-foreground",
            )}
            onClick={(event) => {
              event.preventDefault();
              event.stopPropagation();
              onTogglePinnedConversation(conversation.id);
            }}
            title={isPinned ? "取消置顶" : "置顶对话"}
            type="button"
          >
            <Pin className={cn("size-3.5", isPinned && "fill-current")} />
          </button>
          <button
            aria-label="归档对话"
            className="inline-flex size-5 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground"
            onClick={(event) => {
              event.preventDefault();
              event.stopPropagation();
              onArchiveConversation(conversation.id);
            }}
            title="归档对话"
            type="button"
          >
            <Archive className="size-3.5" />
          </button>
        </span>
      )}
    </SidebarMenuItem>
  );
}

function ProjectTopBar({
  leftSidebarState,
  onNewPage,
  onToggleLeftSidebar,
}: {
  canShowTerminalPanel: boolean;
  conversationTitle: string;
  hasConversation: boolean;
  leftSidebarState: "expanded" | "collapsed";
  onNewPage: () => void;
  onToggleLeftSidebar: () => void;
  pageIcon?: React.ReactNode;
  pageTitle?: string;
  showTerminalButton?: boolean;
}) {
  const isMac = window.aivo?.platform === "darwin";
  const leftCompactWidth = isMac ? 202 : 148;

  return (
    <header className="pointer-events-none relative flex h-full min-w-0 text-foreground">
      <div
        className={cn(
          "pointer-events-auto relative flex h-full shrink-0 items-center overflow-hidden text-sidebar-foreground transition-[width] duration-[var(--project-panel-transition-duration,200ms)] ease-linear",
          leftSidebarState === "collapsed" && "border-b border-border/60",
        )}
        style={{
          width:
            leftSidebarState === "expanded"
              ? "var(--project-left-sidebar-width)"
              : `${leftCompactWidth}px`,
        }}
      >
        <ProjectWindowControls isMac={isMac} />
        <div
          className="pointer-events-auto flex h-full shrink-0 items-center gap-1 px-3"
          data-app-no-drag
        >
          <ProjectTopBarIconButton
            aria-label="展开或收起侧边栏"
            onClick={onToggleLeftSidebar}
          >
            <PanelLeft />
          </ProjectTopBarIconButton>
          <ProjectTopBarIconButton aria-label="返回" onClick={() => undefined}>
            <ArrowLeft />
          </ProjectTopBarIconButton>
          <ProjectTopBarIconButton aria-label="前进" onClick={() => undefined}>
            <ArrowRight />
          </ProjectTopBarIconButton>
          <ProjectTopBarIconButton aria-label="新建页面" onClick={onNewPage}>
            <SquarePen />
          </ProjectTopBarIconButton>
        </div>
        <div className="h-full min-w-0 flex-1" data-app-drag />
      </div>
    </header>
  );
}

function ProjectMainTopBar({
  conversationTitle,
  hasConversation,
  isLayoutPanelOpen,
  onToggleLayoutPanel,
  pageIcon,
  pageTitle,
  rightOpen,
  showLayoutButton,
  showTerminalButton,
}: {
  conversationTitle: string;
  hasConversation: boolean;
  isLayoutPanelOpen?: boolean;
  onToggleLayoutPanel?: () => void;
  pageIcon?: React.ReactNode;
  pageTitle?: string;
  rightOpen?: boolean;
  showLayoutButton?: boolean;
  showTerminalButton?: boolean;
}) {
  const title = pageTitle || (hasConversation ? conversationTitle : "");
  const floatingActionsInset = showTerminalButton ? 76 : 40;
  const layoutActionRight = rightOpen ? 0 : floatingActionsInset;
  const actionsInset = showLayoutButton
    ? `${layoutActionRight + 44}px`
    : `${layoutActionRight}px`;

  return (
    <div className="pointer-events-auto relative flex h-full min-w-0 flex-1 border-b border-border/60 bg-background/80 text-foreground shadow-sm shadow-background/30 backdrop-blur-xl supports-[backdrop-filter]:bg-background/65">
      <div
        className="flex min-w-0 flex-1 items-center gap-2 ps-3"
        style={{ paddingRight: actionsInset }}
      >
        {title ? (
          <>
            <div className="flex min-w-0 items-center gap-2" data-app-drag>
              {pageIcon ?? (
                <FileText
                  aria-hidden="true"
                  className="size-4 shrink-0 text-muted-foreground"
                />
              )}
              <AnimatedTitle
                className="min-w-0 text-sm font-semibold text-foreground"
                value={title.trim() || "未命名会话"}
              />
            </div>
            {hasConversation ? (
              <span data-app-no-drag>
                <ProjectTopBarIconButton aria-label="更多会话操作">
                  <Ellipsis />
                </ProjectTopBarIconButton>
              </span>
            ) : null}
          </>
        ) : null}
        <div className="h-full min-w-0 flex-1" data-app-drag />
      </div>
      <ProjectMainTopBarActions
        isLayoutPanelOpen={isLayoutPanelOpen}
        onToggleLayoutPanel={onToggleLayoutPanel}
        right={layoutActionRight}
        showLayoutButton={showLayoutButton}
      />
    </div>
  );
}

function ProjectMainTopBarActions({
  isLayoutPanelOpen,
  onToggleLayoutPanel,
  right,
  showLayoutButton,
}: {
  isLayoutPanelOpen?: boolean;
  onToggleLayoutPanel?: () => void;
  right: number;
  showLayoutButton?: boolean;
}) {
  return (
    <div
      className="pointer-events-auto absolute right-0 top-0 z-[60] flex h-9 shrink-0 items-center justify-end gap-2 pe-3 text-foreground"
      data-app-no-drag
      style={{ right }}
    >
      {showLayoutButton ? (
        <ProjectTopBarIconButton
          aria-label="切换系统环境"
          aria-pressed={isLayoutPanelOpen}
          className={cn(isLayoutPanelOpen && "bg-muted text-foreground")}
          onClick={onToggleLayoutPanel}
        >
          <LayoutGrid />
        </ProjectTopBarIconButton>
      ) : null}
    </div>
  );
}

function ProjectFloatingRightTopBarActions({
  isRightSidebarMaximized,
  onToggleRightSidebarMaximized,
  rightOpen,
  showTerminalButton,
}: {
  isRightSidebarMaximized: boolean;
  onToggleRightSidebarMaximized: () => void;
  rightOpen: boolean;
  showTerminalButton?: boolean;
}) {
  const { toggleSidebar: toggleRightSidebar } = useSidebar();

  return (
    <div
      className="pointer-events-auto absolute right-0 top-0 z-[80] flex h-9 shrink-0 items-center justify-end gap-2 pe-3 text-foreground"
      data-app-no-drag
    >
      {rightOpen ? (
        <ProjectTopBarIconButton
          aria-label={isRightSidebarMaximized ? "恢复右侧栏宽度" : "全屏右侧栏"}
          aria-pressed={isRightSidebarMaximized}
          onClick={onToggleRightSidebarMaximized}
          title={isRightSidebarMaximized ? "恢复右侧栏宽度" : "全屏右侧栏"}
        >
          {isRightSidebarMaximized ? <Minimize2 /> : <Maximize2 />}
        </ProjectTopBarIconButton>
      ) : null}
      {showTerminalButton ? (
        <TerminalDockTrigger
          aria-label="打开终端面板"
          className="text-muted-foreground"
        >
          <PanelBottom />
        </TerminalDockTrigger>
      ) : null}
      <ProjectTopBarIconButton
        aria-label="打开或关闭侧边面板"
        onClick={toggleRightSidebar}
      >
        <PanelRight />
      </ProjectTopBarIconButton>
    </div>
  );
}

function ProjectWindowControls({ isMac }: { isMac: boolean }) {
  return (
    <div
      aria-hidden="true"
      className={cn("shrink-0", isMac ? "w-[54px]" : "w-0")}
      data-app-no-drag
    />
  );
}

function ProjectTopBarIconButton({
  className,
  ...props
}: ComponentProps<typeof Button>) {
  return (
    <Button
      className={cn("text-muted-foreground", className)}
      size="icon"
      type="button"
      variant="ghost"
      {...props}
    />
  );
}
