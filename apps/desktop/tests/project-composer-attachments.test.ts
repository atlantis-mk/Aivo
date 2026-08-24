import assert from "node:assert/strict";
import test from "node:test";

import {
  attachmentFileTypeLabel,
  isSupportedBinaryComposerAttachment,
  isTextComposerAttachment,
  modelSupportsAttachment,
  readComposerAttachmentFiles,
  readNativeComposerAttachment,
  routeComposerLocalSelections,
} from "../src/features/projects/project-composer-attachments.ts";

test("markdown attachments are sent as text without file-capable model metadata", async () => {
  const file = new File(["# Deployment\n\nRun the installer."], "deployment.md", {
    type: "text/markdown",
  });

  const result = await readComposerAttachmentFiles([file], undefined, undefined);

  assert.deepEqual(result.rejections, []);
  assert.equal(result.attachments.length, 1);
  assert.equal(result.attachments[0]?.data, "");
  assert.equal(result.attachments[0]?.text, "# Deployment\n\nRun the installer.");
  assert.equal(result.attachments[0]?.mimeType, "text/markdown");
});

test("native markdown selections decode base64 content as UTF-8 text", () => {
  const text = "# 部署说明\n\n执行安装。";
  const result = readNativeComposerAttachment({
    kind: "file",
    name: "deployment.md",
    mimeType: "text/plain",
    size: new TextEncoder().encode(text).byteLength,
    data: Buffer.from(text).toString("base64"),
  }, undefined, undefined);

  assert.deepEqual(result.rejections, []);
  assert.equal(result.attachments[0]?.data, "");
  assert.equal(result.attachments[0]?.text, text);
});

test("native text selections reject invalid UTF-8 instead of sending replacement characters", () => {
  const result = readNativeComposerAttachment({
    kind: "file",
    name: "broken.md",
    mimeType: "text/plain",
    size: 1,
    data: Buffer.from([0xff]).toString("base64"),
  }, undefined, undefined);

  assert.equal(result.attachments.length, 0);
  assert.match(result.rejections[0] ?? "", /UTF-8/);
});

test("generic native MIME is corrected from a supported filename", () => {
  const result = readNativeComposerAttachment({
    kind: "file",
    name: "brief.pdf",
    mimeType: "application/octet-stream",
    size: 3,
    data: "cGRm",
  }, undefined, {
    id: "document-model",
    providerId: "test-provider",
    name: "Document model",
    capabilities: ["file"],
  });

  assert.deepEqual(result.rejections, []);
  assert.equal(result.attachments[0]?.mimeType, "application/pdf");
  assert.equal(result.attachments[0]?.data, "cGRm");
});

test("unknown UTF-8 files are classified as text instead of octet-stream", () => {
  const text = "plain extensionless notes\n";
  const result = readNativeComposerAttachment({
    kind: "file",
    name: "NOTES",
    mimeType: "application/octet-stream",
    size: text.length,
    data: Buffer.from(text).toString("base64"),
  }, undefined, undefined);

  assert.deepEqual(result.rejections, []);
  assert.equal(result.attachments[0]?.mimeType, "text/plain");
  assert.equal(result.attachments[0]?.data, "");
  assert.equal(result.attachments[0]?.text, text);
});

test("unknown binary files are rejected before model capability routing", () => {
  const result = readNativeComposerAttachment({
    kind: "file",
    name: "archive.zip",
    mimeType: "application/octet-stream",
    size: 4,
    data: Buffer.from([0x50, 0x4b, 0x03, 0x04]).toString("base64"),
  }, undefined, {
    id: "document-model",
    providerId: "test-provider",
    name: "Document model",
    capabilities: ["file"],
  });

  assert.equal(result.attachments.length, 0);
  assert.match(result.rejections[0] ?? "", /文件类型不受支持/);
});

test("binary attachments still require model file capability", () => {
  assert.equal(
    modelSupportsAttachment(
      undefined,
      undefined,
      "file",
      "application/pdf",
      "brief.pdf",
    ),
    false,
  );
  assert.equal(
    modelSupportsAttachment(
      undefined,
      {
        id: "document-model",
        providerId: "test-provider",
        name: "Document model",
        capabilities: ["file"],
      },
      "file",
      "application/pdf",
      "brief.pdf",
    ),
    true,
  );
  assert.equal(
    modelSupportsAttachment(
      undefined,
      {
        id: "catalog-attachment-model",
        providerId: "test-provider",
        name: "Catalog attachment model",
        capabilities: ["attachments"],
      },
      "file",
      "application/pdf",
      "brief.pdf",
    ),
    true,
  );
});

test("source and structured text types are recognized without relying only on MIME", () => {
  assert.equal(isTextComposerAttachment("application/octet-stream", "main.go"), true);
  assert.equal(isTextComposerAttachment("application/yaml", "config.yaml"), true);
  assert.equal(isTextComposerAttachment("application/octet-stream", "archive.zip"), false);
});

test("binary attachment MIME support is an explicit allowlist", () => {
  assert.equal(isSupportedBinaryComposerAttachment("application/pdf"), true);
  assert.equal(isSupportedBinaryComposerAttachment("image/png"), true);
  assert.equal(isSupportedBinaryComposerAttachment("application/octet-stream"), false);
  assert.equal(isSupportedBinaryComposerAttachment("application/zip"), false);
});

test("file cards prefer a compact uppercase extension label", () => {
  assert.equal(
    attachmentFileTypeLabel({ name: "rust-toolchain.toml", mimeType: "text/plain" }),
    "TOML",
  );
  assert.equal(
    attachmentFileTypeLabel({ name: "CONTRIBUTING.md", mimeType: "text/plain" }),
    "MD",
  );
  assert.equal(
    attachmentFileTypeLabel({ name: "LICENSE", mimeType: "text/plain" }),
    "TXT",
  );
});

test("local selections route files and only one folder through the shared flow", () => {
  const files: string[] = [];
  const directories: string[] = [];
  const result = routeComposerLocalSelections([
    {
      kind: "file",
      name: "README.md",
      mimeType: "text/plain",
      size: 4,
      data: "dGVzdA==",
    },
    { kind: "directory", path: "/workspace/first" },
    {
      kind: "file",
      name: "brief.pdf",
      mimeType: "application/pdf",
      size: 3,
      data: "cGRm",
    },
    { kind: "directory", path: "/workspace/second" },
  ], {
    onDirectory: (path) => directories.push(path),
    onFile: (file) => files.push(file.name),
  });

  assert.deepEqual(files, ["README.md", "brief.pdf"]);
  assert.deepEqual(directories, ["/workspace/first"]);
  assert.equal(result.ignoredDirectoryCount, 1);
});
