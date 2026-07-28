import assert from "node:assert/strict";
import { chmod, mkdtemp, readFile, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import test from "node:test";

import tmuxIntrayExtension from "./index.ts";

type ToolResultEvent = { isError: boolean; toolName: string };
type Handler = (event: ToolResultEvent) => Promise<void>;

test("notifies for tool errors except bash errors", async (t) => {
  const handlers = new Map<string, Handler>();
  const pi = {
    on(eventName: string, handler: Handler) {
      handlers.set(eventName, handler);
    },
  };

  tmuxIntrayExtension(pi as never);

  assert.deepEqual([...handlers.keys()], ["agent_end", "tool_result", "session_shutdown"]);

  const tempDirectory = await mkdtemp(join(tmpdir(), "pi-tmux-intray-"));
  const callsFile = join(tempDirectory, "calls");
  const fakeIntray = join(tempDirectory, "tmux-intray");
  await writeFile(fakeIntray, '#!/bin/sh\nprintf "%s\\n" "$*" >> "$TMUX_INTRAY_TEST_CALLS"\n');
  await chmod(fakeIntray, 0o755);

  const originalIntrayPath = process.env.TMUX_INTRAY_PATH;
  process.env.TMUX_INTRAY_PATH = fakeIntray;
  process.env.TMUX_INTRAY_TEST_CALLS = callsFile;
  t.after(async () => {
    if (originalIntrayPath === undefined) delete process.env.TMUX_INTRAY_PATH;
    else process.env.TMUX_INTRAY_PATH = originalIntrayPath;
    delete process.env.TMUX_INTRAY_TEST_CALLS;
    await rm(tempDirectory, { recursive: true, force: true });
  });

  const toolResult = handlers.get("tool_result");
  assert.ok(toolResult);

  await toolResult({ isError: true, toolName: "bash" });
  await toolResult({ isError: false, toolName: "read" });
  await assert.rejects(readFile(callsFile, "utf8"), { code: "ENOENT" });

  await toolResult({ isError: true, toolName: "read" });
  const calls = await readFile(callsFile, "utf8");
  assert.match(calls, /add --level=error .*-- Tool error: read\n$/);
});
