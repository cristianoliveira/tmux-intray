#!/usr/bin/env node
/**
 * Configuration loader for OpenCode Tmux Intray Plugin
 *
 * Loads configuration from ~/.config/opencode-tmux-intray/opencode-config.json
 * and provides utilities for working with the configuration.
 */

import { readFile } from 'node:fs/promises';
import { join, dirname } from 'node:path';
import { homedir } from 'node:os';

/**
 * Default configuration used when no config file exists or config is invalid
 */
export const DEFAULT_CONFIG = {
  enabled: true,
  agentName: 'opencode',
  notifications: {
    'session.idle': {
      enabled: true,
      message: 'Task completed',
      status: 'success'
    },
    'session.error': {
      enabled: true,
      message: 'Session error',
      status: 'error'
    },
    'session.status': {
      enabled: false,
      message: 'Session status: {properties.status}',
      status: 'pending'
    },
    'session.created': {
      enabled: false,
      message: 'New session created',
      status: 'success'
    },
    'session.updated': {
      enabled: false,
      message: 'Session updated',
      status: 'success'
    },
    'permission.updated': {
      enabled: true,
      message: 'Permission needed',
      status: 'pending'
    },
    'permission.replied': {
      enabled: false,
      message: 'Permission replied',
      status: 'success'
    },
    'question.asked': {
      enabled: false,
      message: 'Question asked: {question}',
      status: 'pending'
    },
    'permission.asked': {
      enabled: false,
      message: 'Permission asked: {permission}',
      status: 'pending'
    }
  },
  sound: {
    enabled: true,
    file: '/System/Library/Sounds/Glass.aiff'
  },
  tts: {
    enabled: false,
    message: 'Agent {agentName} completed with status {status}',
    voice: 'Alex'
  }
};

/**
 * Load configuration from file, falling back to defaults if file doesn't exist or is invalid
 * @param {string} [path] - Optional custom path to config file
 * @returns {Promise<object>} Configuration object
 */
export async function loadConfig(path) {
  // Calculate config path dynamically if not provided
  const configPath = path ?? (process.env.OPENCODE_TMUX_INTRAY_CONFIG_PATH
    ? process.env.OPENCODE_TMUX_INTRAY_CONFIG_PATH
    : join(homedir(), '.config', 'opencode-tmux-intray', 'opencode-config.json'));

  try {
    const content = await readFile(configPath, 'utf8');
     const config = JSON.parse(content);
       return validateConfig(config);
  } catch (error) {
     if (error.code === 'ENOENT') {
       // File doesn't exist - return defaults
       return DEFAULT_CONFIG;
     }
    // Parse error or other error - log warning and return defaults
    if (error instanceof SyntaxError) {
      console.warn(`[opencode-tmux-intray] Invalid JSON in config file ${configPath}: ${error.message}`);
    } else {
      console.warn(`[opencode-tmux-intray] Failed to load config from ${configPath}: ${error.message}`);
    }
    return DEFAULT_CONFIG;
  }
}

/**
 * Validate configuration structure and return normalized config
 * @param {any} config - Raw configuration object
 * @returns {object} Validated configuration
 */
export function validateConfig(config) {
  // If config is not an object, return defaults
  if (!config || typeof config !== 'object' || Array.isArray(config)) {
    console.warn('[opencode-tmux-intray] Configuration must be an object, using defaults');
    return DEFAULT_CONFIG;
  }

   const result = {
     ...DEFAULT_CONFIG,
     notifications: { ...DEFAULT_CONFIG.notifications },
     sound: { ...DEFAULT_CONFIG.sound },
     tts: { ...DEFAULT_CONFIG.tts }
   };

   // Validate global enabled flag
   if (typeof config.enabled === 'boolean') {
     result.enabled = config.enabled;
   } else if (config.enabled !== undefined) {
     console.warn('[opencode-tmux-intray] \'enabled\' must be a boolean, using default (true)');
   }

   // Validate agentName
   if (typeof config.agentName === 'string') {
     result.agentName = config.agentName;
   } else if (config.agentName !== undefined) {
     console.warn('[opencode-tmux-intray] \'agentName\' must be a string, using default (opencode)');
   }

  // Validate notifications object
  if (config.notifications && typeof config.notifications === 'object' && !Array.isArray(config.notifications)) {
    // Merge each notification config
    for (const [eventType, eventConfig] of Object.entries(config.notifications)) {
      if (eventConfig && typeof eventConfig === 'object' && !Array.isArray(eventConfig)) {
        const defaultEventConfig = DEFAULT_CONFIG.notifications[eventType];
         if (defaultEventConfig) {
            // Merge with defaults
            const mergedConfig = { ...defaultEventConfig, ...eventConfig };
             // Validate status field
             if (mergedConfig.status !== undefined && !['success', 'error', 'pending'].includes(mergedConfig.status)) {
               console.warn(`[opencode-tmux-intray] Status '${mergedConfig.status}' for event '${eventType}' must be one of success, error, pending, using default ('success')`);
               mergedConfig.status = 'success';
             }
             // Validate enabled field
             if (mergedConfig.enabled !== undefined && typeof mergedConfig.enabled !== 'boolean') {
               console.warn(`[opencode-tmux-intray] 'enabled' for event '${eventType}' must be a boolean, using default (true)`);
               mergedConfig.enabled = true;
             }
             // Validate message field
             if (mergedConfig.message !== undefined && typeof mergedConfig.message !== 'string') {
               console.warn(`[opencode-tmux-intray] 'message' for event '${eventType}' must be a string, using default ('')`);
               mergedConfig.message = '';
             }
            result.notifications[eventType] = mergedConfig;
         } else {
          // Unknown event type - preserve but warn
          console.warn(`[opencode-tmux-intray] Unknown event type '${eventType}' in configuration`);
          result.notifications[eventType] = eventConfig;
        }
      } else {
        console.warn(`[opencode-tmux-intray] Invalid configuration for event '${eventType}', using defaults`);
      }
    }
   } else if (config.notifications !== undefined) {
     console.warn('[opencode-tmux-intray] \'notifications\' must be an object, using empty object');
     result.notifications = {};
   }

  // Validate sound settings
  if (config.sound && typeof config.sound === 'object' && !Array.isArray(config.sound)) {
    result.sound = { ...DEFAULT_CONFIG.sound, ...config.sound };
  }

  // Validate tts settings
  if (config.tts && typeof config.tts === 'object' && !Array.isArray(config.tts)) {
    result.tts = { ...DEFAULT_CONFIG.tts, ...config.tts };
  }

  // Warn about unknown top-level keys
  const knownKeys = ['enabled', 'agentName', 'notifications', 'sound', 'tts'];
  for (const key of Object.keys(config)) {
    if (!knownKeys.includes(key)) {
      console.warn(`[opencode-tmux-intray] Unknown configuration key '${key}'`);
      // Preserve unknown keys
      result[key] = config[key];
    }
  }

  return result;
}

/**
 * Check if an event is enabled in the configuration
 * @param {object} config - Configuration object
 * @param {string} eventType - Event type (e.g., 'session.idle')
 * @returns {boolean} True if event is enabled
 */
export function isEventEnabled(config, eventType) {
  // Global disable overrides everything
  if (config.enabled === false) {
    return false;
  }

  const eventConfig = config.notifications[eventType];
  if (!eventConfig) {
    // Unknown event type defaults to disabled
    return false;
  }

  return eventConfig.enabled !== false;
}

/**
 * Get configuration for a specific event
 * @param {object} config - Configuration object
 * @param {string} eventType - Event type
 * @returns {object} Event configuration with enabled, message, status
 */
export function getEventConfig(config, eventType) {
  const eventConfig = config.notifications[eventType];
  if (!eventConfig) {
    // Return a default configuration for unknown events
    return {
      enabled: false,
      message: '',
      status: 'success'
    };
  }

   const enabled = config.enabled !== false && eventConfig.enabled !== false;
   return {
     enabled,
     message: eventConfig.message || '',
     status: eventConfig.status || 'success'
   };
}

/**
 * Substitute template placeholders with values from event object
 * @param {string} template - Template string with {placeholder} syntax
 * @param {object} event - Event object
 * @returns {string} Substituted string
 */
export function substituteTemplate(template, event) {
  if (!template || typeof template !== 'string') {
    return '';
  }

  if (!event || typeof event !== 'object') {
    return template;
  }

  return template.replace(/\{([^{}]+)\}/g, (match, path) => {
    const parts = path.trim().split('.');
    let value = event;
    for (const part of parts) {
      if (value && typeof value === 'object' && part in value) {
        value = value[part];
      } else {
        value = '';
        break;
      }
    }
    return value != null ? String(value) : '';
  });
}