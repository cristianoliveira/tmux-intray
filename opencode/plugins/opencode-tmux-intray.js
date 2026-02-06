#!/usr/bin/env node
/**
 * OpenCode Tmux Intray Plugin
 *
 * OpenCode plugin that hooks session events and calls tmux-intray
 * to display notifications in tmux when OpenCode sessions complete,
 * error, require permissions, or ask questions.
 */

import { exec } from 'node:child_process';
import { promisify } from 'node:util';
import { promises as fs } from 'node:fs';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { loadConfig, isEventEnabled, getEventConfig, substituteTemplate } from './opencode-tmux-intray/config-loader.js';

const execAsync = promisify(exec);

/**
 * Check if plugin is running in test mode.
 * Test mode can be enabled via TEST_MODE environment variable or NODE_ENV=test.
 * @returns {boolean} True if in test mode
 */
function isTestMode() {
  return process.env.TEST_MODE === '1' || process.env.NODE_ENV === 'test';
}

/**
 * Get the tmux-intray command to use.
 * Checks TMUX_INTRAY_PATH environment variable first.
 * Then checks for local binary at ../../tmux-intray relative to this file.
 * Falls back to 'tmux-intray' command in PATH.
 * When TEST_MODE=1 is set, the plugin may log additional debug information.
 * @returns {Promise<string>} Command string
 */
async function getTmuxIntrayCommand() {
  let command;
  // Environment variable takes precedence (used by tests for mocking)
  if (process.env.TMUX_INTRAY_PATH) {
    command = process.env.TMUX_INTRAY_PATH;
  } else {
    // Try local binary relative to this plugin file
    const __filename = fileURLToPath(import.meta.url);
    const __dirname = dirname(__filename);
    const localBinary = join(__dirname, '../../tmux-intray');
    
    try {
      // Check if file exists and is executable
      await fs.access(localBinary, fs.constants.X_OK);
      command = localBinary;
    } catch {
      // File doesn't exist or not executable
      command = 'tmux-intray';
    }
  }

  await logDebug(`Using tmux-intray command: ${command}`);
  return command;
}

/**
 * Log error to file for debugging
 * @param {Error} error - Error object
 * @returns {Promise<void>}
 */
async function logError(error) {
  try {
    const logDir = join(process.cwd(), '.tmp');
    const logFile = join(logDir, 'debug.log');
    await fs.mkdir(logDir, { recursive: true });
    const timestamp = new Date().toISOString();
    const message = `[${timestamp}] [opencode-tmux-intray] Error: ${error.message}\n`;
    await fs.appendFile(logFile, message);
  } catch (logErr) {
    // Ignore errors in logging
  }
}

/**
 * Log debug message to file when in test mode.
 * @param {string} message - Debug message
 * @returns {Promise<void>}
 */
async function logDebug(message) {
  if (!isTestMode()) return;
  try {
    const logDir = join(process.cwd(), '.tmp');
    const logFile = join(logDir, 'debug.log');
    await fs.mkdir(logDir, { recursive: true });
    const timestamp = new Date().toISOString();
    const logMessage = `[${timestamp}] [opencode-tmux-intray] Debug: ${message}\n`;
    await fs.appendFile(logFile, logMessage);
  } catch (logErr) {
    // Ignore errors in logging
  }
}




/**
 * Call tmux-intray with given status and message.
 * The Go CLI automatically detects tmux context (session/window/pane IDs)
 * so the plugin just needs to pass the level and message.
 * @param {string} status - Notification status (success, error, pending)
 * @param {string} message - Notification message
 * @returns {Promise<void>}
 */
async function notify(status, message) {
    // Map status to tmux-intray level
    const levelMap = {
      'error': 'error',
      'pending': 'warning',
      'success': 'info'
    };
    const level = levelMap[status] || 'info';

    try {
      const tmuxIntrayCmd = await getTmuxIntrayCommand();
      
      // Build the command - let Go CLI handle context detection
      const addCmd = `${tmuxIntrayCmd} add --level="${level}" "${message}"`;
      
      await logDebug(`notify: executing ${tmuxIntrayCmd} add with level=${level}`);
      
      // Call tmux-intray with level and message only
      await execAsync(addCmd);
    } catch (error) {
      // Log error but don't crash the plugin
      console.error(`[opencode-tmux-intray] Failed to send notification: ${error.message}`);
      await logError(error);
    }
}

/**
 * OpenCode plugin function
 * @param {Object} context - OpenCode plugin context
 * @param {Object} context.client - OpenCode client SDK
 * @returns {Promise<Object>} Plugin hooks
 */
async function opencodeTmuxIntrayPlugin({ client }) {
    // Load configuration once at initialization
    const config = await loadConfig();

   return {
      /**
       * Event handler for OpenCode events
       * Handles events configured in opencode-config.json
       * Default events: session.idle, session.error, session.status, permission.updated
       * @param {Object} params - Event parameters
       * @param {Object} params.event - Event object
       * @returns {Promise<void>}
       */
      event: async ({ event }) => {
        // Special handling for session.status - only notify if status is 'pending'
        if (event.type === 'session.status') {
          if (event.properties?.status === 'pending' && isEventEnabled(config, event.type)) {
            const eventConfig = getEventConfig(config, event.type);
            const message = substituteTemplate(eventConfig.message, event);
            await notify(eventConfig.status, message);
          }
          return;
        }

        // For all other events, check if enabled and send notification
        if (isEventEnabled(config, event.type)) {
          const eventConfig = getEventConfig(config, event.type);
          const message = substituteTemplate(eventConfig.message, event);
          await notify(eventConfig.status, message);
        }
      },
   };
 }

// Named export for OpenCode plugin system
export { opencodeTmuxIntrayPlugin };
// Default export for compatibility
export default opencodeTmuxIntrayPlugin;
