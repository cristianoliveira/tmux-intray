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
import { join } from 'node:path';
import { loadConfig, isEventEnabled, getEventConfig, substituteTemplate } from './opencode-tmux-intray/config-loader.js';

const execAsync = promisify(exec);

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
 * Get current tmux session name if running inside tmux
 * @returns {Promise<string>} Session name or empty string
 */
async function getTmuxSession() {
  // First try using TMUX env var to check if we're in tmux
  if (process.env.TMUX) {
    try {
      const { stdout } = await execAsync('tmux display-message -p "#S"');
      return stdout.trim();
    } catch (error) {
      // tmux command failed
      await logError(new Error(`getTmuxSession failed with TMUX set: ${error.message}`));
    }
  }

  // Fallback: Try running tmux command anyway (it might work even without TMUX env)
  // This handles cases where the env var wasn't inherited but tmux is available
  try {
    const { stdout } = await execAsync('tmux display-message -p "#S"');
    const session = stdout.trim();
    if (session) {
      return session;
    }
  } catch (error) {
    // tmux not available or not in a session - this is expected outside tmux
  }

  return '';
}

/**
 * Log notification to global log file for visibility
 * @param {string} status - Notification status
 * @param {string} message - Notification message
 * @param {string} session - Tmux session name
 * @returns {Promise<void>}
 */
async function logNotification(status, message, session) {
  try {
    const logFile = '/tmp/opencode-tmux-intray.log';
    const timestamp = new Date().toISOString();
    const logMessage = `[${timestamp}] [opencode] NOTIFY: status="${status}" message="${message}" session="${session}"\n`;
    await fs.appendFile(logFile, logMessage);
  } catch (logErr) {
    // Ignore errors in logging
  }
}

/**
 * Call tmux-intray with given status and message
 * @param {string} status - Notification status (success, error, pending)
 * @param {string} message - Notification message
 * @param {string} session - Optional tmux session name (default empty)
 * @returns {Promise<void>}
 */
async function notify(status, message, session = '') {
  // Log notification for visibility/debugging
  await logNotification(status, message, session);

   // Map status to tmux-intray level
  const levelMap = {
    'error': 'error',
    'pending': 'warning',
    'success': 'info'
  };
  const level = levelMap[status] || 'info';

  try {
    // Use execAsync to call tmux-intray with level and message
    await execAsync(`tmux-intray add --level="${level}" "${message}"`);
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
  // Cache tmux session at plugin initialization time
  // This is more reliable than detecting per-event since:
  // 1. The session shouldn't change during plugin lifecycle
  // 2. Environment variables are more likely available at init time
  const cachedSession = await getTmuxSession();

  // Load configuration once at initialization
  const config = await loadConfig();

  // Log for debugging
  if (!cachedSession) {
    await logError(new Error(`Warning: No tmux session detected at init. TMUX env: ${process.env.TMUX || 'not set'}`));
  }

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
      // Use cached session, fallback to detection if cache is empty
      const session = cachedSession || await getTmuxSession();

      // Special handling for session.status - only notify if status is 'pending'
      if (event.type === 'session.status') {
        if (event.properties?.status === 'pending' && isEventEnabled(config, event.type)) {
          const eventConfig = getEventConfig(config, event.type);
          const message = substituteTemplate(eventConfig.message, event);
          await notify(eventConfig.status, message, session);
        }
        return;
      }

       // For all other events, check if enabled and send notification
       if (isEventEnabled(config, event.type)) {
         const eventConfig = getEventConfig(config, event.type);
         const message = substituteTemplate(eventConfig.message, event);
         await notify(eventConfig.status, message, session);
       }
    },
  };
}

// Named export for OpenCode plugin system
export { opencodeTmuxIntrayPlugin };
// Default export for compatibility
export default opencodeTmuxIntrayPlugin;
