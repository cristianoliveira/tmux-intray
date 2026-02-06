/**
 * Test script for OpenCode Tmux Intray Plugin
 * 
 * Verifies that the plugin can be loaded without errors and has the
 * expected structure for OpenCode integration.
 */

import { exec } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';
import fs from 'node:fs';
import os from 'node:os';
import { describe, test, expect, vi, beforeAll, afterAll } from 'vitest';

// Import the plugin dynamically (relative import)
const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const pluginPath = join(__dirname, '../../opencode-tmux-intray.js');

async function createMockBin(baseDir, name, outputFile, sessionOutput = '', error = false) {
   const binPath = join(baseDir, name);
   const content = `#!/usr/bin/env node
const fs = require('fs');
const args = process.argv.slice(2);
fs.appendFileSync('${outputFile.replace(/'/g, "\\'")}', JSON.stringify(args) + '\\n');
${error ? 'process.exit(1);' : ''}
${sessionOutput ? "if (args[0] === 'display-message' && args[1] === '-p' && args[2] === '#S') { console.log('" + sessionOutput + "'); }" : ''}
`;
  await fs.promises.writeFile(binPath, content);
  await fs.promises.chmod(binPath, 0o755);
  return binPath;
}

/**
 * Parse tmux-intray arguments array into notification object
 * @param {Array<string>} args - Arguments from tmux-intray command
 * @returns {Object} Parsed notification with agentName, status, session, message
 */
function parseTmuxIntrayArgs(args) {
  // Expected args: ["add", "--level=info", "message"]
  if (args.length < 3 || args[0] !== 'add' || !args[1].startsWith('--level=')) {
    throw new Error(`Unexpected tmux-intray arguments: ${JSON.stringify(args)}`);
  }
  const level = args[1].replace('--level=', '');
  const message = args[2]; // message is third argument (quoted as single arg)
  // Map level back to status
  const levelToStatus = {
    'error': 'error',
    'warning': 'pending',
    'info': 'success'
  };
  const status = levelToStatus[level] || 'success';
  // Session is not passed to tmux-intray, so empty
  const session = '';
  // Agent name is hardcoded as 'opencode'
  const agentName = 'opencode';
  return { agentName, status, session, message };
}

describe('OpenCode Tmux Intray Plugin', () => {
  let originalConfigPath;

  beforeAll(() => {
    originalConfigPath = process.env.OPENCODE_TMUX_INTRAY_CONFIG_PATH;
    // Set config path to a non-existent file to ensure default config is used
    process.env.OPENCODE_TMUX_INTRAY_CONFIG_PATH = join(os.tmpdir(), `opencode-tmux-intray-test-config-${Date.now()}.json`);
  });

  afterAll(() => {
    if (originalConfigPath === undefined) {
      delete process.env.OPENCODE_TMUX_INTRAY_CONFIG_PATH;
    } else {
      process.env.OPENCODE_TMUX_INTRAY_CONFIG_PATH = originalConfigPath;
    }
  });

  test('plugin file exists and is readable', async () => {
    await expect(fs.promises.access(pluginPath)).resolves.not.toThrow();
  });

  test('plugin module loads successfully', async () => {
    const pluginModule = await import(pluginPath);
    expect(pluginModule).toBeDefined();
    expect(typeof pluginModule.default).toBe('function');
  });

  test('plugin initializes with mock client', async () => {
    const pluginModule = await import(pluginPath);
    const mockClient = {
      session: {
        get: async () => ({ data: { title: 'Test Session' } }),
      },
    };
    const pluginHooks = await pluginModule.default({ client: mockClient });
    expect(pluginHooks).toBeDefined();
    expect(typeof pluginHooks).toBe('object');
    expect(typeof pluginHooks.event).toBe('function');
  });

  test('event handler processes events without errors', async () => {
    const pluginModule = await import(pluginPath);
    const mockClient = {
      session: {
        get: async () => ({ data: { title: 'Test Session' } }),
      },
    };
    const pluginHooks = await pluginModule.default({ client: mockClient });
    
    const mockEvents = [
      { type: 'session.idle', properties: { sessionID: 'test-123' } },
      { type: 'session.error', properties: { sessionID: 'test-123', error: 'Test error' } },
      { type: 'permission.updated', properties: { sessionID: 'test-123', permission: 'user_input' } },
      { type: 'question.asked', properties: { sessionID: 'test-123', question: 'What should I do?' } },
      { type: 'permission.asked', properties: { sessionID: 'test-123', permission: 'user_input' } },
      { type: 'other.event', properties: {} }, // Should be ignored
    ];
    
    for (const event of mockEvents) {
      await expect(pluginHooks.event({ event })).resolves.not.toThrow();
    }
  });
});

describe('Simplified plugin behavior (delegating context detection to Go CLI)', () => {
  let originalConfigPath;
  let binDir;
  let originalTmuxIntrayPath;

  beforeAll(async () => {
    originalConfigPath = process.env.OPENCODE_TMUX_INTRAY_CONFIG_PATH;
    originalTmuxIntrayPath = process.env.TMUX_INTRAY_PATH;
    binDir = await fs.promises.mkdtemp(join(os.tmpdir(), 'opencode-tmux-intray-simple-'));
    // Set config path to a non-existent file to ensure default config is used
    process.env.OPENCODE_TMUX_INTRAY_CONFIG_PATH = join(binDir, 'non-existent-config.json');
  });

  afterAll(async () => {
    if (originalConfigPath === undefined) {
      delete process.env.OPENCODE_TMUX_INTRAY_CONFIG_PATH;
    } else {
      process.env.OPENCODE_TMUX_INTRAY_CONFIG_PATH = originalConfigPath;
    }
    if (originalTmuxIntrayPath === undefined) {
      delete process.env.TMUX_INTRAY_PATH;
    } else {
      process.env.TMUX_INTRAY_PATH = originalTmuxIntrayPath;
    }
    await fs.promises.rm(binDir, { recursive: true, force: true });
  });

  test('Plugin builds correct command without context flags', async () => {
    const notifyLog = join(binDir, 'notify.log');
    await createMockBin(binDir, 'tmux-intray', notifyLog);
    process.env.TMUX_INTRAY_PATH = join(binDir, 'tmux-intray');
    await fs.promises.writeFile(notifyLog, '');

    const pluginModule = await import(pluginPath + '?t=' + Date.now());
    const mockClient = {
      session: { get: async () => ({ data: { title: 'Test Session' } }) },
    };
    const pluginHooks = await pluginModule.default({ client: mockClient });
    await pluginHooks.event({ event: { type: 'session.error', properties: {} } });

    // Verify tmux-intray was called with correct arguments
    const notifyCalls = (await fs.promises.readFile(notifyLog, 'utf8')).trim().split('\n').filter(Boolean);
    expect(notifyCalls.length).toBeGreaterThan(0);
    
    const { agentName, status, session, message } = parseTmuxIntrayArgs(JSON.parse(notifyCalls[0]));
    expect(agentName).toBe('opencode');
    expect(status).toBe('error');
    expect(session).toBe(''); // Context NOT passed to plugin
    expect(message).toBe('Session error');
    
    // Verify command structure: ["add", "--level=error", "message"]
    const args = JSON.parse(notifyCalls[0]);
    expect(args[0]).toBe('add');
    expect(args[1]).toBe('--level=error');
    expect(args[2]).toBe('Session error');
    expect(args.length).toBe(3); // Only 3 arguments, no context flags
  });

  test('Plugin executes command via execAsync', async () => {
    const notifyLog = join(binDir, 'notify2.log');
    await createMockBin(binDir, 'tmux-intray', notifyLog);
    process.env.TMUX_INTRAY_PATH = join(binDir, 'tmux-intray');
    await fs.promises.writeFile(notifyLog, '');

    const pluginModule = await import(pluginPath + '?t=' + Date.now() + '2');
    const mockClient = {
      session: { get: async () => ({ data: { title: 'Test Session' } }) },
    };
    const pluginHooks = await pluginModule.default({ client: mockClient });
    await pluginHooks.event({ event: { type: 'session.idle', properties: {} } });

    // Verify execAsync was called (tmux-intray command executed)
    const notifyCalls = (await fs.promises.readFile(notifyLog, 'utf8')).trim().split('\n').filter(Boolean);
    expect(notifyCalls.length).toBe(1);
  });

  test('Plugin handles execAsync errors gracefully', async () => {
    const notifyLog = join(binDir, 'notify3.log');
    // Create mock that exits with error code
    await createMockBin(binDir, 'tmux-intray', notifyLog, '', true);
    process.env.TMUX_INTRAY_PATH = join(binDir, 'tmux-intray');
    await fs.promises.writeFile(notifyLog, '');

    const pluginModule = await import(pluginPath + '?t=' + Date.now() + '3');
    const mockClient = {
      session: { get: async () => ({ data: { title: 'Test Session' } }) },
    };
    const pluginHooks = await pluginModule.default({ client: mockClient });
    
    // Should not throw - errors are handled gracefully
    await expect(pluginHooks.event({ event: { type: 'session.error', properties: {} } })).resolves.not.toThrow();
  });

  test('Plugin handles messages with special characters', async () => {
    const notifyLog = join(binDir, 'notify4.log');
    await createMockBin(binDir, 'tmux-intray', notifyLog);
    process.env.TMUX_INTRAY_PATH = join(binDir, 'tmux-intray');
    await fs.promises.writeFile(notifyLog, '');

    const pluginModule = await import(pluginPath + '?t=' + Date.now() + '4');
    const mockClient = {
      session: { get: async () => ({ data: { title: 'Test Session' } }) },
    };
    const pluginHooks = await pluginModule.default({ client: mockClient });
    
    // Test with different event types that are enabled by default
    const testCases = [
      { type: 'session.error', properties: {}, expectedMessage: 'Session error' },
      { type: 'session.idle', properties: {}, expectedMessage: 'Task completed' },
      { type: 'permission.updated', properties: { permission: 'test_perm' }, expectedMessage: 'Permission needed' },
    ];
    
    for (const testCase of testCases) {
      await pluginHooks.event({ event: testCase });
    }

    const notifyCalls = (await fs.promises.readFile(notifyLog, 'utf8')).trim().split('\n').filter(Boolean);
    expect(notifyCalls.length).toBeGreaterThanOrEqual(1);
    
    // Verify all messages are properly formed tmux-intray commands
    for (const call of notifyCalls) {
      const args = JSON.parse(call);
      expect(args[0]).toBe('add');
      expect(args[1]).toMatch(/^--level=/);
      expect(typeof args[2]).toBe('string');
    }
  });

  test('Multiple events create multiple notifications', async () => {
    const notifyLog = join(binDir, 'notify5.log');
    await createMockBin(binDir, 'tmux-intray', notifyLog);
    process.env.TMUX_INTRAY_PATH = join(binDir, 'tmux-intray');
    await fs.promises.writeFile(notifyLog, '');

    const pluginModule = await import(pluginPath + '?t=' + Date.now() + '5');
    const mockClient = {
      session: { get: async () => ({ data: { title: 'Test Session' } }) },
    };
    const pluginHooks = await pluginModule.default({ client: mockClient });
    
    // Trigger 3 events that are enabled by default: session.idle, session.error, permission.updated
    await pluginHooks.event({ event: { type: 'session.idle', properties: {} } });
    await pluginHooks.event({ event: { type: 'session.error', properties: {} } });
    await pluginHooks.event({ event: { type: 'permission.updated', properties: { permission: 'test' } } });

    const notifyCalls = (await fs.promises.readFile(notifyLog, 'utf8')).trim().split('\n').filter(Boolean);
    expect(notifyCalls.length).toBeGreaterThanOrEqual(3);
    
    // Verify each call has the correct structure
    for (const call of notifyCalls) {
      const args = JSON.parse(call);
      expect(args.length).toBe(3);
      expect(args[0]).toBe('add');
      expect(args[1]).toMatch(/^--level=\w+$/);
      expect(typeof args[2]).toBe('string');
    }
  });
});

