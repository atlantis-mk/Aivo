import assert from "node:assert/strict";
import { describe, it } from "node:test";

import {
  attachmentFileTypeLabel,
  formatAttachmentOnlyPrompt,
  promptWithTextAttachments,
} from "./project-composer-attachments.ts";

describe("directory attachments", () => {
  it("adds folder paths to the transport prompt without changing the workspace", () => {
    const prompt = promptWithTextAttachments("检查这个目录", [
      {
        id: "directory-1",
        name: "backend",
        mimeType: "inode/directory",
        size: 0,
        kind: "directory" as const,
        data: "",
        path: "/tmp/backend",
      },
    ]);

    assert.match(prompt, /检查这个目录/);
    assert.match(prompt, /\/tmp\/backend/);
  });

  it("renders directory attachments distinctly", () => {
    const attachment = {
      id: "directory-1",
      name: "backend",
      mimeType: "inode/directory",
      size: 0,
      kind: "directory" as const,
      data: "",
      path: "/tmp/backend",
    };

    assert.equal(attachmentFileTypeLabel(attachment), "FOLDER");
    assert.equal(formatAttachmentOnlyPrompt([attachment]), "目录：backend");
  });
});
