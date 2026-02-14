#!/usr/bin/env node
/**
 * OpenCode Tmux Intray Plugin
 *Error
 * OpenCode plugin that hooks session events and calls tmux-intray
 * to display notifications in tmux when OpenCode sessions complete,
 * error, require permissions, or ask questions.
 */

import { execFile } from 'node:child_process';
import { promisify } from 'node:util';
import { promises as fs } from 'node:fs';
import { join } from 'node:path';

const execFileAsync = promisify(execFile);

/**
 * Log message to file with timestamp and component name.
 * @param {string} level - Log level (Debug, Info, Error, Success)
 * @param {string} functionName - Name of the calling function
 * @param {string} message - Log message
 * @returns {Promise<void>}
 */
async function log(level, functionName, message) {
    try {
        const logFile = '/tmp/opencode-tmux-intray.log';
        const timestamp = new Date().toISOString();
        const logMessage = `[${timestamp}] [opencode-plugin] ${functionName}: ${message}\n`;
        await fs.appendFile(logFile, logMessage);
    } catch (logErr) {
        // Ignore errors in logging - don't crash the plugin
    }
}

/**
 * Log error to file with full error details.
 * @param {string} functionName - Name of the calling function
 * @param {Error} error - Error object
 * @returns {Promise<void>}
 */
async function logError(functionName, error) {
    try {
        const logFile = '/tmp/opencode-tmux-intray.log';
        const timestamp = new Date().toISOString();
        const errorDetails = error.stack || error.message;
        const message = `[${timestamp}] [opencode-plugin] ${functionName}: ERROR - ${errorDetails}\n`;
        await fs.appendFile(logFile, message);
    } catch (logErr) {
        // Ignore errors in logging
    }
}

/**
 * Log debug message to file.
 * @param {string} functionName - Name of the calling function
 * @param {string} message - Debug message
 * @returns {Promise<void>}
 */
async function logDebug(functionName, message) {
    await log('Debug', functionName, message);
}

/**
 * Get the tmux-intray command to use.
 *
 * Priority order for binary lookup:
 * 1. TMUX_INTRAY_PATH environment variable (explicit override for tests/deployment)
 * 2. TMUX_INTRAY_BIN environment variable (binary location hint for CI/deployment)
 * 3. 'tmux-intray' in PATH (standard for installed binaries via `go install`)
 *
 * @returns {Promise<string>} Command string
 */
async function getTmuxIntrayCommand() {
    // Priority 1: Explicit override for tests/deployment
    if (process.env.TMUX_INTRAY_PATH) {
        await logDebug('getTmuxIntrayCommand', `using TMUX_INTRAY_PATH=${process.env.TMUX_INTRAY_PATH}`);
        return process.env.TMUX_INTRAY_PATH;
    }

    // Priority 2: Binary location hint for CI/deployment
    if (process.env.TMUX_INTRAY_BIN) {
        await logDebug('getTmuxIntrayCommand', `using TMUX_INTRAY_BIN=${process.env.TMUX_INTRAY_BIN}`);
        return process.env.TMUX_INTRAY_BIN;
    }

    // Priority 3: Use PATH (standard for installed binaries)
    await logDebug('getTmuxIntrayCommand', 'using tmux-intray from PATH');
    return 'tmux-intray';
}

/**
 * Check if plugin is running in test mode.
 * Test mode can be enabled via TEST_MODE=1 environment variable.
 * When in test mode, tmux context capture is skipped.
 * @returns {boolean} True if in test mode
 */
function isTestMode() {
    return process.env.TEST_MODE === '1';
}

/**
 * Get the current tmux session ID.
 * Returns the session ID in the format $N (e.g., $0, $1, $2).
 * If not in a tmux session or tmux command fails, returns empty string.
 * @returns {Promise<string>} Session ID in $N format, or empty string if unavailable
 */
async function getTmuxSessionID() {
    // In test mode, skip context capture to allow tests to control behavior
    if (isTestMode()) {
        await logDebug('getTmuxSessionID', 'test mode - skipping context capture');
        return '';
    }

    try {
        const { stdout } = await execFileAsync('tmux', ['display-message', '-p', '#{session_id}'], {
            env: process.env
        });
        const sessionID = stdout.trim();
        await logDebug('getTmuxSessionID', `captured session=${sessionID}`);
        return sessionID;
    } catch (error) {
        await logDebug('getTmuxSessionID', `failed to get session ID: ${error.message}`);
        return '';  // Not in tmux or command failed, return empty
    }
}

/**
 * Get the current tmux window ID.
 * Returns the window ID in the format @N (e.g., @0, @1, @16).
 * If not in a tmux session or tmux command fails, returns empty string.
 * @returns {Promise<string>} Window ID in @N format, or empty string if unavailable
 */
async function getTmuxWindowID() {
    // In test mode, skip context capture to allow tests to control behavior
    if (isTestMode()) {
        await logDebug('getTmuxWindowID', 'test mode - skipping context capture');
        return '';
    }

    try {
        const { stdout } = await execFileAsync('tmux', ['display-message', '-p', '#{window_id}'], {
            env: process.env
        });
        const windowID = stdout.trim();
        await logDebug('getTmuxWindowID', `captured window=${windowID}`);
        return windowID;
    } catch (error) {
        await logDebug('getTmuxWindowID', `failed to get window ID: ${error.message}`);
        return '';  // Not in tmux or command failed, return empty
    }
}

/**
 * Get the current tmux pane ID.
 * Returns the pane ID in the format %N (e.g., %0, %1, %21).
 * If not in a tmux session or tmux command fails, returns empty string.
 * @returns {Promise<string>} Pane ID in %N format, or empty string if unavailable
 */
async function getTmuxPaneID() {
    // In test mode, skip context capture to allow tests to control behavior
    if (isTestMode()) {
        await logDebug('getTmuxPaneID', 'test mode - skipping context capture');
        return '';
    }

    try {
        const { stdout } = await execFileAsync('tmux', ['display-message', '-p', '#{pane_id}'], {
            env: process.env
        });
        const paneID = stdout.trim();
        await logDebug('getTmuxPaneID', `captured pane=${paneID}`);
        return paneID;
    } catch (error) {
        await logDebug('getTmuxPaneID', `failed to get pane ID: ${error.message}`);
        return '';  // Not in tmux or command failed, return empty
    }
}

/**
 * Call tmux-intray with given status and message.
 * Captures tmux context (session/window/pane IDs) and passes them as flags
 * to the CLI. The CLI uses these values as primary context, with auto-detection
 * as fallback when flags are not provided.
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

        // Capture context from tmux
        const sessionID = await getTmuxSessionID();
        const windowID = await getTmuxWindowID();
        const paneID = await getTmuxPaneID();

        // Normalize pane ID: ensure it starts with '%' if it's a number
        let normalizedPaneID = paneID;
        if (normalizedPaneID && !normalizedPaneID.startsWith('%')) {
            normalizedPaneID = '%' + normalizedPaneID;
        }

        // Build command with context flags (if available)
        // Use array format instead of string to avoid shell interpolation
        // IMPORTANT: Do NOT quote values - execFile with array args doesn't use shell parsing
        const args = ['add', `--level=${level}`];
        if (sessionID) {
            args.push(`--session=${sessionID}`);
        }
        if (windowID) {
            args.push(`--window=${windowID}`);
        }
        if (normalizedPaneID) {
            args.push(`--pane=${normalizedPaneID}`);
        }
        args.push('--');
        args.push(message);

        // Create a version of args with quoted values for logging (shell-safe)
        const loggedArgs = args.map(arg => {
            if (arg.startsWith('--session=') || arg.startsWith('--window=') || arg.startsWith('--pane=')) {
                const eqIndex = arg.indexOf('=');
                const flag = arg.substring(0, eqIndex);
                const value = arg.substring(eqIndex + 1);
                // If value is already quoted, don't add extra quotes
                if ((value.startsWith("'") && value.endsWith("'")) || (value.startsWith('"') && value.endsWith('"'))) {
                    return arg;
                }
                return `${flag}='${value}'`;
            }
            return arg;
        });

        const commandStr = `${tmuxIntrayCmd} ${loggedArgs.join(' ')}`;
        await logDebug('notify', `executing command: ${commandStr}`);

        // Call tmux-intray with context flags using execFile (no shell)
        await execFileAsync(tmuxIntrayCmd, args, {
            env: process.env
        });

        await logDebug('notify', `success - notification created with message: "${message}"`);
    } catch (error) {
        // Log error but don't crash the plugin
        let errorMsg = `Failed to send notification: ${error.message}`;
        if (error.code === 'ENOENT') {
            errorMsg = `tmux-intray command not found. Please ensure tmux-intray is installed and in PATH. Original error: ${error.message}`;
        } else if (error.code) {
            errorMsg += ` (exit code: ${error.code})`;
        }
        await logError('notify', error);
    }
}

/**
 * Get message and status for an event type.
 * Simple hardcoded mapping - no config file needed.
 * @param {string} eventType - Event type (e.g., 'session.idle')
 * @returns {object} Event configuration with message and status
 */
function getEventConfig(eventType) {
    const eventConfigs = {
        'session.idle': {
            message: 'Task completed',
            status: 'success'
        },
        'session.error': {
            message: 'Session error',
            status: 'error'
        },
        'session.status': {
            message: 'Session status: pending',
            status: 'pending'
        },
        'permission.updated': {
            message: 'Permission needed',
            status: 'pending'
        }
    };

    return eventConfigs[eventType] || {
        message: `Event: ${eventType}`,
        status: 'info'
    };
}

/**
 * OpenCode plugin function
 * @param {Object} context - OpenCode plugin context
 * @param {Object} context.client - OpenCode client SDK
 * @returns {Promise<Object>} Plugin hooks
 */
async function opencodeTmuxIntrayPlugin({ client }) {
    // Log plugin initialization
    await logDebug('opencodeTmuxIntrayPlugin', 'plugin initializing');

    return {
        /**
         * Event handler for OpenCode events
         * @param {Object} params - Event parameters
         * @param {Object} params.event - Event object
         * @returns {Promise<void>}
         */
        event: async ({ event }) => {
            await logDebug('event', `received event type: ${event.type}`);

            // Special handling for session.status - only notify if status is 'pending'
            if (event.type === 'session.status') {
                if (event.properties?.status === 'pending') {
                    const config = getEventConfig(event.type);
                    await notify(config.status, config.message);
                }
                return;
            }

            // Handle known event types
            const knownEvents = ['session.idle', 'session.error', 'permission.updated'];
            if (knownEvents.includes(event.type)) {
                const config = getEventConfig(event.type);
                await notify(config.status, config.message);
            } else {
                // Unknown events - log but don't notify
                await logDebug('event', `unknown event type: ${event.type}, ignoring`);
            }
        },
    };
}

// Named export for OpenCode plugin system
export { opencodeTmuxIntrayPlugin };
// Default export for compatibility
export default opencodeTmuxIntrayPlugin;
// Export context capture functions for unit testing
export { getTmuxSessionID, getTmuxWindowID, getTmuxPaneID };
