import anthropicIcon from "@/assets/icons/provider/anthropic.svg";
import googleIcon from "@/assets/icons/provider/google.svg";
import openAIIcon from "@/assets/icons/provider/openai.svg";
import syntheticIcon from "@/assets/icons/provider/synthetic.svg";
import { providerDisplayName } from "@/features/providers/provider-defaults";
import type { ProviderChoice } from "@/features/providers/provider-types";

export const capabilityPills = [
  "我能帮你整理文件",
  "我能帮你浏览总结",
  "我可以帮你使用电脑",
  "我能帮你操作 App",
  "我能搜集全网信息",
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
