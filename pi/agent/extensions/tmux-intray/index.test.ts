import assert from "node:assert/strict";
import test from "node:test";

import tmuxIntrayExtension from "./index.ts";

test("registers lifecycle notifications without tool error messages", async () => {
  const handlers = new Map<string, () => Promise<void>>();
  const pi = {
    on(eventName: string, handler: () => Promise<void>) {
      handlers.set(eventName, handler);
    },
  };

  tmuxIntrayExtension(pi as never);

  assert.deepEqual([...handlers.keys()], ["agent_end", "session_shutdown"]);

  process.env.TMUX_INTRAY_PATH = "/usr/bin/true";
  await handlers.get("agent_end")?.();
  await handlers.get("session_shutdown")?.();
});
