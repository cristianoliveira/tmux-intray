/**
 * Real-world test of tmux-intray integration
 * 
 * Tests that the plugin can send notifications to the actual tmux-intray command
 * and logs to /tmp/opencode-tmux-intray.log.
 */

import { exec } from 'node:child_process';
import { promisify } from 'node:util';
import { promises as fs, constants } from 'node:fs';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { dirname } from 'node:path';
import { describe, test, expect, beforeAll, afterAll } from 'vitest';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const pluginPath = join(__dirname, '../../opencode-tmux-intray.js');

const execAsync = promisify(exec);

function getLocalTmuxIntrayPath() {
  const localBinary = join(__dirname, '../../../../bin/tmux-intray');
  return localBinary;
}

async function isLocalTmuxIntrayExecutable() {
  try {
    await fs.access(getLocalTmuxIntrayPath(), constants.X_OK);
    return true;
  } catch {
    return false;
  }
}

async function cleanupLogs() {
  const logFile = '/tmp/opencode-tmux-intray.log';
  try {
    await fs.unlink(logFile);
  } catch (err) {
    // ignore if not exists
  }
}

async function readLogs() {
  const logFile = '/tmp/opencode-tmux-intray.log';
  try {
    const content = await fs.readFile(logFile, 'utf8');
    return content.trim().split('\n').filter(Boolean);
  } catch (err) {
    return [];
  }
}

const hasTmuxIntray = async () => {
  return await isLocalTmuxIntrayExecutable();
};

describe('Real-world tmux-intray integration', () => {
  let tmuxIntrayInstalled = false;
  
  beforeAll(async () => {
    await cleanupLogs();
    tmuxIntrayInstalled = await hasTmuxIntray();
  });

  afterAll(async () => {
    await cleanupLogs();
  });

  test('plugin loads and initializes', async () => {
    const pluginModule = await import(pluginPath);
    expect(pluginModule).toBeDefined();
    expect(typeof pluginModule.default).toBe('function');
    
    const mockClient = {
      session: {
        get: async () => ({ data: { title: 'Test Session' } }),
      },
    };
    
    const hooks = await pluginModule.default({ client: mockClient });
    expect(hooks).toBeDefined();
    expect(typeof hooks.event).toBe('function');
  });

  test.skipIf(!tmuxIntrayInstalled)('sends notification to tmux-intray command', async () => {
    const pluginModule = await import(pluginPath);
    const mockClient = {
      session: {
        get: async () => ({ data: { title: 'Test Session' } }),
      },
    };
    const hooks = await pluginModule.default({ client: mockClient });
    
    // Trigger a session.idle event (default enabled)
    await hooks.event({
      event: {
        type: 'session.idle',
        sessionTitle: 'Real Test Task',
        properties: { sessionID: 'test-real-123' },
      },
    });
    
    // Wait a moment for async operations (exec may take time)
    await new Promise(resolve => setTimeout(resolve, 500));
    
    // Check that log file was created
    const logs = await readLogs();
    expect(logs.length).toBeGreaterThan(0);
    
    // Parse last log entry
    const lastLog = logs[logs.length - 1];
    
    // Verify log contains expected fields
    expect(lastLog).toContain('NOTIFY:');
    expect(lastLog).toContain('status="success"');
  });

  test('tmux-intray command is available', async () => {
    await expect(execAsync(`${getLocalTmuxIntrayPath()} --help`)).resolves.not.toThrow();
  }, 10000); // longer timeout for external command
});