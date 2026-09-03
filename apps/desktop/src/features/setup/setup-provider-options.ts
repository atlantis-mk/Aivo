import anthropicIcon from "@/assets/icons/provider/anthropic.svg";
import googleIcon from "@/assets/icons/provider/google.svg";
import openAIIcon from "@/assets/icons/provider/openai.svg";
import syntheticIcon from "@/assets/icons/provider/synthetic.svg";
import { providerDisplayName } from "@/features/providers/provider-defaults";
import type { ProviderChoice } from "@/features/providers/provider-types";
import {
  AiComputerIcon,
  File01Icon,
  Grid2X2Icon,
  Search01Icon,
  TextAlignLeft01Icon,
} from "@hugeicons/core-free-icons";

export const welcomeCapabilities = [
  { icon: File01Icon, label: "整理文件" },
  { icon: TextAlignLeft01Icon, label: "浏览与总结" },
  { icon: AiComputerIcon, label: "操作电脑" },
  { icon: Grid2X2Icon, label: "使用应用" },
  { icon: Search01Icon, label: "搜集资料" },
];

const providerIconModules = import.meta.glob<string>(
  "@/assets/icons/provider/*.svg",
  {
    eager: true,
    import: "default",
    query: "?url",
  },
);

const primaryProviderIds = new Set(["openai", "anthropic", "google", "synthetic"]);

export const providerChoices: ProviderChoice[] = [
  {
    id: "openai",
    name: "OpenAI",
    iconSrc: openAIIcon,
  },
  {
    id: "claude-code",
    name: "Claude Code",
    iconSrc: anthropicIcon,
  },
  {
    id: "gemini",
    name: "Gemini",
    iconSrc: googleIcon,
  },
  {
    id: "volcengine-agent-plan",
    name: "火山方舟 Agent Plan",
    iconSrc: syntheticIcon,
  },
  {
    id: "other",
    name: "其他",
    iconSrc: syntheticIcon,
    opensProviderPicker: true,
  },
  {
    id: "custom-api",
    name: "Custom API",
    iconSrc: syntheticIcon,
  },
];

export const otherProviderChoices: ProviderChoice[] = Object.entries(
  providerIconModules,
)
  .map(([path, iconSrc]) => {
    const id = path.split("/").pop()?.replace(/\.svg$/, "") ?? "";
    return {
      id,
      name: providerDisplayName(id),
      iconSrc,
      custom: true,
    };
  })
  .filter((provider) => provider.id && !primaryProviderIds.has(provider.id))
  .sort((first, second) => first.name.localeCompare(second.name));
