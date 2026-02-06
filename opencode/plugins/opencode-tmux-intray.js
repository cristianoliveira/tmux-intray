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
 * Get current tmux session ID if running inside tmux.
 * Session IDs are in the format $N (e.g., $0, $1, $10).
 * These IDs are needed for the tmux-intray jump command to work correctly.
 * @returns {Promise<string>} Session ID in format $N or empty string if not in tmux
 */
async function getTmuxSessionID() {
  try {
    const { stdout } = await execAsync('tmux display-message -p "#{session_id}"');
    const sessionID = stdout.trim();
    
    // ASSERTION (Power of 10 Rule 5): Validate format is $N or empty
    if (sessionID && !/^\$\d+$/.test(sessionID)) {
      await logError(new Error(`getTmuxSessionID: Invalid format "${sessionID}" (expected $N format)`));
      return '';
    }
    
    await logDebug(`getTmuxSessionID: ${sessionID}`);
    return sessionID;
  } catch (error) {
    // Not in tmux or command failed - this is expected outside tmux
    await logDebug(`getTmuxSessionID failed (expected if not in tmux): ${error.message}`);
    return '';
  }
}

/**
 * Get current tmux window ID if running inside tmux.
 * Window IDs are in the format @N (e.g., @0, @1, @16).
 * These IDs are needed for the tmux-intray jump command to work correctly.
 * @returns {Promise<string>} Window ID in format @N or empty string if not in tmux
 */
async function getTmuxWindowID() {
  try {
    const { stdout } = await execAsync('tmux display-message -p "#{window_id}"');
    const windowID = stdout.trim();
    
    // ASSERTION (Power of 10 Rule 5): Validate format is @N or empty
    if (windowID && !/^@\d+$/.test(windowID)) {
      await logError(new Error(`getTmuxWindowID: Invalid format "${windowID}" (expected @N format)`));
      return '';
    }
    
    await logDebug(`getTmuxWindowID: ${windowID}`);
    return windowID;
  } catch (error) {
    // Not in tmux or command failed - this is expected outside tmux
    await logDebug(`getTmuxWindowID failed (expected if not in tmux): ${error.message}`);
    return '';
  }
}

/**
 * Get current tmux pane ID if running inside tmux.
 * Pane IDs are in the format %N (e.g., %0, %1, %21).
 * These IDs are needed for the tmux-intray jump command to work correctly.
 * When a notification is created with a pane ID, users can jump back to that
 * specific pane using the jump command.
 * @returns {Promise<string>} Pane ID in format %N or empty string if not in tmux
 */
async function getTmuxPaneID() {
  try {
    const { stdout } = await execAsync('tmux display-message -p "#{pane_id}"');
    const paneID = stdout.trim();
    
    // ASSERTION (Power of 10 Rule 5): Validate format is %N or empty
    if (paneID && !/^%\d+$/.test(paneID)) {
      await logError(new Error(`getTmuxPaneID: Invalid format "${paneID}" (expected %N format)`));
      return '';
    }
    
    await logDebug(`getTmuxPaneID: ${paneID}`);
    return paneID;
  } catch (error) {
    // Not in tmux or command failed - this is expected outside tmux
    await logDebug(`getTmuxPaneID failed (expected if not in tmux): ${error.message}`);
    return '';
  }
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
 * Also captures tmux context (session/window/pane IDs) to enable jump functionality.
 * Context capture is best-effort: if any context is unavailable, the command
 * will still work, just without that context field (graceful degradation).
 * @param {string} status - Notification status (success, error, pending)
 * @param {string} message - Notification message
 * @param {string} session - Optional tmux session name (default empty)
 * @param {string} sessionID - Optional cached session ID (from plugin init)
 * @param {string} windowID - Optional cached window ID (from plugin init)
 * @param {string} paneID - Optional cached pane ID (from plugin init)
 * @param {boolean} contextCached - Whether context was already captured at init time
 * @returns {Promise<void>}
 */
async function notify(status, message, session = '', sessionID = '', windowID = '', paneID = '', contextCached = false) {
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
      const tmuxIntrayCmd = await getTmuxIntrayCommand();
      
      // Use provided context IDs if caching is enabled, otherwise capture them now
      let session_id = sessionID;
      let window_id = windowID;
      let pane_id = paneID;
      
      if (!contextCached) {
        // Capture tmux context (session/window/pane IDs).
        // These IDs are required by the jump command to navigate back to the pane
        // where the notification was created. Capture is best-effort: if any fails,
        // we still send the notification, just without that context field.
        session_id = await getTmuxSessionID();
        window_id = await getTmuxWindowID();
        pane_id = await getTmuxPaneID();
      }
      
      // Build the base command
      let addCmd = `${tmuxIntrayCmd} add --level="${level}" "${message}"`;
      
      // ASSERTION (Power of 10 Rule 5): Only add flags if context is available
      // This ensures we don't pass empty or invalid values to tmux-intray
      if (session_id) {
        addCmd += ` --session="${session_id}"`;
      }
      if (window_id) {
        addCmd += ` --window="${window_id}"`;
      }
      if (pane_id) {
        addCmd += ` --pane="${pane_id}"`;
      }
      
      // Log the final command for debugging (without revealing full message in logs)
      await logDebug(`notify: executing ${tmuxIntrayCmd} add with level=${level}, sessionID=${session_id || 'none'}, windowID=${window_id || 'none'}, paneID=${pane_id || 'none'}`);
      
      // Use execAsync to call tmux-intray with level, message, and context
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
   // Cache tmux context at plugin initialization time
   // This is more reliable than detecting per-event since:
   // 1. The context shouldn't change during plugin lifecycle
   // 2. Environment variables are more likely available at init time
   const cachedSession = await getTmuxSession();
   const cachedSessionID = await getTmuxSessionID();
   const cachedWindowID = await getTmuxWindowID();
   const cachedPaneID = await getTmuxPaneID();

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
       // Use cached context, fallback to detection if cache is empty
       const session = cachedSession || await getTmuxSession();
       const sessionID = cachedSessionID;
       const windowID = cachedWindowID;
       const paneID = cachedPaneID;

       // Special handling for session.status - only notify if status is 'pending'
       if (event.type === 'session.status') {
         if (event.properties?.status === 'pending' && isEventEnabled(config, event.type)) {
           const eventConfig = getEventConfig(config, event.type);
           const message = substituteTemplate(eventConfig.message, event);
           await notify(eventConfig.status, message, session, sessionID, windowID, paneID, true);
         }
         return;
       }

        // For all other events, check if enabled and send notification
        if (isEventEnabled(config, event.type)) {
          const eventConfig = getEventConfig(config, event.type);
          const message = substituteTemplate(eventConfig.message, event);
          await notify(eventConfig.status, message, session, sessionID, windowID, paneID, true);
        }
     },
   };
 }

// Named export for OpenCode plugin system
export { opencodeTmuxIntrayPlugin };
// Default export for compatibility
export default opencodeTmuxIntrayPlugin;
