import {
  Fragment,
  Schema,
  Slice,
  type Node as PMNode,
} from "prosemirror-model";
import { EditorState, TextSelection, type Command } from "prosemirror-state";
import {
  baseKeymap,
  chainCommands,
  createParagraphNear,
  liftEmptyBlock,
  newlineInCode,
  splitBlock,
  toggleMark,
} from "prosemirror-commands";
import { closeHistory, history, redo, undo } from "prosemirror-history";
import { keymap } from "prosemirror-keymap";
import {
  InputRule,
  inputRules,
  textblockTypeInputRule,
  undoInputRule,
  wrappingInputRule,
} from "prosemirror-inputrules";
import {
  defaultMarkdownParser,
  defaultMarkdownSerializer,
  MarkdownParser,
  MarkdownSerializer,
  schema as markdownSchema,
} from "prosemirror-markdown";
import {
  liftListItem,
  sinkListItem,
  splitListItem,
} from "prosemirror-schema-list";
import {
  activePromptMentionReferences,
  type PromptMentionReference,
} from "./project-prompt-mention-model.ts";

export function mentionHref(reference: PromptMentionReference) {
  return `aivo-mention://${reference.kind}/${encodeURIComponent(reference.id)}`;
}

export const promptEditorSchema = new Schema({
  nodes: markdownSchema.spec.nodes
    .update("heading", {
      ...markdownSchema.spec.nodes.get("heading"),
      content: "inline*",
    })
    .append({
      mention: {
        inline: true,
        group: "inline",
        atom: true,
        selectable: true,
        attrs: { reference: {} },
        // HTML round trips through an ordinary link and is resolved against the
        // known resource catalog on paste. Clipboard HTML cannot invent metadata.
        toDOM: (node) => [
          "a",
          { href: mentionHref(node.attrs.reference) },
          node.attrs.reference.token,
        ],
        leafText: (node) => node.attrs.reference.token,
      },
    }),
  marks: markdownSchema.spec.marks,
});

const parser = new MarkdownParser(
  promptEditorSchema,
  defaultMarkdownParser.tokenizer,
  {
    ...defaultMarkdownParser.tokens,
    softbreak: { node: "hard_break" },
  },
);
export const promptMarkdownSerializer = new MarkdownSerializer(
  {
    ...defaultMarkdownSerializer.nodes,
    mention(state, node) {
      const reference = node.attrs.reference as PromptMentionReference;
      state.write(`[${state.esc(reference.token)}](${mentionHref(reference)})`);
    },
  },
  defaultMarkdownSerializer.marks,
);

export function resolveMentionNodes(
  node: PMNode,
  references: PromptMentionReference[],
): PMNode {
  if (node.isText) {
    const link = node.marks.find((mark) => mark.type.name === "link");
    const reference =
      link && references.find((item) => mentionHref(item) === link.attrs.href);
    if (reference)
      return promptEditorSchema.nodes.mention.create({ reference });
    return node;
  }
  return node.copy(
    Fragment.fromArray(
      Array.from({ length: node.childCount }, (_, index) =>
        resolveMentionNodes(node.child(index), references),
      ),
    ),
  );
}

export function parsePromptDocument(
  markdown: string,
  references: PromptMentionReference[] = [],
) {
  return resolveMentionNodes(parser.parse(markdown), references);
}

export function serializePromptDocument(doc: PMNode) {
  return promptMarkdownSerializer.serialize(doc);
}

export function promptDocumentReferences(doc: PMNode) {
  const references: PromptMentionReference[] = [];
  doc.descendants((node) => {
    if (node.type.name === "mention") references.push(node.attrs.reference);
  });
  return activePromptMentionReferences(references);
}

/** Resolve only references actually present in the submitted document. */
export function parsePromptSubmission(
  markdown: string,
  references: PromptMentionReference[],
) {
  const doc = parsePromptDocument(markdown, references);
  const serializer = new MarkdownSerializer(
    {
      ...promptMarkdownSerializer.nodes,
      mention(state, node) {
        state.text(node.attrs.reference.token);
      },
    },
    defaultMarkdownSerializer.marks,
  );
  // Keep ordinary Markdown byte-for-byte intact when there are no mentions.
  const active = promptDocumentReferences(doc);
  return {
    text: active.length ? serializer.serialize(doc) : markdown,
    references: active,
  };
}

export function editorMentionQuery(state: EditorState) {
  const { $from, empty } = state.selection;
  if (
    !empty ||
    !$from.parent.isTextblock ||
    $from.parent.type.spec.code ||
    $from
      .marks()
      .some((mark) => mark.type.name === "code" || mark.type.name === "link")
  )
    return null;
  const before = $from.parent.textBetween(0, $from.parentOffset, "", "\ufffc");
  const match = /(?:^|[\s\ufffc])@([^\s@]*)$/.exec(before);
  if (!match) return null;
  return {
    query: match[1],
    from: $from.pos - match[1].length - 1,
    to: $from.pos,
  };
}

export function insertPromptMention(
  state: EditorState,
  from: number,
  to: number,
  reference: PromptMentionReference,
) {
  const tr = closeHistory(state.tr).replaceWith(from, to, [
    promptEditorSchema.nodes.mention.create({ reference }),
    promptEditorSchema.text(" "),
  ]);
  tr.setSelection(TextSelection.create(tr.doc, from + 2));
  // The workspace has one destination project. Switching it also removes its
  // previous inline token, while preserving all other text and references.
  if (reference.kind === "project") {
    const obsolete: number[] = [];
    tr.doc.descendants((node, pos) => {
      if (
        node.type.name === "mention" &&
        node.attrs.reference.kind === "project" &&
        node.attrs.reference.id !== reference.id
      )
        obsolete.push(pos);
    });
    for (const pos of obsolete.reverse()) tr.delete(pos, pos + 1);
  }
  return tr.scrollIntoView();
}

function markRule(pattern: RegExp, name: string) {
  return new InputRule(pattern, (state, match, start, end) => {
    const text = match[1];
    if (
      !text ||
      /\\$/.test(state.doc.textBetween(Math.max(0, start - 1), start))
    )
      return null;
    return state.tr
      .replaceWith(
        start,
        end,
        promptEditorSchema.text(text, [
          promptEditorSchema.marks[name].create(),
        ]),
      )
      .removeStoredMark(promptEditorSchema.marks[name]);
  });
}

export const insertPromptParagraph = chainCommands(
  newlineInCode,
  splitListItem(promptEditorSchema.nodes.list_item),
  createParagraphNear,
  liftEmptyBlock,
  splitBlock,
);

const deleteMention =
  (backwards: boolean): Command =>
  (state, dispatch) => {
    if (!state.selection.empty) return false;
    const { $from } = state.selection;
    const node = backwards ? $from.nodeBefore : $from.nodeAfter;
    if (node?.type.name !== "mention") return false;
    dispatch?.(
      closeHistory(state.tr).delete(
        backwards ? $from.pos - node.nodeSize : $from.pos,
        backwards ? $from.pos : $from.pos + node.nodeSize,
      ),
    );
    return true;
  };

export function createPromptEditorState(doc: PMNode) {
  const { nodes, marks } = promptEditorSchema;
  return EditorState.create({
    doc,
    plugins: [
      inputRules({
        rules: [
          markRule(/\*\*([^*\n]+)\*\*$/, "strong"),
          markRule(/__([^_\n]+)__$/, "strong"),
          markRule(/(?<!\*)\*([^*\n]+)\*$/, "em"),
          markRule(/(?<!_)_([^_\n]+)_$/, "em"),
          markRule(/`([^`\n]+)`$/, "code"),
          new InputRule(
            /\[([^\]\n]+)\]\((https?:\/\/[^\s)]+)\)$/,
            (state, match, start, end) =>
              state.tr.replaceWith(
                start,
                end,
                promptEditorSchema.text(match[1], [
                  marks.link.create({ href: match[2] }),
                ]),
              ),
          ),
          textblockTypeInputRule(/^(#{1,6})\s$/, nodes.heading, (match) => ({
            level: match[1].length,
          })),
          textblockTypeInputRule(/^```$/, nodes.code_block),
          new InputRule(
            /(?:^|\n)```$/,
            (state, _match, start) => {
              const { $from } = state.selection;
              if ($from.parent.type !== nodes.code_block) return null;
              const text = state.doc.textBetween($from.start(), start, "\n");
              const code = nodes.code_block.create(
                $from.parent.attrs,
                text ? promptEditorSchema.text(text) : null,
              );
              const before = $from.before();
              const tr = state.tr.replaceWith(before, $from.after(), [
                code,
                nodes.paragraph.create(),
              ]);
              return tr.setSelection(
                TextSelection.create(tr.doc, before + code.nodeSize + 1),
              );
            },
            { inCode: true },
          ),
          wrappingInputRule(/^\s*>\s$/, nodes.blockquote),
          wrappingInputRule(/^\s*([-+*])\s$/, nodes.bullet_list),
          wrappingInputRule(/^(\d+)\.\s$/, nodes.ordered_list, (match) => ({
            order: +match[1],
          })),
        ],
      }),
      history(),
      keymap({
        "Mod-z": undo,
        "Shift-Mod-z": redo,
        "Mod-y": redo,
        "Mod-b": toggleMark(marks.strong),
        "Mod-i": toggleMark(marks.em),
        "Mod-`": toggleMark(marks.code),
        Backspace: chainCommands(undoInputRule, deleteMention(true)),
        Delete: deleteMention(false),
        "Shift-Enter": insertPromptParagraph,
        Tab: sinkListItem(nodes.list_item),
        "Shift-Tab": liftListItem(nodes.list_item),
      }),
      keymap(baseKeymap),
    ],
  });
}

export function promptPasteSlice(
  text: string,
  references: PromptMentionReference[],
) {
  return Slice.maxOpen(parsePromptDocument(text, references).content);
}
