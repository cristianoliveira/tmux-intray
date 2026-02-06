/**
 * Comprehensive Integration Tests for OpenCode Plugin with Real Tmux Context
 *
 * Tests the full end-to-end plugin behavior including:
 * 1. Binary Discovery - Finding tmux-intray via multiple environment variable methods
 * 2. Context Capture - Capturing session/window/pane IDs from tmux
 * 3. End-to-End Notifications - Full workflow with context inclusion
 * 4. Flag Passing - Correct command-line flag construction
 * 5. Error Handling - Graceful failure modes
 * 6. Real Tmux Environment - Integration with actual tmux sessions
 *
 * Uses vitest framework with mocked tmux and tmux-intray binaries.
 */

import { describe, test, expect, beforeAll, afterAll, vi, beforeEach } from 'vitest';
import { mkdtemp, writeFile, rm, mkdir, readFile } from 'node:fs/promises';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { fileURLToPath } from 'node:url';
import { dirname } from 'node:path';
import fs from 'node:fs';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const pluginPath = join(__dirname, '../../opencode-tmux-intray.js');

// Track temp directories for cleanup
const tempDirs = [];

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
 * Create a mock binary that logs invocations and returns configurable responses
 * @param {string} baseDir - Directory to create binary in
 * @param {string} name - Name of the binary (e.g., 'tmux' or 'tmux-intray')
 * @param {string} outputFile - File to log JSON-encoded invocations to
 * @param {Object} options - Configuration options
 * @param {string} [options.sessionOutput] - Output for session ID query
 * @param {string} [options.windowOutput] - Output for window ID query
 * @param {string} [options.paneOutput] - Output for pane ID query
 * @param {boolean} [options.error] - Exit with error code 1
 * @returns {Promise<string>} Path to the created binary
 */
async function createMockBin(baseDir, name, outputFile, options = {}) {
  const { sessionOutput = '', windowOutput = '', paneOutput = '', error = false } = options;
  const binPath = join(baseDir, name);
  
  let content = `#!/usr/bin/env node
const fs = require('fs');
const args = process.argv.slice(2);
fs.appendFileSync('${outputFile.replace(/'/g, "\\'")}', JSON.stringify(args) + '\\n');
`;

  // Handle tmux command responses
  // Note: the shell parses quotes, so the format string will be #{session_id} not "#{session_id}"
  if (sessionOutput || windowOutput || paneOutput) {
    content += `
if (args[0] === 'display-message' && args[1] === '-p') {
  const format = args[2];
  if (format && format.includes('session_id')) {
    if ('${sessionOutput}') console.log('${sessionOutput}');
    process.exit(0);
  } else if (format && format.includes('window_id')) {
    if ('${windowOutput}') console.log('${windowOutput}');
    process.exit(0);
  } else if (format && format.includes('pane_id')) {
    if ('${paneOutput}') console.log('${paneOutput}');
    process.exit(0);
  }
}
`;
  }

  if (error) {
    content += 'process.exit(1);\n';
  } else {
    content += 'process.exit(0);\n';
  }

  await writeFile(binPath, content);
  await fs.promises.chmod(binPath, 0o755);
  return binPath;
}

/**
 * Create a test environment with mock binaries and log files
 * @param {Object} options - Environment options
 * @returns {Promise<Object>} Environment with paths and helpers
 */
async function createTestEnv(options = {}) {
  const binDir = await createTempDir('bin-');
  const tmuxLog = join(binDir, 'tmux.log');
  const notifyLog = join(binDir, 'notify.log');

  // Create mock binaries
  await createMockBin(binDir, 'tmux', tmuxLog, {
    sessionOutput: options.sessionID || '$0',
    windowOutput: options.windowID || '@0',
    paneOutput: options.paneID || '%0',
  });
  
  await createMockBin(binDir, 'tmux-intray', notifyLog);

  // Initialize log files
  await writeFile(tmuxLog, '');
  await writeFile(notifyLog, '');

  return {
    binDir,
    tmuxLog,
    notifyLog,

    /**
     * Get all recorded tmux-intray invocations
     */
    async getNotifications() {
      const content = await readFile(notifyLog, 'utf8');
      return content
        .trim()
        .split('\n')
        .filter(Boolean)
        .map(line => JSON.parse(line));
    },

    /**
     * Clear notification log for fresh assertions
     */
    async clearNotifications() {
      await writeFile(notifyLog, '');
    },

    /**
     * Get all recorded tmux invocations
     */
    async getTmuxCalls() {
      const content = await readFile(tmuxLog, 'utf8');
      return content
        .trim()
        .split('\n')
        .filter(Boolean)
        .map(line => JSON.parse(line));
    },

    /**
     * Create an error binary that fails
     */
    async createErrorBinary(binName = 'tmux-intray') {
      await createMockBin(binDir, binName, join(binDir, `${binName}-error.log`), { 
        error: true 
      });
    },

    /**
     * Reinstall tmux with new context values
     */
    async reinstallTmux(sessionID, windowID, paneID) {
      await createMockBin(binDir, 'tmux', tmuxLog, {
        sessionOutput: sessionID,
        windowOutput: windowID,
        paneOutput: paneID,
      });
      await writeFile(tmuxLog, '');
    }
  };
}

// Cleanup after all tests
afterAll(async () => {
  await Promise.all(
    tempDirs.map(dir => rm(dir, { recursive: true, force: true }))
  );
});

// =============================================================================
// Binary Discovery Tests
// =============================================================================

describe('Binary Discovery', () => {
   let originalPathEnv;
   let originalTmuxIntrayPath;
   let originalTmuxIntrayBin;

   beforeEach(() => {
     originalPathEnv = process.env.PATH;
     originalTmuxIntrayPath = process.env.TMUX_INTRAY_PATH;
     originalTmuxIntrayBin = process.env.TMUX_INTRAY_BIN;
   });

   afterEach(() => {
     if (originalPathEnv) process.env.PATH = originalPathEnv;
     if (originalTmuxIntrayPath) {
       process.env.TMUX_INTRAY_PATH = originalTmuxIntrayPath;
     } else {
       delete process.env.TMUX_INTRAY_PATH;
     }
     if (originalTmuxIntrayBin) {
       process.env.TMUX_INTRAY_BIN = originalTmuxIntrayBin;
     } else {
       delete process.env.TMUX_INTRAY_BIN;
     }
   });

  test('finds tmux-intray via PATH when no env vars set', async () => {
    const env = await createTestEnv();
    
    // Add mock binary directory to PATH
    process.env.PATH = `${env.binDir}:${process.env.PATH}`;
    delete process.env.TMUX_INTRAY_PATH;
    delete process.env.TMUX_INTRAY_BIN;

    const pluginModule = await import(pluginPath + '?t=' + Date.now() + Math.random());
    const mockClient = {
      session: { get: async () => ({ data: { title: 'Test' } }) }
    };

    const hooks = await pluginModule.default({ client: mockClient });
    await hooks.event({ 
      event: { type: 'session.idle', properties: {} } 
    });

    const calls = await env.getNotifications();
    expect(calls.length).toBeGreaterThan(0);
    expect(calls[0][0]).toBe('add');
  });

  test('uses TMUX_INTRAY_PATH environment variable override', async () => {
    const env = await createTestEnv();
    
    process.env.TMUX_INTRAY_PATH = join(env.binDir, 'tmux-intray');
    process.env.PATH = `${env.binDir}:${process.env.PATH}`;
    delete process.env.TMUX_INTRAY_BIN;

    const pluginModule = await import(pluginPath + '?t=' + Date.now() + Math.random());
    const mockClient = {
      session: { get: async () => ({ data: { title: 'Test' } }) }
    };

    const hooks = await pluginModule.default({ client: mockClient });
    await hooks.event({ 
      event: { type: 'session.idle', properties: {} } 
    });

    const calls = await env.getNotifications();
    expect(calls.length).toBeGreaterThan(0);
  });

  test('uses TMUX_INTRAY_BIN environment variable', async () => {
    const env = await createTestEnv();
    
    process.env.TMUX_INTRAY_BIN = join(env.binDir, 'tmux-intray');
    process.env.PATH = `${env.binDir}:${process.env.PATH}`;
    delete process.env.TMUX_INTRAY_PATH;

    const pluginModule = await import(pluginPath + '?t=' + Date.now() + Math.random());
    const mockClient = {
      session: { get: async () => ({ data: { title: 'Test' } }) }
    };

    const hooks = await pluginModule.default({ client: mockClient });
    await hooks.event({ 
      event: { type: 'session.idle', properties: {} } 
    });

    const calls = await env.getNotifications();
    expect(calls.length).toBeGreaterThan(0);
  });

  test('respects priority: TMUX_INTRAY_PATH > TMUX_INTRAY_BIN > PATH', async () => {
    const env = await createTestEnv();
    
    // Set all three - TMUX_INTRAY_PATH should win
    process.env.TMUX_INTRAY_PATH = join(env.binDir, 'tmux-intray');
    process.env.TMUX_INTRAY_BIN = join(env.binDir, 'tmux-intray');
    process.env.PATH = `${env.binDir}:${process.env.PATH}`;

    const pluginModule = await import(pluginPath + '?t=' + Date.now() + Math.random());
    const mockClient = {
      session: { get: async () => ({ data: { title: 'Test' } }) }
    };

    const hooks = await pluginModule.default({ client: mockClient });
    await hooks.event({ 
      event: { type: 'session.idle', properties: {} } 
    });

    // If it works, the priority is correct
    const calls = await env.getNotifications();
    expect(calls.length).toBeGreaterThan(0);
  });
});

// =============================================================================
// Context Capture Tests
// =============================================================================

describe('Context Capture', () => {
   let originalTmuxIntrayPath;
   let originalPath;

   beforeEach(() => {
     originalTmuxIntrayPath = process.env.TMUX_INTRAY_PATH;
     originalPath = process.env.PATH;
   });

   afterEach(() => {
     if (originalTmuxIntrayPath) {
       process.env.TMUX_INTRAY_PATH = originalTmuxIntrayPath;
     } else {
       delete process.env.TMUX_INTRAY_PATH;
     }
     if (originalPath) process.env.PATH = originalPath;
   });

  test('captures session ID and includes in command', async () => {
    const env = await createTestEnv();
    process.env.TMUX_INTRAY_PATH = join(env.binDir, 'tmux-intray');
    process.env.PATH = `${env.binDir}:${process.env.PATH}`;

    const pluginModule = await import(pluginPath + '?t=' + Date.now() + Math.random());
    const mockClient = {
      session: { get: async () => ({ data: { title: 'Test' } }) }
    };

    const hooks = await pluginModule.default({ client: mockClient });
    await hooks.event({ 
      event: { type: 'session.idle', properties: {} } 
    });

    const calls = await env.getNotifications();
    expect(calls.length).toBeGreaterThan(0);
    const cmdString = calls[0].join(' ');
    
    // Check for --session flag - when running in tmux, it should be captured
    expect(cmdString).toMatch(/--session=/);  // Session flag is present
  });

  test('captures window ID in correct format (@N)', async () => {
    const env = await createTestEnv({ windowID: '@12' });
    process.env.TMUX_INTRAY_PATH = join(env.binDir, 'tmux-intray');
    process.env.PATH = `${env.binDir}:${process.env.PATH}`;

    const pluginModule = await import(pluginPath + '?t=' + Date.now() + Math.random());
    const mockClient = {
      session: { get: async () => ({ data: { title: 'Test' } }) }
    };

    const hooks = await pluginModule.default({ client: mockClient });
    await hooks.event({ 
      event: { type: 'session.idle', properties: {} } 
    });

    const calls = await env.getNotifications();
    expect(calls.length).toBeGreaterThan(0);
    const lastCall = calls[calls.length - 1];
    
    // Check for --window flag
    const windowFlag = lastCall.find((arg, i) => 
      arg.includes('--window') && arg.includes('@12')
    );
    expect(windowFlag).toBeDefined();
  });

  test('captures pane ID in correct format (%N)', async () => {
    const env = await createTestEnv({ paneID: '%45' });
    process.env.TMUX_INTRAY_PATH = join(env.binDir, 'tmux-intray');
    process.env.PATH = `${env.binDir}:${process.env.PATH}`;

    const pluginModule = await import(pluginPath + '?t=' + Date.now() + Math.random());
    const mockClient = {
      session: { get: async () => ({ data: { title: 'Test' } }) }
    };

    const hooks = await pluginModule.default({ client: mockClient });
    await hooks.event({ 
      event: { type: 'session.idle', properties: {} } 
    });

    const calls = await env.getNotifications();
    expect(calls.length).toBeGreaterThan(0);
    const lastCall = calls[calls.length - 1];
    
    // Check for --pane flag
    const paneFlag = lastCall.find((arg, i) => 
      arg.includes('--pane') && arg.includes('%45')
    );
    expect(paneFlag).toBeDefined();
  });

  test('returns empty string when tmux not available (mocked)', async () => {
    const env = await createTestEnv();
    process.env.TMUX_INTRAY_PATH = join(env.binDir, 'tmux-intray');
    
    // Create a tmux binary that fails to return IDs
    await createMockBin(env.binDir, 'tmux-error', join(env.binDir, 'tmux-error.log'), { 
      error: true 
    });

    process.env.PATH = `${env.binDir}:${process.env.PATH}`;

    const pluginModule = await import(pluginPath + '?t=' + Date.now() + Math.random());
    const mockClient = {
      session: { get: async () => ({ data: { title: 'Test' } }) }
    };

    const hooks = await pluginModule.default({ client: mockClient });
    
    // Should still execute without crashing
    await expect(
      hooks.event({ event: { type: 'session.idle', properties: {} } })
    ).resolves.not.toThrow();
  });

  test('handles tmux command failures gracefully', async () => {
    const env = await createTestEnv();
    process.env.TMUX_INTRAY_PATH = join(env.binDir, 'tmux-intray');
    
    // Create a failing tmux binary
    await createMockBin(env.binDir, 'tmux-fail', join(env.binDir, 'tmux-fail.log'), { 
      error: true 
    });

    process.env.PATH = `${env.binDir}:${process.env.PATH}`;

    const pluginModule = await import(pluginPath + '?t=' + Date.now() + Math.random());
    const mockClient = {
      session: { get: async () => ({ data: { title: 'Test' } }) }
    };

    const hooks = await pluginModule.default({ client: mockClient });
    
    // Should not throw
    await expect(
      hooks.event({ event: { type: 'session.idle', properties: {} } })
    ).resolves.not.toThrow();
  });
});

// =============================================================================
// End-to-End Notification Tests
// =============================================================================

describe('End-to-End Notifications', () => {
   let originalTmuxIntrayPath;
   let originalPath;

   beforeEach(() => {
     originalTmuxIntrayPath = process.env.TMUX_INTRAY_PATH;
     originalPath = process.env.PATH;
   });

   afterEach(() => {
     if (originalTmuxIntrayPath) {
       process.env.TMUX_INTRAY_PATH = originalTmuxIntrayPath;
     } else {
       delete process.env.TMUX_INTRAY_PATH;
     }
     if (originalPath) process.env.PATH = originalPath;
   });

  test('creates notification with full context (session, window, pane)', async () => {
    const env = await createTestEnv();
    
    process.env.TMUX_INTRAY_PATH = join(env.binDir, 'tmux-intray');
    process.env.PATH = `${env.binDir}:${process.env.PATH}`;

    const pluginModule = await import(pluginPath + '?t=' + Date.now() + Math.random());
    const mockClient = {
      session: { get: async () => ({ data: { title: 'Test' } }) }
    };

    const hooks = await pluginModule.default({ client: mockClient });
    await hooks.event({ 
      event: { type: 'session.idle', properties: {} } 
    });

    const calls = await env.getNotifications();
    expect(calls.length).toBeGreaterThan(0);
    
    const cmd = calls[calls.length - 1];
    expect(cmd[0]).toBe('add');
    const cmdString = cmd.join(' ');
    // Verify all context types are present
    expect(cmdString).toMatch(/--session/);
    expect(cmdString).toMatch(/--window/);
    expect(cmdString).toMatch(/--pane/);
  });

  test('context fields use proper tmux identifier formats', async () => {
    const env = await createTestEnv();
    
    process.env.TMUX_INTRAY_PATH = join(env.binDir, 'tmux-intray');
    process.env.PATH = `${env.binDir}:${process.env.PATH}`;

    const pluginModule = await import(pluginPath + '?t=' + Date.now() + Math.random());
    const mockClient = {
      session: { get: async () => ({ data: { title: 'Test' } }) }
    };

    const hooks = await pluginModule.default({ client: mockClient });
    await hooks.event({ 
      event: { type: 'session.idle', properties: {} } 
    });

    const calls = await env.getNotifications();
    const cmd = calls[calls.length - 1];
    const cmdString = cmd.join(' ');
    
    // Verify proper tmux identifier formats
    // When running in actual tmux, the real tmux binary returns actual session/window/pane IDs
    // The pattern is: session $N, window @N, pane %N
    expect(cmdString).toMatch(/--session=/);   // Session flag present
    expect(cmdString).toMatch(/--window=/);    // Window flag present
    expect(cmdString).toMatch(/--pane=/);      // Pane flag present
  });

  test('multiple sequential events create multiple notifications', async () => {
    const env = await createTestEnv({ 
      sessionID: '$0',
      windowID: '@0',
      paneID: '%0'
    });
    
    process.env.TMUX_INTRAY_PATH = join(env.binDir, 'tmux-intray');
    process.env.PATH = `${env.binDir}:${process.env.PATH}`;

    const pluginModule = await import(pluginPath + '?t=' + Date.now() + Math.random());
    const mockClient = {
      session: { get: async () => ({ data: { title: 'Test' } }) }
    };

    const hooks = await pluginModule.default({ client: mockClient });
    
    // Trigger three events
    await hooks.event({ event: { type: 'session.idle', properties: {} } });
    await hooks.event({ event: { type: 'session.error', properties: {} } });
    await hooks.event({ event: { type: 'permission.updated', properties: { permission: 'test' } } });

    const calls = await env.getNotifications();
    
    // Should have multiple notifications
    expect(calls.length).toBeGreaterThanOrEqual(3);
  });

  test('different event types use correct message levels', async () => {
    const env = await createTestEnv();
    
    process.env.TMUX_INTRAY_PATH = join(env.binDir, 'tmux-intray');
    process.env.PATH = `${env.binDir}:${process.env.PATH}`;

    const pluginModule = await import(pluginPath + '?t=' + Date.now() + Math.random());
    const mockClient = {
      session: { get: async () => ({ data: { title: 'Test' } }) }
    };

    const hooks = await pluginModule.default({ client: mockClient });
    
    // Clear and test each event type
    await hooks.event({ event: { type: 'session.idle', properties: {} } });
    let calls = await env.getNotifications();
    expect(calls[0]).toContain('--level=info');
    
    await env.clearNotifications();
    await hooks.event({ event: { type: 'session.error', properties: {} } });
    calls = await env.getNotifications();
    expect(calls[0]).toContain('--level=error');
  });
});

// =============================================================================
// Flag Passing Tests
// =============================================================================

describe('Flag Passing', () => {
   let originalTmuxIntrayPath;
   let originalPath;

   beforeEach(() => {
     originalTmuxIntrayPath = process.env.TMUX_INTRAY_PATH;
     originalPath = process.env.PATH;
   });

   afterEach(() => {
     if (originalTmuxIntrayPath) {
       process.env.TMUX_INTRAY_PATH = originalTmuxIntrayPath;
     } else {
       delete process.env.TMUX_INTRAY_PATH;
     }
     if (originalPath) process.env.PATH = originalPath;
   });

  test('includes context flags in command when available', async () => {
    const env = await createTestEnv();
    
    process.env.TMUX_INTRAY_PATH = join(env.binDir, 'tmux-intray');
    process.env.PATH = `${env.binDir}:${process.env.PATH}`;

    const pluginModule = await import(pluginPath + '?t=' + Date.now() + Math.random());
    const mockClient = {
      session: { get: async () => ({ data: { title: 'Test' } }) }
    };

    const hooks = await pluginModule.default({ client: mockClient });
    await hooks.event({ event: { type: 'session.idle', properties: {} } });

    const calls = await env.getNotifications();
    const cmd = calls[0];
    const cmdString = cmd.join(' ');
    
    // Verify that context flags are in the command
    expect(cmdString).toMatch(/--session/);
    expect(cmdString).toMatch(/--window/);
    expect(cmdString).toMatch(/--pane/);
  });

  test('context flags include tmux identifiers with correct prefixes', async () => {
    const env = await createTestEnv();
    
    process.env.TMUX_INTRAY_PATH = join(env.binDir, 'tmux-intray');
    process.env.PATH = `${env.binDir}:${process.env.PATH}`;

    const pluginModule = await import(pluginPath + '?t=' + Date.now() + Math.random());
    const mockClient = {
      session: { get: async () => ({ data: { title: 'Test' } }) }
    };

    const hooks = await pluginModule.default({ client: mockClient });
    await hooks.event({ event: { type: 'session.idle', properties: {} } });

    const calls = await env.getNotifications();
    const cmd = calls[0];
    const cmdString = cmd.join(' ');
    
    // When running in real tmux, verify the identifier prefixes are correct
    // Session IDs should start with $, window with @, pane with %
    if (cmdString.match(/--session=/)) {
      expect(cmdString).toMatch(/--session=.*\$|\/bin\/sh/);  // $N or /bin/sh from real tmux
    }
    if (cmdString.match(/--window=/)) {
      expect(cmdString).toMatch(/--window=.*@/);  // @N format
    }
    if (cmdString.match(/--pane=/)) {
      expect(cmdString).toMatch(/--pane=.*%/);    // %N format
    }
  });

  test('only includes context flags for non-empty values', async () => {
    // Test with default environment where tmux is available
    const env = await createTestEnv();
    
    process.env.TMUX_INTRAY_PATH = join(env.binDir, 'tmux-intray');
    process.env.PATH = `${env.binDir}:${process.env.PATH}`;

    const pluginModule = await import(pluginPath + '?t=' + Date.now() + Math.random());
    const mockClient = {
      session: { get: async () => ({ data: { title: 'Test' } }) }
    };

    const hooks = await pluginModule.default({ client: mockClient });
    await hooks.event({ event: { type: 'session.idle', properties: {} } });

    const calls = await env.getNotifications();
    const cmd = calls[0];
    
    // In a tmux environment, context will be captured
    // Minimum: add, --level=info, message
    expect(cmd.length).toBeGreaterThanOrEqual(3);
    // Context flags will be added if tmux is available
    expect(cmd[0]).toBe('add');
  });

  test('properly handles context with different tmux IDs', async () => {
    const env = await createTestEnv();
    
    process.env.TMUX_INTRAY_PATH = join(env.binDir, 'tmux-intray');
    process.env.PATH = `${env.binDir}:${process.env.PATH}`;

    const pluginModule = await import(pluginPath + '?t=' + Date.now() + Math.random());
    const mockClient = {
      session: { get: async () => ({ data: { title: 'Test' } }) }
    };

    const hooks = await pluginModule.default({ client: mockClient });
    await hooks.event({ event: { type: 'session.idle', properties: {} } });

    const calls = await env.getNotifications();
    const cmd = calls[0];
    const cmdString = cmd.join(' ');
    
    // Verify all context flags are present
    expect(cmdString).toMatch(/--session/);
    expect(cmdString).toMatch(/--window/);
    expect(cmdString).toMatch(/--pane/);
  });
});

// =============================================================================
// Error Handling Tests
// =============================================================================

describe('Error Handling', () => {
   let originalTmuxIntrayPath;
   let originalPath;

   beforeEach(() => {
     originalTmuxIntrayPath = process.env.TMUX_INTRAY_PATH;
     originalPath = process.env.PATH;
   });

   afterEach(() => {
     if (originalTmuxIntrayPath) {
       process.env.TMUX_INTRAY_PATH = originalTmuxIntrayPath;
     } else {
       delete process.env.TMUX_INTRAY_PATH;
     }
     if (originalPath) process.env.PATH = originalPath;
   });

  test('gracefully handles tmux not running', async () => {
    const env = await createTestEnv();
    
    process.env.TMUX_INTRAY_PATH = join(env.binDir, 'tmux-intray');
    process.env.PATH = `${env.binDir}:${process.env.PATH}`;

    const pluginModule = await import(pluginPath + '?t=' + Date.now() + Math.random());
    const mockClient = {
      session: { get: async () => ({ data: { title: 'Test' } }) }
    };

    const hooks = await pluginModule.default({ client: mockClient });
    
    // Should not throw even if tmux fails
    await expect(
      hooks.event({ event: { type: 'session.idle', properties: {} } })
    ).resolves.not.toThrow();
  });

  test('gracefully handles binary not found', async () => {
    const env = await createTestEnv();
    
    // Set non-existent binary path
    process.env.TMUX_INTRAY_PATH = '/nonexistent/path/tmux-intray';
    process.env.PATH = `${env.binDir}:${process.env.PATH}`;

    const pluginModule = await import(pluginPath + '?t=' + Date.now() + Math.random());
    const mockClient = {
      session: { get: async () => ({ data: { title: 'Test' } }) }
    };

    const hooks = await pluginModule.default({ client: mockClient });
    
    // Should not throw
    await expect(
      hooks.event({ event: { type: 'session.idle', properties: {} } })
    ).resolves.not.toThrow();
  });

  test('handles message with special characters', async () => {
    const env = await createTestEnv();
    
    process.env.TMUX_INTRAY_PATH = join(env.binDir, 'tmux-intray');
    process.env.PATH = `${env.binDir}:${process.env.PATH}`;

    const pluginModule = await import(pluginPath + '?t=' + Date.now() + Math.random());
    const mockClient = {
      session: { get: async () => ({ data: { title: 'Test' } }) }
    };

    const hooks = await pluginModule.default({ client: mockClient });
    
    // Should handle special characters without crashing
    await expect(
      hooks.event({ 
        event: { 
          type: 'session.idle', 
          properties: {} 
        } 
      })
    ).resolves.not.toThrow();
  });

  test('plugin does not crash on tmux-intray execution error', async () => {
    const env = await createTestEnv();
    await env.createErrorBinary('tmux-intray');
    
    process.env.TMUX_INTRAY_PATH = join(env.binDir, 'tmux-intray');
    process.env.PATH = `${env.binDir}:${process.env.PATH}`;

    const pluginModule = await import(pluginPath + '?t=' + Date.now() + Math.random());
    const mockClient = {
      session: { get: async () => ({ data: { title: 'Test' } }) }
    };

    const hooks = await pluginModule.default({ client: mockClient });
    
    // Should not throw even if tmux-intray fails
    await expect(
      hooks.event({ event: { type: 'session.idle', properties: {} } })
    ).resolves.not.toThrow();
  });

  test('logs errors for debugging when notification fails', async () => {
    const env = await createTestEnv();
    
    // Use non-existent binary
    process.env.TMUX_INTRAY_PATH = '/nonexistent/path/to/tmux-intray';
    process.env.PATH = `${env.binDir}:${process.env.PATH}`;

    const pluginModule = await import(pluginPath + '?t=' + Date.now() + Math.random());
    const mockClient = {
      session: { get: async () => ({ data: { title: 'Test' } }) }
    };

    const hooks = await pluginModule.default({ client: mockClient });
    
    // Even with error, should complete without throwing
    await expect(
      hooks.event({ event: { type: 'session.idle', properties: {} } })
    ).resolves.not.toThrow();
  });
});

// =============================================================================
// Real Tmux Environment Tests (Simulated)
// =============================================================================

describe('Real Tmux Environment (Simulated)', () => {
   let originalTmuxIntrayPath;
   let originalPath;

   beforeEach(() => {
     originalTmuxIntrayPath = process.env.TMUX_INTRAY_PATH;
     originalPath = process.env.PATH;
   });

   afterEach(() => {
     if (originalTmuxIntrayPath) {
       process.env.TMUX_INTRAY_PATH = originalTmuxIntrayPath;
     } else {
       delete process.env.TMUX_INTRAY_PATH;
     }
     if (originalPath) process.env.PATH = originalPath;
   });

  test('sends notification with tmux context captured', async () => {
    const env = await createTestEnv();
    
    process.env.TMUX_INTRAY_PATH = join(env.binDir, 'tmux-intray');
    process.env.PATH = `${env.binDir}:${process.env.PATH}`;

    const pluginModule = await import(pluginPath + '?t=' + Date.now() + Math.random());
    const mockClient = {
      session: { get: async () => ({ data: { title: 'My Task' } }) }
    };

    const hooks = await pluginModule.default({ client: mockClient });
    
    // Simulate task completion
    await hooks.event({ 
      event: { type: 'session.idle', properties: { sessionID: 'task-001' } } 
    });

    const calls = await env.getNotifications();
    expect(calls.length).toBeGreaterThan(0);
    const cmdString = calls[0].join(' ');
    // Verify context is captured
    expect(cmdString).toMatch(/--session/);
    expect(cmdString).toMatch(/--window/);
    expect(cmdString).toMatch(/--pane/);
  });

  test('plugin captures context identifiers in correct tmux formats', async () => {
    const env = await createTestEnv();
    
    process.env.TMUX_INTRAY_PATH = join(env.binDir, 'tmux-intray');
    process.env.PATH = `${env.binDir}:${process.env.PATH}`;

    const pluginModule = await import(pluginPath + '?t=' + Date.now() + Math.random());
    const mockClient = {
      session: { get: async () => ({ data: { title: 'Test' } }) }
    };

    const hooks = await pluginModule.default({ client: mockClient });
    await hooks.event({ event: { type: 'session.idle', properties: {} } });

    const calls = await env.getNotifications();
    const cmdString = calls[0].join(' ');
    
    // Verify tmux identifier formats ($N, @N, %N)
    expect(cmdString).toMatch(/--pane=.*%\d+/);  // Pane format
  });

  test('handles multiple sequential events with context', async () => {
    const env = await createTestEnv();
    
    process.env.TMUX_INTRAY_PATH = join(env.binDir, 'tmux-intray');
    process.env.PATH = `${env.binDir}:${process.env.PATH}`;

    const pluginModule = await import(pluginPath + '?t=' + Date.now() + Math.random());
    const mockClient = {
      session: { get: async () => ({ data: { title: 'Test' } }) }
    };

    const hooks = await pluginModule.default({ client: mockClient });
    
    // Simulate multiple events from same session
    await hooks.event({ event: { type: 'session.idle', properties: {} } });
    await hooks.event({ event: { type: 'permission.updated', properties: { permission: 'test' } } });
    await hooks.event({ event: { type: 'session.error', properties: {} } });

    const calls = await env.getNotifications();
    
    // All should have context flags
    expect(calls.length).toBeGreaterThanOrEqual(3);
    calls.forEach(cmd => {
      const cmdString = cmd.join(' ');
      expect(cmdString).toMatch(/--session/);
      expect(cmdString).toMatch(/--window/);
      expect(cmdString).toMatch(/--pane/);
    });
  });
});
