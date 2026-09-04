import { useState } from "react";
import { ChevronDown } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  groupModelOptionsByProvider,
  modelOptionKey,
  reasoningEffortLabel,
  serviceTierLabel,
  type ModelOption,
} from "@/features/projects/project-model-options";
import { compactModelLabel } from "@/features/projects/project-model-settings-menu-model";
import {
  ModelPickerSubmenu,
  ReasoningEffortMenuItems,
  ServiceTierSubmenu,
} from "@/features/projects/project-model-settings-menu-sections";
import { ProviderConnectDialog } from "@/features/projects/project-provider-connect-dialog";
import { useAppConfig } from "@/lib/app-config";
import type { ModelInfo } from "@/lib/provider-catalog";
import { getProviderCatalogForProject } from "@/services/aivo";

export function ModelSettingsMenu({
  compact,
  allModelOptions,
  modelId,
  modelLabel,
  modelOptions,
  onModelSelect,
  onReasoningEffortSelect,
  onServiceTierSelect,
  projectPath,
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
  projectPath: string;
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
  const activeModel =
    allModelOptions.find(
      (model) =>
        modelOptionKey(model.providerId, model.id) === activeModelKey,
    ) ?? modelOptions.find((model) => model.id === modelId);

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
          <ReasoningEffortMenuItems
            model={activeModel}
            onReasoningEffortSelect={onReasoningEffortSelect}
            reasoningEffort={reasoningEffort}
          />
          <ModelPickerSubmenu
            activeModelKey={activeModelKey}
            groupedModels={groupedModels}
            modelId={modelId}
            modelLabel={modelLabel}
            modelOptions={modelOptions}
            onAddProvider={() => {
              setMenuOpen(false);
              setConnectDialogOpen(true);
            }}
            onModelSelect={onModelSelect}
            query={query}
            setQuery={setQuery}
          />
          {showServiceTier ? (
            <ServiceTierSubmenu
              model={activeModel}
              onServiceTierSelect={onServiceTierSelect}
              serviceTier={serviceTier}
            />
          ) : null}
        </DropdownMenuContent>
      </DropdownMenu>
      <ProviderConnectDialog
        catalogProviders={catalog?.providers ?? []}
        onConnected={async (option) => {
          if (option) onModelSelect(option);
          if (projectPath) {
            setCatalog(await getProviderCatalogForProject(projectPath));
          } else {
            await reload();
          }
        }}
        onOpenChange={setConnectDialogOpen}
        open={connectDialogOpen}
        projectPath={projectPath}
        setCatalog={setCatalog}
        setConfig={setConfig}
        setError={setError}
      />
    </>
  );
}
