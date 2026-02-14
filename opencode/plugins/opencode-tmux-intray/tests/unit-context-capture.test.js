/**
 * Unit Tests for Context Capture Functions
 *
 * Tests the getTmuxSessionID, getTmuxWindowID, and getTmuxPaneID functions
 * in isolation with JavaScript-level mocking of execFileAsync.
 *
 * This test suite validates:
 * 1. Successful capture returns correct format (session_id, @N, %N)
 * 2. Error handling returns empty string
 * 3. execFileAsync is called with correct arguments
 * 4. Return values are properly trimmed
 * 5. Edge cases like whitespace and empty output
 *
 * These unit tests complement integration tests by catching errors
 * at the JavaScript level (e.g., undefined variables) that can be
 * silently masked by graceful error handling.
 */

import { describe, test, expect, beforeEach, vi } from 'vitest';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const pluginPath = join(__dirname, '../../opencode-tmux-intray.js');

// We'll mock the child_process module before importing the plugin
vi.mock('node:child_process', () => {
  return {
    execFile: vi.fn(),
  };
});

// Import after mocking
import { execFile } from 'node:child_process';
import { promisify } from 'node:util';
import { getTmuxSessionID, getTmuxWindowID, getTmuxPaneID } from '../../opencode-tmux-intray.js';

describe('Context Capture Functions - Unit Tests', () => {
  let execFileAsync;

  beforeEach(() => {
    // Clear all mocks before each test
    vi.clearAllMocks();
    
    // Create a new promisified version for each test
    execFileAsync = promisify(execFile);
  });

  describe('getTmuxSessionID', () => {
    test('captures session ID', async () => {
      // Mock execFileAsync to return session ID
      vi.mocked(execFile).mockImplementation((cmd, args, ...rest) => {
        const callback = rest[rest.length - 1]; // Last arg is always callback
        setTimeout(() => callback(null, { stdout: 'myproject\n' }), 0);
      });

      const result = await getTmuxSessionID();

      expect(result).toBe('myproject');
      expect(execFile).toHaveBeenCalledWith(
        'tmux',
        ['display-message', '-p', '#{session_id}'],
        expect.objectContaining({ env: expect.any(Object) }),
        expect.any(Function)
      );
    });

    test('handles whitespace in session ID output', async () => {
      // Mock to return whitespace-padded output
      vi.mocked(execFile).mockImplementation((cmd, args, ...rest) => {
        const callback = rest[rest.length - 1]; // Last arg is always callback
        setTimeout(() => callback(null, { stdout: '  mysession  \n' }), 0);
      });

      const result = await getTmuxSessionID();

      // Should be trimmed
      expect(result).toBe('mysession');
    });

    test('handles newlines in output', async () => {
      // Mock to return output with multiple newlines
      vi.mocked(execFile).mockImplementation((cmd, args, ...rest) => {
        const callback = rest[rest.length - 1]; // Last arg is always callback
        setTimeout(() => callback(null, { stdout: '\n\nwork\n\n' }), 0);
      });

      const result = await getTmuxSessionID();

      expect(result).toBe('work');
    });

    test('returns empty string on error', async () => {
      // Mock execFileAsync to throw an error
      vi.mocked(execFile).mockImplementation((cmd, args, ...rest) => {
        const callback = rest[rest.length - 1]; // Last arg is always callback
        setTimeout(() => callback(new Error('tmux not running')), 0);
      });

      const result = await getTmuxSessionID();

      expect(result).toBe('');
    });

    test('handles execFileAsync throwing non-standard errors', async () => {
      // Mock to throw a different error
      vi.mocked(execFile).mockImplementation((cmd, args, ...rest) => {
        const callback = rest[rest.length - 1]; // Last arg is always callback
        setTimeout(() => callback(new Error('ENOENT: command not found')), 0);
      });

      const result = await getTmuxSessionID();

      expect(result).toBe('');
    });

    test('does not crash on null error', async () => {
      // Some callback patterns pass error: null for success
      vi.mocked(execFile).mockImplementation((cmd, args, ...rest) => {
        const callback = rest[rest.length - 1]; // Last arg is always callback
        setTimeout(() => callback(null, { stdout: 'dev\n' }), 0);
      });

      const result = await getTmuxSessionID();

      expect(result).toBe('dev');
    });
  });

  describe('getTmuxWindowID', () => {
    test('captures window ID in @N format', async () => {
      // Mock execFileAsync to return window ID
      vi.mocked(execFile).mockImplementation((cmd, args, ...rest) => {
        const callback = rest[rest.length - 1]; // Last arg is always callback
        setTimeout(() => callback(null, { stdout: '@42\n' }), 0);
      });

      const result = await getTmuxWindowID();

      expect(result).toBe('@42');
      expect(execFile).toHaveBeenCalledWith(
        'tmux',
        ['display-message', '-p', '#{window_id}'],
        expect.objectContaining({ env: expect.any(Object) }),
        expect.any(Function)
      );
    });

    test('handles whitespace in window ID output', async () => {
      // Mock to return whitespace-padded output
      vi.mocked(execFile).mockImplementation((cmd, args, ...rest) => {
        const callback = rest[rest.length - 1]; // Last arg is always callback
        setTimeout(() => callback(null, { stdout: '\n  @16\t\n' }), 0);
      });

      const result = await getTmuxWindowID();

      expect(result).toBe('@16');
    });

    test('handles large window IDs', async () => {
      // Window IDs can be large numbers
      vi.mocked(execFile).mockImplementation((cmd, args, ...rest) => {
        const callback = rest[rest.length - 1]; // Last arg is always callback
        setTimeout(() => callback(null, { stdout: '@999\n' }), 0);
      });

      const result = await getTmuxWindowID();

      expect(result).toBe('@999');
    });

    test('returns empty string on error', async () => {
      // Mock execFileAsync to throw an error
      vi.mocked(execFile).mockImplementation((cmd, args, ...rest) => {
        const callback = rest[rest.length - 1]; // Last arg is always callback
        setTimeout(() => callback(new Error('command failed')), 0);
      });

      const result = await getTmuxWindowID();

      expect(result).toBe('');
    });

    test('handles ENOENT error gracefully', async () => {
      // Common error when binary not found
      const error = new Error('ENOENT: no such file or directory');
      vi.mocked(execFile).mockImplementation((cmd, args, ...rest) => {
        const callback = rest[rest.length - 1]; // Last arg is always callback
        setTimeout(() => callback(error), 0);
      });

      const result = await getTmuxWindowID();

      expect(result).toBe('');
    });
  });

  describe('getTmuxPaneID', () => {
    test('captures pane ID in %N format', async () => {
      // Mock execFileAsync to return pane ID
      vi.mocked(execFile).mockImplementation((cmd, args, ...rest) => {
        const callback = rest[rest.length - 1]; // Last arg is always callback
        setTimeout(() => callback(null, { stdout: '%21\n' }), 0);
      });

      const result = await getTmuxPaneID();

      expect(result).toBe('%21');
      expect(execFile).toHaveBeenCalledWith(
        'tmux',
        ['display-message', '-p', '#{pane_id}'],
        expect.objectContaining({ env: expect.any(Object) }),
        expect.any(Function)
      );
    });

    test('handles whitespace in pane ID output', async () => {
      // Mock to return whitespace-padded output
      vi.mocked(execFile).mockImplementation((cmd, args, ...rest) => {
        const callback = rest[rest.length - 1]; // Last arg is always callback
        setTimeout(() => callback(null, { stdout: '  %99  \n\n' }), 0);
      });

      const result = await getTmuxPaneID();

      expect(result).toBe('%99');
    });

    test('handles zero pane ID', async () => {
      // Pane ID 0 is valid
      vi.mocked(execFile).mockImplementation((cmd, args, ...rest) => {
        const callback = rest[rest.length - 1]; // Last arg is always callback
        setTimeout(() => callback(null, { stdout: '%0\n' }), 0);
      });

      const result = await getTmuxPaneID();

      expect(result).toBe('%0');
    });

    test('returns empty string on error', async () => {
      // Mock execFileAsync to throw an error
      vi.mocked(execFile).mockImplementation((cmd, args, ...rest) => {
        const callback = rest[rest.length - 1]; // Last arg is always callback
        setTimeout(() => callback(new Error('tmux not available')), 0);
      });

      const result = await getTmuxPaneID();

      expect(result).toBe('');
    });

    test('handles timeout-like errors', async () => {
      // Simulate timeout or other async error
      vi.mocked(execFile).mockImplementation((cmd, args, ...rest) => {
        const callback = rest[rest.length - 1]; // Last arg is always callback
        setTimeout(() => callback(new Error('ETIMEDOUT')), 0);
      });

      const result = await getTmuxPaneID();

      expect(result).toBe('');
    });
  });

  describe('execFileAsync usage verification', () => {
    test('getTmuxSessionID calls execFileAsync with correct arguments', async () => {
      vi.mocked(execFile).mockImplementation((cmd, args, ...rest) => {
        const callback = rest[rest.length - 1]; // Last arg is always callback
        setTimeout(() => callback(null, { stdout: 'myproject\n' }), 0);
      });

      await getTmuxSessionID();

      expect(execFile).toHaveBeenCalledTimes(1);
      expect(execFile).toHaveBeenCalledWith(
        'tmux',
        ['display-message', '-p', '#{session_id}'],
        expect.objectContaining({ env: expect.any(Object) }),
        expect.any(Function)
      );
    });

    test('getTmuxWindowID calls execFileAsync with correct arguments', async () => {
      vi.mocked(execFile).mockImplementation((cmd, args, ...rest) => {
        const callback = rest[rest.length - 1]; // Last arg is always callback
        setTimeout(() => callback(null, { stdout: '@42\n' }), 0);
      });

      await getTmuxWindowID();

      expect(execFile).toHaveBeenCalledTimes(1);
      expect(execFile).toHaveBeenCalledWith(
        'tmux',
        ['display-message', '-p', '#{window_id}'],
        expect.objectContaining({ env: expect.any(Object) }),
        expect.any(Function)
      );
    });

    test('getTmuxPaneID calls execFileAsync with correct arguments', async () => {
      vi.mocked(execFile).mockImplementation((cmd, args, ...rest) => {
        const callback = rest[rest.length - 1]; // Last arg is always callback
        setTimeout(() => callback(null, { stdout: '%21\n' }), 0);
      });

      await getTmuxPaneID();

      expect(execFile).toHaveBeenCalledTimes(1);
      expect(execFile).toHaveBeenCalledWith(
        'tmux',
        ['display-message', '-p', '#{pane_id}'],
        expect.objectContaining({ env: expect.any(Object) }),
        expect.any(Function)
      );
    });
  });

  describe('edge cases and robustness', () => {
    test('handles empty stdout gracefully', async () => {
      // Empty output should be returned as empty string
      vi.mocked(execFile).mockImplementation((cmd, args, ...rest) => {
        const callback = rest[rest.length - 1]; // Last arg is always callback
        setTimeout(() => callback(null, { stdout: '' }), 0);
      });

      const result = await getTmuxSessionID();

      expect(result).toBe('');
    });

    test('handles only whitespace in stdout', async () => {
      // Only spaces/newlines should be trimmed to empty string
      vi.mocked(execFile).mockImplementation((cmd, args, ...rest) => {
        const callback = rest[rest.length - 1]; // Last arg is always callback
        setTimeout(() => callback(null, { stdout: '   \n  \t' }), 0);
      });

      const result = await getTmuxWindowID();

      expect(result).toBe('');
    });

    test('handles very long session IDs', async () => {
      // Some session IDs might be very long
      const longName = 'session-' + 'x'.repeat(100);
      vi.mocked(execFile).mockImplementation((cmd, args, ...rest) => {
        const callback = rest[rest.length - 1]; // Last arg is always callback
        setTimeout(() => callback(null, { stdout: longName + '\n' }), 0);
      });

      const result = await getTmuxSessionID();

      expect(result).toBe(longName);
    });

    test('multiple calls maintain isolation', async () => {
      // Each call should be independent
      const execFileAsyncLocal = promisify(execFile);

      // First call
      vi.mocked(execFile).mockImplementation((cmd, args, ...rest) => {
        const callback = rest[rest.length - 1]; // Last arg is always callback
        setTimeout(() => callback(null, { stdout: 'session1\n' }), 0);
      });
      const result1 = await getTmuxSessionID();

      // Second call
      vi.mocked(execFile).mockImplementation((cmd, args, ...rest) => {
        const callback = rest[rest.length - 1]; // Last arg is always callback
        setTimeout(() => callback(null, { stdout: 'session2\n' }), 0);
      });
      const result2 = await getTmuxSessionID();

      expect(result1).toBe('session1');
      expect(result2).toBe('session2');
      expect(execFile).toHaveBeenCalledTimes(2);
    });
  });

  describe('error message handling', () => {
    test('logs error messages from execFileAsync', async () => {
      // When execFileAsync throws with a message
      const errorMsg = 'tmux not running in current session';
      vi.mocked(execFile).mockImplementation((cmd, args, ...rest) => {
        const callback = rest[rest.length - 1]; // Last arg is always callback
        setTimeout(() => callback(new Error(errorMsg)), 0);
      });

      const result = await getTmuxSessionID();

      expect(result).toBe('');
      // The error message should be included in logs (verify by checking no crash)
    });

    test('handles error objects without message property', async () => {
      // Some errors might not have proper message
      vi.mocked(execFile).mockImplementation((cmd, args, ...rest) => {
        const callback = rest[rest.length - 1]; // Last arg is always callback
        const error = new Error();
        error.message = undefined;
        setTimeout(() => callback(error), 0);
      });

      // Should not crash even with missing error message
      const result = await getTmuxWindowID();

      expect(result).toBe('');
    });

    test('never rethrows errors to caller', async () => {
      // Critical: functions must never throw
      vi.mocked(execFile).mockImplementation((cmd, args, ...rest) => {
        const callback = rest[rest.length - 1]; // Last arg is always callback
        setTimeout(() => callback(new Error('Any error')), 0);
      });

      // Should not throw - must return empty string
      await expect(getTmuxPaneID()).resolves.toBe('');
    });
  });

  describe('integration patterns', () => {
    test('can be awaited successfully', async () => {
      vi.mocked(execFile).mockImplementation((cmd, args, ...rest) => {
        const callback = rest[rest.length - 1]; // Last arg is always callback
        setTimeout(() => callback(null, { stdout: 'myproject\n' }), 0);
      });

      // Should be awaitable and return a string
      const result = await getTmuxSessionID();
      expect(typeof result).toBe('string');
    });

    test('works in parallel execution', async () => {
      // All three functions can be called in parallel
      vi.mocked(execFile)
        .mockImplementation((cmd, args, ...rest) => {
          const callback = rest[rest.length - 1]; // Last arg is always callback
          if (args[2]?.includes('session_id')) {
            setTimeout(() => callback(null, { stdout: 'myproject\n' }), 0);
          } else if (args[2]?.includes('window_id')) {
            setTimeout(() => callback(null, { stdout: '@2\n' }), 0);
          } else {
            setTimeout(() => callback(null, { stdout: '%3\n' }), 0);
          }
        });

      const [sessionName, windowID, paneID] = await Promise.all([
        getTmuxSessionID(),
        getTmuxWindowID(),
        getTmuxPaneID(),
      ]);

      expect(sessionName).toBe('myproject');
      expect(windowID).toBe('@2');
      expect(paneID).toBe('%3');
      expect(execFile).toHaveBeenCalledTimes(3);
    });
  });
});
