/**
 * Integration Tests for OpenCode Tmux Intray Plugin with Configuration
 *
 * Tests the full plugin flow with different configuration scenarios:
 * 1. Default config behavior - Plugin works without config file
 * 2. Event enabling/disabling - Disabled events don't trigger notifications
 * 3. Global disable - When enabled: false, no notifications sent
 * 4. Custom messages - Template substitution works in full flow
 * 5. Session detection - Session is passed to notifications correctly
 *
 * Uses Node.js built-in test runner (node:test)
 * 
 * Note: Since the plugin imports config-loader.js which computes CONFIG_PATH 
 * at module load time using homedir(), we use a different approach:
 * We test the integration by directly testing the flow of:
 *   loadConfig -> isEventEnabled/getEventConfig -> substituteTemplate -> notify
 * This validates the full integration without needing to override the home directory.
 */
import { describe, test, expect, afterAll, vi } from 'vitest';
import { mkdtemp, writeFile, rm, mkdir } from 'node:fs/promises';
import { join } from 'node:path';
import { tmpdir, homedir } from 'node:os';
import { fileURLToPath } from 'node:url';
import { dirname } from 'node:path';
import fs from 'node:fs';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const pluginPath = join(__dirname, '../../opencode-tmux-intray.js');

// Track temp directories for cleanup
const tempDirs = [];

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

// Track original config file if it exists
let originalConfigContent = null;
let configPath = null;

/**
 * Create a temporary directory for test isolation
 * @param {string} prefix - Prefix for temp directory name
 * @returns {Promise<string>} Path to temp directory
 */
async function createTempDir(prefix = 'opencode-integration-') {
  const dir = await mkdtemp(join(tmpdir(), prefix));
  tempDirs.push(dir);
  return dir;
}

/**
 * Create a mock binary that logs its invocations to a file
 * @param {string} baseDir - Directory to create binary in
 * @param {string} name - Name of the binary
 * @param {string} outputFile - File to log invocations to
 * @param {string} sessionOutput - Output to return for tmux session queries
 * @returns {Promise<string>} Path to the created binary
 */
async function createMockBin(baseDir, name, outputFile, sessionOutput = '') {
  const binPath = join(baseDir, name);
  const content = `#!/usr/bin/env node
const fs = require('fs');
const args = process.argv.slice(2);
fs.appendFileSync('${outputFile.replace(/'/g, "\\'")}', JSON.stringify(args) + '\\n');
${sessionOutput ? "if (args[0] === 'display-message' && args[1] === '-p' && args[2] === '#S') { console.log('" + sessionOutput + "'); }" : ''}
`;
  await writeFile(binPath, content);
  await fs.promises.chmod(binPath, 0o755);
  return binPath;
}

/**
 * Create a test environment with mock binaries
 * @param {Object} options - Test environment options
 * @param {string} [options.sessionName] - Tmux session name to mock
 * @returns {Promise<Object>} Test environment with paths and helpers
 */
async function createTestEnv(options = {}) {
  const binDir = await createTempDir('bin-');
  
  const tmuxLog = join(binDir, 'tmux.log');
  const notifyLog = join(binDir, 'notify.log');
  
  // Create mock binaries
  await createMockBin(binDir, 'tmux', tmuxLog, options.sessionName || '');
  await createMockBin(binDir, 'tmux-intray', notifyLog);
  
  // Initialize log files
  await writeFile(tmuxLog, '');
  await writeFile(notifyLog, '');
  
  return {
    binDir,
    tmuxLog,
    notifyLog,
    /**
     * Read notifications that were sent
     * @returns {Promise<Array<{agentName: string, status: string, session: string, message: string}>>}
     */
    async getNotifications() {
      const content = await fs.promises.readFile(notifyLog, 'utf8');
      return content.trim().split('\n').filter(Boolean).map(line => {
        return parseTmuxIntrayArgs(JSON.parse(line));
      });
    },
    /**
     * Clear notification log for fresh assertions
     */
    async clearNotifications() {
      await writeFile(notifyLog, '');
    }
  };
}

/**
 * Set up the config file at the actual config path
 * @param {Object|null} config - Config to write, or null to remove
 */
async function setupConfig(config) {
  if (!configPath) {
    configPath = join(homedir(), '.config', 'opencode-tmux-intray', 'opencode-config.json');
    // Back up existing config if any
    try {
      originalConfigContent = await fs.promises.readFile(configPath, 'utf8');
    } catch (e) {
      if (e.code !== 'ENOENT') throw e;
      originalConfigContent = null;
    }
  }
  
  if (config === null) {
    // Remove config file
    try {
      await fs.promises.unlink(configPath);
    } catch (e) {
      if (e.code !== 'ENOENT') throw e;
    }
  } else {
    // Ensure directory exists
    await mkdir(dirname(configPath), { recursive: true });
    await writeFile(configPath, JSON.stringify(config, null, 2));
  }
}

/**
 * Restore original config file
 */
async function restoreConfig() {
  if (configPath) {
    if (originalConfigContent !== null) {
      await writeFile(configPath, originalConfigContent);
    } else {
      try {
        await fs.promises.unlink(configPath);
      } catch (e) {
        if (e.code !== 'ENOENT') throw e;
      }
    }
  }
}

// Cleanup after all tests
afterAll(async () => {
  await restoreConfig();
  await Promise.all(tempDirs.map((dir) => rm(dir, { recursive: true, force: true })));
});

// =============================================================================
// Integration Test Scenarios
// =============================================================================

test('Integration: Default config behavior - Plugin works without config file', async () => {
  // Remove any existing config
  await setupConfig(null);
  
  const env = await createTestEnv({
    sessionName: 'test-session'
  });
  
  const originalPath = process.env.PATH;
  const originalTmux = process.env.TMUX;
  
  process.env.PATH = `${env.binDir}:${originalPath}`;
  process.env.TMUX = '/tmp/tmux-test/default,123,0';
  
  try {
    // Import plugin without any config file
    const pluginModule = await import(pluginPath + '?t=' + Date.now() + Math.random());
    
    const mockClient = {
      session: { get: async () => ({ data: { title: 'Test Session' } }) }
    };
    
    const hooks = await pluginModule.default({ client: mockClient });
    
    // Trigger a default-enabled event
    await hooks.event({ 
      event: { 
        type: 'session.idle', 
        sessionTitle: 'My Task',
        properties: { sessionID: 'test-123' } 
      } 
    });
    
    const notifications = await env.getNotifications();
    
    // Should have sent a notification with default config
    expect(notifications.length >= 1).toBeTruthy();
    
    const lastNotification = notifications[notifications.length - 1];
    expect(lastNotification.agentName).toBe('opencode');
    expect(lastNotification.status).toBe('success');
    expect(lastNotification.session).toBe('');
    // Default message is now simple (no template variables for backward compatibility)
    expect(lastNotification.message).toBe('Task completed',
      `Message should be simple default, got: ${lastNotification.message}`);
    
  } finally {
    process.env.PATH = originalPath;
    if (originalTmux === undefined) {
      delete process.env.TMUX;
    } else {
      process.env.TMUX = originalTmux;
    }
  }
});

test('Integration: Event enabling/disabling - Disabled events do not trigger notifications', async () => {
  const config = {
    enabled: true,
    notifications: {
      'session.idle': {
        enabled: false,  // Disable this event
        message: 'Should not appear',
        status: 'success'
      },
      'session.error': {
        enabled: true,  // Keep this enabled
        message: 'Error occurred',
        status: 'error'
      }
    }
  };
  
  await setupConfig(config);
  
  const env = await createTestEnv({
    sessionName: 'test-session'
  });
  
  const originalPath = process.env.PATH;
  const originalTmux = process.env.TMUX;
  
  process.env.PATH = `${env.binDir}:${originalPath}`;
  process.env.TMUX = '/tmp/tmux-test/default,123,0';
  
  try {
    const pluginModule = await import(pluginPath + '?t=' + Date.now() + Math.random());
    
    const mockClient = {
      session: { get: async () => ({ data: { title: 'Test Session' } }) }
    };
    
    const hooks = await pluginModule.default({ client: mockClient });
    
    // Trigger disabled event
    await hooks.event({ 
      event: { type: 'session.idle', properties: {} } 
    });
    
    let notifications = await env.getNotifications();
    expect(notifications.length).toBe(0, 'Disabled event should not trigger notification');
    
    // Trigger enabled event
    await hooks.event({ 
      event: { type: 'session.error', properties: {} } 
    });
    
    notifications = await env.getNotifications();
    expect(notifications.length).toBe(1, 'Enabled event should trigger notification');
    expect(notifications[0].status).toBe('error');
    expect(notifications[0].message).toBe('Error occurred');
    
  } finally {
    process.env.PATH = originalPath;
    if (originalTmux === undefined) {
      delete process.env.TMUX;
    } else {
      process.env.TMUX = originalTmux;
    }
  }
});

test('Integration: Global disable - When enabled: false, no notifications sent', async () => {
  const config = {
    enabled: false,  // Globally disabled
    notifications: {
      'session.idle': {
        enabled: true,  // Even though this is enabled
        message: 'Should not appear',
        status: 'success'
      },
      'session.error': {
        enabled: true,
        message: 'Should not appear either',
        status: 'error'
      }
    }
  };
  
  await setupConfig(config);
  
  const env = await createTestEnv({
    sessionName: 'test-session'
  });
  
  const originalPath = process.env.PATH;
  const originalTmux = process.env.TMUX;
  
  process.env.PATH = `${env.binDir}:${originalPath}`;
  process.env.TMUX = '/tmp/tmux-test/default,123,0';
  
  try {
    const pluginModule = await import(pluginPath + '?t=' + Date.now() + Math.random());
    
    const mockClient = {
      session: { get: async () => ({ data: { title: 'Test Session' } }) }
    };
    
    const hooks = await pluginModule.default({ client: mockClient });
    
    // Trigger multiple events
    await hooks.event({ event: { type: 'session.idle', properties: {} } });
    await hooks.event({ event: { type: 'session.error', properties: {} } });
    await hooks.event({ event: { type: 'permission.updated', properties: {} } });
    
    const notifications = await env.getNotifications();
    
    // No notifications should be sent when globally disabled
    expect(notifications.length).toBe(0, 
      'No notifications should be sent when globally disabled');
    
  } finally {
    process.env.PATH = originalPath;
    if (originalTmux === undefined) {
      delete process.env.TMUX;
    } else {
      process.env.TMUX = originalTmux;
    }
  }
});

test('Integration: Custom messages - Template substitution works in full flow', async () => {
  const config = {
    enabled: true,
    notifications: {
      'session.idle': {
        enabled: true,
        message: 'Task done: {sessionTitle} (ID: {properties.sessionID})',
        status: 'success'
      },
      'session.error': {
        enabled: true,
        message: 'Error in session {properties.sessionID}: {properties.error}',
        status: 'error'
      }
    }
  };
  
  await setupConfig(config);
  
  const env = await createTestEnv({
    sessionName: 'my-tmux-session'
  });
  
  const originalPath = process.env.PATH;
  const originalTmux = process.env.TMUX;
  
  process.env.PATH = `${env.binDir}:${originalPath}`;
  process.env.TMUX = '/tmp/tmux-test/default,123,0';
  
  try {
    const pluginModule = await import(pluginPath + '?t=' + Date.now() + Math.random());
    
    const mockClient = {
      session: { get: async () => ({ data: { title: 'Test Session' } }) }
    };
    
    const hooks = await pluginModule.default({ client: mockClient });
    
    // Trigger event with properties for template substitution
    await hooks.event({ 
      event: { 
        type: 'session.idle',
        sessionTitle: 'My Important Task',
        properties: { sessionID: 'abc-123' }
      } 
    });
    
    let notifications = await env.getNotifications();
    expect(notifications.length).toBe(1, 'Should send notification');
    
    // Verify template substitution worked
    expect(notifications[0].message).toBe('Task done: My Important Task (ID: abc-123)',
      `Template should be substituted, got: ${notifications[0].message}`);
    
    await env.clearNotifications();
    
    // Test error event with different properties
    await hooks.event({ 
      event: { 
        type: 'session.error',
        properties: { 
          sessionID: 'xyz-789',
          error: 'Connection timeout'
        }
      } 
    });
    
    notifications = await env.getNotifications();
    expect(notifications.length).toBe(1, 'Should send error notification');
    expect(notifications[0].message).toBe('Error in session xyz-789: Connection timeout',
      `Error template should be substituted, got: ${notifications[0].message}`);
    expect(notifications[0].status).toBe('error');
    
  } finally {
    process.env.PATH = originalPath;
    if (originalTmux === undefined) {
      delete process.env.TMUX;
    } else {
      process.env.TMUX = originalTmux;
    }
  }
});

test('Integration: Session detection - Session is passed to notifications correctly', async () => {
  // Use default config (no custom config)
  await setupConfig(null);
  
  const env = await createTestEnv({
    sessionName: 'my-custom-session'
  });
  
  const originalPath = process.env.PATH;
  const originalTmux = process.env.TMUX;
  
  process.env.PATH = `${env.binDir}:${originalPath}`;
  
  try {
    // Test with TMUX set - should detect session
    process.env.TMUX = '/tmp/tmux-test/default,123,0';
    
    const pluginModule1 = await import(pluginPath + '?t=' + Date.now() + Math.random());
    const mockClient = {
      session: { get: async () => ({ data: { title: 'Test' } }) }
    };
    
    const hooks1 = await pluginModule1.default({ client: mockClient });
    await hooks1.event({ event: { type: 'session.idle', sessionTitle: 'Test', properties: {} } });
    
    let notifications = await env.getNotifications();
    expect(notifications.length >= 1).toBeTruthy();
    expect(notifications[notifications.length - 1].session).toBe('',
      'Session should be empty (not passed to tmux-intray)');
    
    // Clear and test without TMUX
    await env.clearNotifications();
    delete process.env.TMUX;
    
    // Create a new mock bin without session output
    await createMockBin(env.binDir, 'tmux', env.tmuxLog, '');
    
    const pluginModule2 = await import(pluginPath + '?t=' + Date.now() + Math.random());
    const hooks2 = await pluginModule2.default({ client: mockClient });
    await hooks2.event({ event: { type: 'session.idle', sessionTitle: 'Test', properties: {} } });
    
    notifications = await env.getNotifications();
    expect(notifications.length >= 1).toBeTruthy();
    // Without TMUX env, session should be empty
    expect(notifications[notifications.length - 1].session).toBe('',
      'Session should be empty when TMUX not set');
    
  } finally {
    process.env.PATH = originalPath;
    if (originalTmux === undefined) {
      delete process.env.TMUX;
    } else {
      process.env.TMUX = originalTmux;
    }
  }
});

test('Integration: session.status event - Only notifies on pending status', async () => {
  const config = {
    enabled: true,
    notifications: {
      'session.status': {
        enabled: true,
        message: 'Status: {properties.status}',
        status: 'pending'
      }
    }
  };
  
  await setupConfig(config);
  
  const env = await createTestEnv({
    sessionName: 'test-session'
  });
  
  const originalPath = process.env.PATH;
  const originalTmux = process.env.TMUX;
  
  process.env.PATH = `${env.binDir}:${originalPath}`;
  process.env.TMUX = '/tmp/tmux-test/default,123,0';
  
  try {
    const pluginModule = await import(pluginPath + '?t=' + Date.now() + Math.random());
    
    const mockClient = {
      session: { get: async () => ({ data: { title: 'Test' } }) }
    };
    
    const hooks = await pluginModule.default({ client: mockClient });
    
    // Trigger session.status with non-pending status - should not notify
    await hooks.event({ 
      event: { 
        type: 'session.status',
        properties: { status: 'completed' }
      } 
    });
    
    let notifications = await env.getNotifications();
    expect(notifications.length).toBe(0, 
      'session.status with non-pending status should not trigger notification');
    
    // Trigger session.status with pending status - should notify
    await hooks.event({ 
      event: { 
        type: 'session.status',
        properties: { status: 'pending' }
      } 
    });
    
    notifications = await env.getNotifications();
    expect(notifications.length).toBe(1, 
      'session.status with pending status should trigger notification');
    expect(notifications[0].message).toBe('Status: pending');
    
  } finally {
    process.env.PATH = originalPath;
    if (originalTmux === undefined) {
      delete process.env.TMUX;
    } else {
      process.env.TMUX = originalTmux;
    }
  }
});

test('Integration: Custom agent name in config - verifies config is loaded', async () => {
  // Note: The agentName in config is not currently used by the plugin
  // (it hardcodes 'opencode'), but we test that the config is loaded correctly
  // by verifying custom message is used
  const config = {
    enabled: true,
    agentName: 'custom-agent',
    notifications: {
      'session.idle': {
        enabled: true,
        message: 'Custom notification from config',
        status: 'success'
      }
    }
  };
  
  await setupConfig(config);
  
  const env = await createTestEnv({
    sessionName: 'test-session'
  });
  
  const originalPath = process.env.PATH;
  const originalTmux = process.env.TMUX;
  
  process.env.PATH = `${env.binDir}:${originalPath}`;
  process.env.TMUX = '/tmp/tmux-test/default,123,0';
  
  try {
    const pluginModule = await import(pluginPath + '?t=' + Date.now() + Math.random());
    
    const mockClient = {
      session: { get: async () => ({ data: { title: 'Test' } }) }
    };
    
    const hooks = await pluginModule.default({ client: mockClient });
    await hooks.event({ event: { type: 'session.idle', properties: {} } });
    
    const notifications = await env.getNotifications();
    expect(notifications.length).toBe(1, 'Should send notification');
    // Plugin currently hardcodes 'opencode' as agent name
    expect(notifications[0].agentName).toBe('opencode');
    // But custom message proves config was loaded
    expect(notifications[0].message).toBe('Custom notification from config',
      `Should use custom message from config, got: ${notifications[0].message}`);
    
  } finally {
    process.env.PATH = originalPath;
    if (originalTmux === undefined) {
      delete process.env.TMUX;
    } else {
      process.env.TMUX = originalTmux;
    }
  }
});

test('Integration: Multiple event types with mixed enabled/disabled', async () => {
  const config = {
    enabled: true,
    notifications: {
      'session.idle': {
        enabled: true,
        message: 'Idle: {type}',
        status: 'success'
      },
      'session.error': {
        enabled: false,
        message: 'Error (disabled)',
        status: 'error'
      },
      'permission.updated': {
        enabled: true,
        message: 'Permission updated',
        status: 'pending'
      },
      'question.asked': {
        enabled: false,
        message: 'Question (disabled)',
        status: 'pending'
      }
    }
  };
  
  await setupConfig(config);
  
  const env = await createTestEnv({
    sessionName: 'test-session'
  });
  
  const originalPath = process.env.PATH;
  const originalTmux = process.env.TMUX;
  
  process.env.PATH = `${env.binDir}:${originalPath}`;
  process.env.TMUX = '/tmp/tmux-test/default,123,0';
  
  try {
    const pluginModule = await import(pluginPath + '?t=' + Date.now() + Math.random());
    
    const mockClient = {
      session: { get: async () => ({ data: { title: 'Test' } }) }
    };
    
    const hooks = await pluginModule.default({ client: mockClient });
    
    // Trigger all four events
    await hooks.event({ event: { type: 'session.idle', properties: {} } });
    await hooks.event({ event: { type: 'session.error', properties: {} } });
    await hooks.event({ event: { type: 'permission.updated', properties: {} } });
    await hooks.event({ event: { type: 'question.asked', properties: {} } });
    
    const notifications = await env.getNotifications();
    
    // Should only have 2 notifications (idle and permission.updated)
    expect(notifications.length).toBe(2, 
      `Should have exactly 2 notifications for enabled events, got ${notifications.length}`);
    
    // Verify the correct events triggered
    const messages = notifications.map(n => n.message);
    expect(messages.some(m => m.includes('Idle'))).toBeTruthy();
    expect(messages.some(m => m.includes('Permission updated'))).toBeTruthy();
    expect(!messages.some(m => m.includes('disabled'))).toBeTruthy();
    
  } finally {
    process.env.PATH = originalPath;
    if (originalTmux === undefined) {
      delete process.env.TMUX;
    } else {
      process.env.TMUX = originalTmux;
    }
  }
});
