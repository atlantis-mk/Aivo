import type { ConversationUserAttachment } from "@/features/projects/conversation-timeline-model";
import {
  numberFromUnknown,
  recordFromUnknown,
  stringFromUnknown,
} from "@/features/projects/project-conversation-payload";
import type { domain } from "../../../bridge/go/models";

export function conversationAttachmentsFromEvent(
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
      kind,
      mimeType,
      name,
      previewUrl:
        kind === "image" && data ? `data:${mimeType};base64,${data}` : undefined,
      size: numberFromUnknown(attachment?.size),
    });
  }
  return attachments;
}
