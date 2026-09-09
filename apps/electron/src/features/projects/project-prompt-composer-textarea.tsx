import { useLayoutEffect, useRef, useState, useId } from "react";
import { NodeSelection, type Command } from "prosemirror-state";
import { EditorView } from "prosemirror-view";
import { closeHistory } from "prosemirror-history";
import { setBlockType, toggleMark, wrapIn } from "prosemirror-commands";
import { wrapInList } from "prosemirror-schema-list";
import {
  ArrowRight01Icon,
  Cancel01Icon,
  File02Icon,
} from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { toast } from "sonner";
import {
  Attachment,
  AttachmentAction,
  AttachmentActions,
  AttachmentContent,
  AttachmentGroup,
  AttachmentMedia,
  AttachmentTitle,
} from "@/components/ui/attachment";
import type {
  PromptMentionAction,
  PromptMentionReference,
} from "@/features/projects/project-prompt-mention-model";
import {
  inferPromptPasteName,
  isExternalPromptPasteTarget,
  promptPasteTitle,
  promptPasteTarget,
  type PendingPromptPaste,
} from "@/features/projects/project-prompt-paste";
import { PromptMentionPicker } from "@/features/projects/project-prompt-mention-picker";
import type {
  AutoTextareaHeightRef,
  PromptComposerProps,
} from "@/features/projects/project-prompt-composer-types";
import {
  createPromptEditorState,
  editorMentionQuery,
  insertPromptMention,
  parsePromptDocument,
  promptDocumentReferences,
  promptEditorSchema,
  promptPasteSlice,
  resolveMentionNodes,
  serializePromptDocument,
} from "./project-prompt-editor-model";
import { Slice } from "prosemirror-model";
import { cn } from "@/lib/utils";

type PromptComposerTextareaProps = Pick<
  PromptComposerProps,
  | "onAddAttachments"
  | "onPromptChange"
  | "onPromptMentionRemove"
  | "onPromptMentionSelect"
  | "onPromptPaste"
  | "onPromptPasteRemove"
  | "onSubmit"
  | "pendingPromptPastes"
  | "prompt"
  | "promptResourceReferences"
  | "projectPath"
  | "projects"
> & {
  onSelectLocalResource: () => Promise<void>;
  textareaRef: AutoTextareaHeightRef;
};

export function PromptComposerTextarea({
  onAddAttachments,
  onPromptChange,
  onPromptMentionRemove,
  onPromptMentionSelect,
  onPromptPaste,
  onPromptPasteRemove,
  onSelectLocalResource,
  onSubmit,
  projectPath,
  projects,
  prompt,
  promptResourceReferences,
  pendingPromptPastes,
  textareaRef,
}: PromptComposerTextareaProps) {
  const editorViewRef = useRef<EditorView | null>(null);
  const rootRef = useRef<HTMLDivElement>(null);
  const publishedPrompt = useRef(prompt);
  const knownReferences = useRef(new Map<string, PromptMentionReference>());
  for (const reference of promptResourceReferences)
    knownReferences.current.set(referenceKey(reference), reference);
  const [query, setQuery] =
    useState<ReturnType<typeof editorMentionQuery>>(null);
  const [activeIndex, setActiveIndex] = useState(0);
  const [focused, setFocused] = useState(false);
  const [dismissed, setDismissed] = useState(false);
  const [selectionToolbar, setSelectionToolbar] = useState(false);
  const [editingLink, setEditingLink] = useState(false);
  const [linkUrl, setLinkUrl] = useState("");
  const listId = useId();
  const mentionOpen = focused && !dismissed && Boolean(query);
  const handlers = useRef({
    onPromptChange,
    onPromptMentionRemove,
    onPromptMentionSelect,
    handleKeyDown,
    handlePaste,
  });
  handlers.current = {
    onPromptChange,
    onPromptMentionRemove,
    onPromptMentionSelect,
    handleKeyDown,
    handlePaste,
  };

  function updateSelection(view: EditorView) {
    setQuery(editorMentionQuery(view.state));
    setSelectionToolbar(
      !view.state.selection.empty &&
        !(view.state.selection instanceof NodeSelection),
    );
  }

  useLayoutEffect(() => {
    const mount = textareaRef.current;
    if (!mount) return;
    const view = new EditorView(mount, {
      state: createPromptEditorState(
        parsePromptDocument(prompt, promptResourceReferences),
      ),
      attributes: {
        role: "textbox",
        "aria-label": "任务描述",
        "aria-multiline": "true",
        "data-placeholder": "随心输入",
        "data-empty": String(!prompt),
        autocomplete: "off",
        autocorrect: "on",
        autocapitalize: "sentences",
        spellcheck: "true",
        translate: "no",
      },
      dispatchTransaction(transaction) {
        const previousReferences = promptDocumentReferences(view.state.doc);
        const state = view.state.apply(transaction);
        view.updateState(state);
        view.dom.dataset.empty = String(
          state.doc.childCount === 1 &&
            state.doc.firstChild?.content.size === 0,
        );
        if (transaction.docChanged) {
          const references = promptDocumentReferences(state.doc);
          for (const reference of references) {
            knownReferences.current.set(referenceKey(reference), reference);
            if (
              !previousReferences.some(
                (item) => referenceKey(item) === referenceKey(reference),
              )
            ) {
              handlers.current.onPromptMentionSelect(reference);
            }
          }
          for (const reference of previousReferences) {
            if (
              !references.some(
                (item) => referenceKey(item) === referenceKey(reference),
              )
            ) {
              handlers.current.onPromptMentionRemove(reference);
            }
          }
          publishedPrompt.current = serializePromptDocument(state.doc);
          handlers.current.onPromptChange(publishedPrompt.current);
        }
        if (transaction.docChanged || transaction.selectionSet) {
          updateSelection(view);
          setActiveIndex(0);
          setDismissed(false);
        }
      },
      handleKeyDown: (view, event) =>
        handlers.current.handleKeyDown(view, event),
      handleDOMEvents: {
        paste: (_view, event) => handlers.current.handlePaste(event),
      },
      clipboardTextParser: (text) =>
        promptPasteSlice(text, [...knownReferences.current.values()]),
      clipboardTextSerializer: (slice) =>
        serializePromptDocument(
          promptEditorSchema.node(
            "doc",
            null,
            slice.content.firstChild?.isBlock
              ? slice.content
              : promptEditorSchema.node("paragraph", null, slice.content),
          ),
        ),
      transformPasted: (slice) => {
        const doc = promptEditorSchema.node("doc", null, slice.content);
        return new Slice(
          resolveMentionNodes(doc, [...knownReferences.current.values()])
            .content,
          slice.openStart,
          slice.openEnd,
        );
      },
      nodeViews: {
        mention: (node) => ({
          dom: createMentionElement(node.attrs.reference),
        }),
      },
    });
    editorViewRef.current = view;
    return () => {
      editorViewRef.current = null;
      view.destroy();
    };
    // React owns only the mount. The same EditorView survives edits and blur.
  }, [textareaRef]);

  useLayoutEffect(() => {
    const view = editorViewRef.current;
    if (!view || prompt === publishedPrompt.current) return;
    publishedPrompt.current = prompt;
    const doc = parsePromptDocument(prompt, [
      ...knownReferences.current.values(),
    ]);
    // External replacement (send, draft switch, pasted-text expansion) is the
    // only reset path. Echoes of editor transactions never rebuild its state.
    view.updateState(createPromptEditorState(doc));
    view.dom.dataset.empty = String(
      doc.childCount === 1 && doc.firstChild?.content.size === 0,
    );
    updateSelection(view);
  }, [prompt]);

  useLayoutEffect(() => {
    const view = editorViewRef.current;
    if (!view) return;
    view.dom.setAttribute("aria-expanded", String(mentionOpen));
    if (mentionOpen) {
      view.dom.setAttribute("aria-controls", listId);
      const option = rootRef.current?.querySelectorAll(
        '[role="option"]:not([disabled])',
      )[activeIndex];
      if (option?.id) view.dom.setAttribute("aria-activedescendant", option.id);
    } else {
      view.dom.removeAttribute("aria-controls");
      view.dom.removeAttribute("aria-activedescendant");
    }
  }, [activeIndex, listId, mentionOpen, query]);

  function selectMention(reference: PromptMentionReference) {
    const view = editorViewRef.current;
    const range = view && editorMentionQuery(view.state);
    if (!view || !range) return;
    view.dispatch(
      insertPromptMention(view.state, range.from, range.to, reference),
    );
    view.dispatch(closeHistory(view.state.tr));
    view.focus();
  }

  function selectMentionAction(item: PromptMentionAction) {
    const view = editorViewRef.current;
    const range = view && editorMentionQuery(view.state);
    if (!view || !range || item.disabled) return;
    view.dispatch(view.state.tr.delete(range.from, range.to));
    setDismissed(true);
    if (item.action === "select-local")
      void onSelectLocalResource().finally(() => view.focus());
  }

  function handleKeyDown(view: EditorView, event: KeyboardEvent) {
    if (view.composing || event.isComposing || event.keyCode === 229)
      return false;
    if (mentionOpen) {
      if (event.key === "Escape") {
        setDismissed(true);
        return true;
      }
      const options = rootRef.current?.querySelectorAll<HTMLButtonElement>(
        '[role="option"]:not([disabled])',
      );
      if (event.key === "ArrowDown" || event.key === "ArrowUp") {
        const count = options?.length ?? 0;
        setActiveIndex((index) =>
          count
            ? (index + (event.key === "ArrowDown" ? 1 : count - 1)) % count
            : 0,
        );
        return true;
      }
      if ((event.key === "Enter" && !event.shiftKey) || event.key === "Tab") {
        options?.[Math.min(activeIndex, options.length - 1)]?.click();
        return true;
      }
    }
    if (event.key === "Enter" && !event.shiftKey) {
      onSubmit(serializePromptDocument(view.state.doc));
      return true;
    }
    return false;
  }

  function handlePaste(event: ClipboardEvent) {
    const clipboard = event.clipboardData;
    if (!clipboard) return false;
    const files = Array.from(clipboard.files);
    const images = files.filter((file) => file.type.startsWith("image/"));
    if (images.length) onAddAttachments(images);
    const paths = files
      .filter((file) => !images.includes(file))
      .map((file) => window.aivoDesktop?.file?.pathForFile?.(file) ?? "")
      .filter(Boolean);
    if (paths.length)
      paths.forEach((path, index) =>
        onPromptPaste(index ? "" : clipboard.getData("text/plain"), path),
      );
    const handled =
      images.length ||
      paths.length ||
      onPromptPaste(clipboard.getData("text/plain"));
    if (handled || files.length) {
      event.preventDefault();
      if (!handled && files.length) onAddAttachments(files);
      return true;
    }
    return false;
  }

  function runCommand(command: Command) {
    const view = editorViewRef.current;
    if (!view) return;
    command(view.state, view.dispatch, view);
    view.focus();
  }

  function showPromptPaste(paste: PendingPromptPaste) {
    const view = editorViewRef.current;
    if (!view) return;
    const tr = view.state.tr.replaceSelection(
      promptPasteSlice(paste.text, [...knownReferences.current.values()]),
    );
    view.dispatch(tr.scrollIntoView());
    onPromptPasteRemove(paste.id);
    view.focus();
  }

  function openPromptPaste(paste: PendingPromptPaste) {
    if (!window.aivoDesktop?.file) {
      toast.error("当前环境不支持系统默认打开");
      return;
    }
    const target = paste.target ?? promptPasteTarget(paste.text);
    const open = !target
      ? window.aivoDesktop.file.openText(
          paste.name ?? inferPromptPasteName(paste.text) ?? "粘贴内容.txt",
          paste.text,
        )
      : isExternalPromptPasteTarget(target)
        ? window.aivoDesktop.file.openExternal(target)
        : window.aivoDesktop.file.openPath(target);
    void open.catch((error: unknown) =>
      toast.error("无法用系统默认方式打开", {
        description: error instanceof Error ? error.message : String(error),
      }),
    );
  }

  return (
    <div
      className="relative min-w-0"
      ref={rootRef}
      onFocusCapture={() => setFocused(true)}
      onBlurCapture={(event) => {
        if (!event.currentTarget.contains(event.relatedTarget as Node | null)) {
          setFocused(false);
          setEditingLink(false);
        }
      }}
    >
      {mentionOpen && query ? (
        <PromptMentionPicker
          id={listId}
          activeIndex={activeIndex}
          onActiveIndexChange={setActiveIndex}
          onSelect={(item) => selectMention(item.reference)}
          onSelectAction={selectMentionAction}
          projectPath={projectPath}
          projects={projects}
          query={query.query}
        />
      ) : null}
      {((focused && selectionToolbar) || editingLink) && !mentionOpen ? (
        <div
          role="toolbar"
          aria-label="文本格式"
          className="absolute bottom-full left-0 z-40 mb-2 flex items-center gap-1 rounded-xl border bg-popover p-1 shadow-md"
          onMouseDown={(event) => {
            if (
              !(event.target instanceof HTMLInputElement) &&
              !(event.target instanceof HTMLSelectElement)
            )
              event.preventDefault();
          }}
        >
          <button
            type="button"
            aria-label="链接"
            className="rounded px-2 py-1 hover:bg-muted"
            onClick={() => setEditingLink((value) => !value)}
          >
            链接
          </button>
          <button
            type="button"
            aria-label="加粗"
            className="rounded px-2 py-1 font-bold hover:bg-muted"
            onClick={() =>
              runCommand(toggleMark(promptEditorSchema.marks.strong))
            }
          >
            B
          </button>
          <button
            type="button"
            aria-label="斜体"
            className="rounded px-2 py-1 italic hover:bg-muted"
            onClick={() => runCommand(toggleMark(promptEditorSchema.marks.em))}
          >
            I
          </button>
          <select
            aria-label="文本样式"
            className="bg-popover p-1 text-sm"
            defaultValue=""
            onChange={(event) => {
              const value = event.target.value;
              runCommand(
                value === "quote"
                  ? wrapIn(promptEditorSchema.nodes.blockquote)
                  : value === "list"
                    ? wrapInList(promptEditorSchema.nodes.bullet_list)
                    : value === "code"
                      ? setBlockType(promptEditorSchema.nodes.code_block)
                      : value === "paragraph"
                        ? setBlockType(promptEditorSchema.nodes.paragraph)
                        : setBlockType(promptEditorSchema.nodes.heading, {
                            level: Number(value),
                          }),
              );
              event.target.value = "";
            }}
          >
            <option value="" disabled>
              文本
            </option>
            <option value="paragraph">正文</option>
            <option value="1">标题 1</option>
            <option value="2">标题 2</option>
            <option value="3">标题 3</option>
            <option value="list">列表</option>
            <option value="quote">引用</option>
            <option value="code">代码</option>
          </select>
          {editingLink ? (
            <form
              className="flex gap-1"
              onSubmit={(event) => {
                event.preventDefault();
                if (/^https?:\/\//i.test(linkUrl))
                  runCommand(
                    toggleMark(promptEditorSchema.marks.link, {
                      href: linkUrl,
                    }),
                  );
                setEditingLink(false);
                setLinkUrl("");
              }}
            >
              <input
                autoFocus
                aria-label="链接地址"
                className="w-48 border px-2 text-sm"
                placeholder="https://"
                value={linkUrl}
                onChange={(event) => setLinkUrl(event.target.value)}
              />
              <button type="submit">确定</button>
            </form>
          ) : null}
        </div>
      ) : null}
      {pendingPromptPastes.length ? (
        <AttachmentGroup className="gap-2">
          {pendingPromptPastes.map((paste) => (
            <PromptPasteCard
              key={paste.id}
              paste={paste}
              onOpen={() => openPromptPaste(paste)}
              onRemove={() => onPromptPasteRemove(paste.id)}
              onShow={() => showPromptPaste(paste)}
            />
          ))}
        </AttachmentGroup>
      ) : null}
      <div
        className="aivo-prompt-editor min-h-8 max-h-[300px] w-full overflow-y-auto text-[14px] leading-[1.625]"
        ref={textareaRef}
      />
    </div>
  );
}

function referenceKey(reference: PromptMentionReference) {
  return reference.kind + ":" + reference.id;
}

function createMentionElement(reference: PromptMentionReference) {
  const mention = document.createElement("span");
  mention.className = "aivo-prompt-mention";
  mention.contentEditable = "false";
  mention.dataset.promptMention = "true";
  mention.dataset.mentionId = reference.id;
  mention.dataset.mentionKind = reference.kind;
  mention.setAttribute("aria-label", reference.token);
  const label = document.createElement("span");
  label.className = "aivo-prompt-mention-label";
  label.textContent = reference.token;
  mention.append(createMentionIcon(reference.kind), label);
  return mention;
}

function PromptPasteCard({
  onOpen,
  onRemove,
  onShow,
  paste,
}: {
  onOpen: () => void;
  onRemove: () => void;
  onShow: () => void;
  paste: PendingPromptPaste;
}) {
  const title = promptPasteTitle(paste);
  const openableTarget = paste.target ?? promptPasteTarget(paste.text);

  return (
    <Attachment
      className={cn(
        "h-10 w-40 flex-nowrap gap-2 rounded-lg bg-background p-1.5",
        openableTarget && "cursor-pointer",
      )}
      onClick={onOpen}
    >
      <AttachmentMedia
        className="size-7 rounded-lg bg-muted/60 text-foreground"
        variant="icon"
      >
        <HugeiconsIcon className="size-4" icon={File02Icon} strokeWidth={2} />
      </AttachmentMedia>
      <AttachmentContent className="min-w-0 overflow-hidden py-0.5 pr-4">
        <AttachmentTitle
          className="w-full text-sm leading-5"
          title={openableTarget}
        >
          {title}
        </AttachmentTitle>
        <button
          className="mt-px flex min-w-0 items-center gap-1 text-[10px] font-medium text-muted-foreground underline underline-offset-2 transition-colors hover:text-foreground"
          onClick={(event) => {
            event.stopPropagation();
            onShow();
          }}
          type="button"
        >
          <span className="truncate">在文本框中显示</span>
          <HugeiconsIcon
            className="size-3 shrink-0"
            icon={ArrowRight01Icon}
            strokeWidth={2}
          />
        </button>
      </AttachmentContent>
      <AttachmentActions className="relative -mr-0.5 -mt-0.5 self-start">
        <AttachmentAction
          aria-label={`移除${title}`}
          className="size-5 rounded-full bg-background/95 p-0 shadow-sm ring-1 ring-border/60 hover:bg-muted"
          onClick={(event) => {
            event.stopPropagation();
            onRemove();
          }}
          type="button"
        >
          <HugeiconsIcon
            className="size-3"
            icon={Cancel01Icon}
            strokeWidth={2}
          />
        </AttachmentAction>
      </AttachmentActions>
    </Attachment>
  );
}

function createMentionIcon(kind: PromptMentionReference["kind"]) {
  const svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
  svg.setAttribute("aria-hidden", "true");
  svg.setAttribute(
    "class",
    "mr-1 inline-block size-4 shrink-0 align-[-0.16em]",
  );
  svg.setAttribute("fill", "none");
  svg.setAttribute("stroke", "currentColor");
  svg.setAttribute("stroke-linecap", "round");
  svg.setAttribute("stroke-linejoin", "round");
  svg.setAttribute("stroke-width", "1.8");
  svg.setAttribute("viewBox", "0 0 24 24");

  const path = document.createElementNS("http://www.w3.org/2000/svg", "path");
  path.setAttribute(
    "d",
    kind === "project"
      ? "M3 7.5A2.5 2.5 0 0 1 5.5 5H10l2 2h6.5A2.5 2.5 0 0 1 21 9.5v8A2.5 2.5 0 0 1 18.5 20h-13A2.5 2.5 0 0 1 3 17.5z"
      : kind === "conversation"
        ? "M20 11.5a7.5 7.5 0 0 1-9.8 7.1L4 20l1.4-4.1A7.5 7.5 0 1 1 20 11.5z"
        : kind === "skill"
          ? "M8 3a2 2 0 1 1 4 0v3h3a2 2 0 1 1 0 4h-3v3a2 2 0 1 1-4 0v-3H5a2 2 0 1 1 0-4h3z"
          : kind === "mcp"
            ? "M5 5h14v5H5zM5 14h14v5H5zM8 7.5h.01M8 16.5h.01"
            : kind === "extension"
              ? "m12 3 7 4v10l-7 4-7-4V7zM5 7l7 4 7-4M12 11v10"
              : "M14.7 6.3a4 4 0 0 0-5 5L4 17l3 3 5.7-5.7a4 4 0 0 0 5-5L14 13l-3-3z",
  );
  svg.append(path);
  return svg;
}
