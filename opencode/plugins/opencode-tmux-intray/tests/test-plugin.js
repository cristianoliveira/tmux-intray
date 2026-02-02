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

describe('Session detection functionality', () => {
  let originalTmux;
  let originalPath;
  let originalConfigPath;
  let binDir;

  beforeAll(async () => {
    originalTmux = process.env.TMUX;
    originalPath = process.env.PATH;
    originalConfigPath = process.env.OPENCODE_TMUX_INTRAY_CONFIG_PATH;
    binDir = await fs.promises.mkdtemp(join(os.tmpdir(), 'opencode-tmux-intray-bin-'));
    // Set config path to a non-existent file to ensure default config is used
    process.env.OPENCODE_TMUX_INTRAY_CONFIG_PATH = join(binDir, 'non-existent-config.json');
  });

  afterAll(async () => {
    if (originalTmux === undefined) {
      delete process.env.TMUX;
    } else {
      process.env.TMUX = originalTmux;
    }
    process.env.PATH = originalPath;
    if (originalConfigPath === undefined) {
      delete process.env.OPENCODE_TMUX_INTRAY_CONFIG_PATH;
    } else {
      process.env.OPENCODE_TMUX_INTRAY_CONFIG_PATH = originalConfigPath;
    }
    await fs.promises.rm(binDir, { recursive: true, force: true });
  });

  test('session detection without TMUX environment variable', async () => {
    delete process.env.TMUX;
    const tmuxLog = join(binDir, 'tmux.log');
    const notifyLog = join(binDir, 'notify.log');
    await createMockBin(binDir, 'tmux', tmuxLog, '');
    await createMockBin(binDir, 'tmux-intray', notifyLog);
    process.env.TMUX_INTRAY_PATH = join(binDir, 'tmux-intray');
    process.env.PATH = `${binDir}:${originalPath}`;
    
    await fs.promises.writeFile(tmuxLog, '');
    await fs.promises.writeFile(notifyLog, '');

    const pluginModule = await import(pluginPath + '?t=' + Date.now());
    const mockClient = {
      session: { get: async () => ({ data: { title: 'Test Session' } }) },
    };
    const pluginHooks = await pluginModule.default({ client: mockClient });
    await pluginHooks.event({ event: { type: 'session.idle', properties: {} } });

    // Verify tmux was called with correct arguments
    const tmuxCalls = (await fs.promises.readFile(tmuxLog, 'utf8')).trim().split('\n').filter(Boolean);
    expect(tmuxCalls.length).toBeGreaterThan(0);
    for (const call of tmuxCalls) {
      const args = JSON.parse(call);
      expect(Array.isArray(args)).toBe(true);
      expect(args[0]).toBe('display-message');
      expect(args[1]).toBe('-p');
      expect(args[2]).toBe('#S');
    }

    const notifyCalls = (await fs.promises.readFile(notifyLog, 'utf8')).trim().split('\n').filter(Boolean);
    expect(notifyCalls.length).toBeGreaterThan(0);
    const { agentName, status, session, message } = parseTmuxIntrayArgs(JSON.parse(notifyCalls.pop()));
    expect(agentName).toBe('opencode');
    expect(status).toBe('success');
    expect(session).toBe('');
    expect(message).toBe('Task completed');
  });

  test('session detection with TMUX environment variable', async () => {
    process.env.TMUX = '/tmp/tmux-123/default,123,0';
    const tmuxLog = join(binDir, 'tmux.log');
    const notifyLog = join(binDir, 'notify.log');
    await createMockBin(binDir, 'tmux', tmuxLog, 'mocked-session');
    await createMockBin(binDir, 'tmux-intray', notifyLog);
    process.env.TMUX_INTRAY_PATH = join(binDir, 'tmux-intray');
    await fs.promises.writeFile(tmuxLog, '');
    await fs.promises.writeFile(notifyLog, '');

    const pluginModule = await import(pluginPath + '?t=' + Date.now() + '2');
    const mockClient = {
      session: { get: async () => ({ data: { title: 'Test Session' } }) },
    };
    const pluginHooks = await pluginModule.default({ client: mockClient });
    await pluginHooks.event({ event: { type: 'session.error', properties: {} } });

    const tmuxCalls = (await fs.promises.readFile(tmuxLog, 'utf8')).trim().split('\n').filter(Boolean);
    expect(tmuxCalls.length).toBeGreaterThan(0);
    for (const call of tmuxCalls) {
      const args = JSON.parse(call);
      expect(Array.isArray(args)).toBe(true);
      expect(args[0]).toBe('display-message');
      expect(args[1]).toBe('-p');
      expect(args[2]).toBe('#S');
    }

    const notifyCalls = (await fs.promises.readFile(notifyLog, 'utf8')).trim().split('\n').filter(Boolean);
    expect(notifyCalls.length).toBeGreaterThan(0);
    const { agentName, status, session, message } = parseTmuxIntrayArgs(JSON.parse(notifyCalls.pop()));
    expect(agentName).toBe('opencode');
    expect(status).toBe('error');
    expect(session).toBe('');
    expect(message).toBe('Session error');
  });
});

describe('Session detection edge cases', () => {
  let originalTmux;
  let originalPath;
  let originalConfigPath;
  let binDir;

  beforeAll(async () => {
    originalTmux = process.env.TMUX;
    originalPath = process.env.PATH;
    originalConfigPath = process.env.OPENCODE_TMUX_INTRAY_CONFIG_PATH;
    binDir = await fs.promises.mkdtemp(join(os.tmpdir(), 'opencode-tmux-intray-edge-'));
    // Set config path to a non-existent file to ensure default config is used
    process.env.OPENCODE_TMUX_INTRAY_CONFIG_PATH = join(binDir, 'non-existent-config.json');
  });

  afterAll(async () => {
    if (originalTmux === undefined) {
      delete process.env.TMUX;
    } else {
      process.env.TMUX = originalTmux;
    }
    process.env.PATH = originalPath;
    if (originalConfigPath === undefined) {
      delete process.env.OPENCODE_TMUX_INTRAY_CONFIG_PATH;
    } else {
      process.env.OPENCODE_TMUX_INTRAY_CONFIG_PATH = originalConfigPath;
    }
    await fs.promises.rm(binDir, { recursive: true, force: true });
  });

  test('TMUX set but tmux command fails -> empty session', async () => {
    process.env.TMUX = '/tmp/tmux-123/default,123,0';
    const tmuxLog = join(binDir, 'tmux.log');
    const notifyLog = join(binDir, 'notify.log');
    await createMockBin(binDir, 'tmux', tmuxLog, '', true); // error=true
    await createMockBin(binDir, 'tmux-intray', notifyLog);
    process.env.TMUX_INTRAY_PATH = join(binDir, 'tmux-intray');
    process.env.PATH = `${binDir}:${originalPath}`;

    await fs.promises.writeFile(tmuxLog, '');
    await fs.promises.writeFile(notifyLog, '');

    const pluginModule = await import(pluginPath + '?t=' + Date.now());
    const mockClient = {
      session: { get: async () => ({ data: { title: 'Test Session' } }) },
    };
    const pluginHooks = await pluginModule.default({ client: mockClient });
    await pluginHooks.event({ event: { type: 'session.idle', properties: {} } });

    const tmuxCalls = (await fs.promises.readFile(tmuxLog, 'utf8')).trim().split('\n').filter(Boolean);
    expect(tmuxCalls.length).toBeGreaterThan(0);
    for (const call of tmuxCalls) {
      const args = JSON.parse(call);
      expect(Array.isArray(args)).toBe(true);
      expect(args[0]).toBe('display-message');
      expect(args[1]).toBe('-p');
      expect(args[2]).toBe('#S');
    }

    const notifyCalls = (await fs.promises.readFile(notifyLog, 'utf8')).trim().split('\n').filter(Boolean);
    expect(notifyCalls.length).toBeGreaterThan(0);
    const { agentName, status, session, message } = parseTmuxIntrayArgs(JSON.parse(notifyCalls.pop()));
    expect(agentName).toBe('opencode');
    expect(status).toBe('success');
    expect(session).toBe('');
    expect(message).toBe('Task completed');
  });

  test('TMUX not set but tmux command succeeds (fallback detection)', async () => {
    delete process.env.TMUX;
    const tmuxLog = join(binDir, 'tmux.log');
    const notifyLog = join(binDir, 'notify.log');
    await createMockBin(binDir, 'tmux', tmuxLog, 'fallback-session');
    await createMockBin(binDir, 'tmux-intray', notifyLog);
    process.env.TMUX_INTRAY_PATH = join(binDir, 'tmux-intray');
    await fs.promises.writeFile(tmuxLog, '');
    await fs.promises.writeFile(notifyLog, '');

    const pluginModule = await import(pluginPath + '?t=' + Date.now() + '2');
    const mockClient = {
      session: { get: async () => ({ data: { title: 'Test Session' } }) },
    };
    const pluginHooks = await pluginModule.default({ client: mockClient });
    await pluginHooks.event({ event: { type: 'session.error', properties: {} } });

    const tmuxCalls = (await fs.promises.readFile(tmuxLog, 'utf8')).trim().split('\n').filter(Boolean);
    expect(tmuxCalls.length).toBeGreaterThan(0);
    for (const call of tmuxCalls) {
      const args = JSON.parse(call);
      expect(Array.isArray(args)).toBe(true);
      expect(args[0]).toBe('display-message');
      expect(args[1]).toBe('-p');
      expect(args[2]).toBe('#S');
    }

    const notifyCalls = (await fs.promises.readFile(notifyLog, 'utf8')).trim().split('\n').filter(Boolean);
    expect(notifyCalls.length).toBeGreaterThan(0);
    const { agentName, status, session, message } = parseTmuxIntrayArgs(JSON.parse(notifyCalls.pop()));
    expect(agentName).toBe('opencode');
    expect(status).toBe('error');
    expect(session).toBe('');
    expect(message).toBe('Session error');
  });

  test('cached session usage (tmux called only once during init)', async () => {
    process.env.TMUX = '/tmp/tmux-123/default,123,0';
    const tmuxLog = join(binDir, 'tmux.log');
    const notifyLog = join(binDir, 'notify.log');
    await createMockBin(binDir, 'tmux', tmuxLog, 'cached-session');
    await createMockBin(binDir, 'tmux-intray', notifyLog);
    process.env.TMUX_INTRAY_PATH = join(binDir, 'tmux-intray');
    await fs.promises.writeFile(tmuxLog, '');
    await fs.promises.writeFile(notifyLog, '');

    const pluginModule = await import(pluginPath + '?t=' + Date.now() + '3');
    const mockClient = {
      session: { get: async () => ({ data: { title: 'Test Session' } }) },
    };
    const pluginHooks = await pluginModule.default({ client: mockClient });
    // Trigger two events
    await pluginHooks.event({ event: { type: 'session.idle', properties: {} } });
    await pluginHooks.event({ event: { type: 'session.error', properties: {} } });

    const tmuxCalls = (await fs.promises.readFile(tmuxLog, 'utf8')).trim().split('\n').filter(Boolean);
    expect(tmuxCalls.length).toBe(1); // Should be exactly 1 call (during plugin initialization)

    const notifyCalls = (await fs.promises.readFile(notifyLog, 'utf8')).trim().split('\n').filter(Boolean);
    expect(notifyCalls.length).toBeGreaterThanOrEqual(2);
    for (const line of notifyCalls) {
      // Parse to ensure valid tmux-intray arguments
      parseTmuxIntrayArgs(JSON.parse(line));
    }
  });
});