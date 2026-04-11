import type { ExtensionAPI } from "@mariozechner/pi-coding-agent";
import { execFile } from "node:child_process";
import { promisify } from "node:util";
import { promises as fs } from "node:fs";

const execFileAsync = promisify(execFile);
const LOG_FILE = "/tmp/pi-tmux-intray.log";

async function log(message: string): Promise<void> {
  try {
    const timestamp = new Date().toISOString();
    await fs.appendFile(LOG_FILE, `[${timestamp}] [pi-tmux-intray] ${message}\n`);
  } catch {
    // Never crash extension because of logging failures
  }
}

async function logError(where: string, error: unknown): Promise<void> {
  const msg = error instanceof Error ? (error.stack ?? error.message) : String(error);
  await log(`${where}: ERROR - ${msg}`);
}

function getTmuxIntrayCommand(): string {
  if (process.env.TMUX_INTRAY_PATH) return process.env.TMUX_INTRAY_PATH;
  if (process.env.TMUX_INTRAY_BIN) return process.env.TMUX_INTRAY_BIN;
  return "tmux-intray";
}

async function getTmuxValue(format: string): Promise<string> {
  try {
    const { stdout } = await execFileAsync("tmux", ["display-message", "-p", format], {
      env: process.env,
    });
    return stdout.trim();
  } catch {
    return "";
  }
}

async function getTmuxContext(): Promise<{ session: string; window: string; pane: string }> {
  const [session, window, paneRaw] = await Promise.all([
    getTmuxValue("#{session_id}"),
    getTmuxValue("#{window_id}"),
    getTmuxValue("#{pane_id}"),
  ]);

  const pane = paneRaw && !paneRaw.startsWith("%") ? `%${paneRaw}` : paneRaw;
  return { session, window, pane };
}

async function notify(level: "info" | "error" | "warning", message: string): Promise<void> {
  try {
    const cmd = getTmuxIntrayCommand();
    const ctx = await getTmuxContext();

    const args: string[] = ["add", `--level=${level}`];
    if (ctx.session) args.push(`--session=${ctx.session}`);
    if (ctx.window) args.push(`--window=${ctx.window}`);
    if (ctx.pane) args.push(`--pane=${ctx.pane}`);
    args.push("--", message);

    await log(`notify: ${cmd} ${args.join(" ")}`);
    await execFileAsync(cmd, args, { env: process.env });
  } catch (error) {
    await logError("notify", error);
  }
}

export default function tmuxIntrayExtension(pi: ExtensionAPI) {
  pi.on("agent_end", async () => {
    await notify("info", "Task completed");
  });

  pi.on("tool_result", async (event) => {
    if (event.isError) {
      await notify("error", `Tool error: ${event.toolName}`);
    }
  });

  pi.on("session_shutdown", async () => {
    await notify("warning", "Session shutdown");
  });
}
