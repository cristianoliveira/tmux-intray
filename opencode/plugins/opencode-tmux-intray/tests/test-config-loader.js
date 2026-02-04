/**
 * Unit tests for config-loader.js
 * Uses Vitest test runner
 */
import { describe, test, expect, afterAll, vi } from 'vitest';
import { mkdtemp, writeFile, rm, chmod } from 'node:fs/promises';
import { join } from 'node:path';
import { tmpdir } from 'node:os';

import {
  loadConfig,
  validateConfig,
  isEventEnabled,
  getEventConfig,
  substituteTemplate,
  DEFAULT_CONFIG
} from '../config-loader.js';

// Track temp directories for cleanup
const tempDirs = [];

async function createTempDir(prefix = 'config-loader-test-') {
  const dir = await mkdtemp(join(tmpdir(), prefix));
  tempDirs.push(dir);
  return dir;
}

async function cleanup() {
  await Promise.all(tempDirs.map((dir) => rm(dir, { recursive: true, force: true })));
}

afterAll(cleanup);

// =============================================================================
// loadConfig tests
// =============================================================================

test('loadConfig: returns default config when file does not exist', async () => {
  const tempDir = await createTempDir();
  const nonExistentPath = join(tempDir, 'does-not-exist.json');
  
  const config = await loadConfig(nonExistentPath);
  
  expect(config).toEqual(DEFAULT_CONFIG);
});

test('loadConfig: parses valid JSON and merges with defaults', async () => {
  const tempDir = await createTempDir();
  const configPath = join(tempDir, 'config.json');
  
  const userConfig = {
    agentName: 'custom-agent',
    notifications: {
      'session.idle': {
        message: 'Custom idle message'
      }
    }
  };
  
  await writeFile(configPath, JSON.stringify(userConfig));
  const config = await loadConfig(configPath);
  
  // User overrides should be applied
  expect(config.agentName).toBe('custom-agent');
  expect(config.notifications['session.idle'].message).toBe('Custom idle message');
  
  // Defaults should be preserved for non-overridden fields
  expect(config.enabled).toBe(true);
  expect(config.notifications['session.idle'].enabled).toBe(true);
  expect(config.notifications['session.idle'].status).toBe('success');
  expect(config.notifications['session.error'].enabled).toBe(true);
});

test('loadConfig: returns defaults for invalid JSON (with warning)', async () => {
  const tempDir = await createTempDir();
  const configPath = join(tempDir, 'invalid.json');
  
  await writeFile(configPath, '{ invalid json }');
  
  // Capture console.warn
  const warnings = [];
  const warnSpy = vi.spyOn(console, 'warn').mockImplementation((...args) => {
    warnings.push(args.join(' '));
  });
  
  try {
    const config = await loadConfig(configPath);
    
    expect(config).toEqual(DEFAULT_CONFIG);
    expect(warnings.some(w => w.includes('Invalid JSON'))).toBeTruthy();
  } finally {
    warnSpy.mockRestore();
  }
});

test('loadConfig: returns defaults for permission errors', async () => {
  const tempDir = await createTempDir();
  const configPath = join(tempDir, 'no-read.json');
  
  await writeFile(configPath, JSON.stringify({ agentName: 'test' }));
  
  // Make file unreadable (skip on Windows where chmod doesn't work the same way)
  if (process.platform !== 'win32') {
    await chmod(configPath, 0o000);
    
    // Capture console.warn
    const warnings = [];
  const warnSpy = vi.spyOn(console, 'warn').mockImplementation((...args) => {
    warnings.push(args.join(' '));
  });
    
    try {
      const config = await loadConfig(configPath);
      
      expect(config).toEqual(DEFAULT_CONFIG);
      expect(warnings.some(w => w.includes('Failed to load config'))).toBeTruthy();
    } finally {
      warnSpy.mockRestore();
      // Restore permissions for cleanup
      await chmod(configPath, 0o644);
    }
  }
});

// =============================================================================
// validateConfig tests
// =============================================================================

test('validateConfig: accepts valid config without warnings', async () => {
  const warnings = [];
  const warnSpy = vi.spyOn(console, 'warn').mockImplementation((...args) => {
    warnings.push(args.join(' '));
  });
  
  try {
    const validConfig = {
      enabled: true,
      agentName: 'my-agent',
      notifications: {
        'session.idle': {
          enabled: true,
          message: 'Task done',
          status: 'success'
        }
      }
    };
    
    const result = validateConfig(validConfig);
    
    expect(result.enabled).toBe(true);
    expect(result.agentName).toBe('my-agent');
    expect(result.notifications['session.idle'].enabled).toBe(true);
    expect(warnings.length).toBe(0, 'Should not produce warnings for valid config');
  } finally {
    warnSpy.mockRestore();
  }
});

test('validateConfig: warns and defaults invalid enabled field', async () => {
  const warnings = [];
  const warnSpy = vi.spyOn(console, 'warn').mockImplementation((...args) => {
    warnings.push(args.join(' '));
  });
  
  try {
    const invalidConfig = {
      enabled: 'yes', // Should be boolean
      agentName: 'test'
    };
    
    const result = validateConfig(invalidConfig);
    
    expect(result.enabled).toBe(true, 'Should default to true');
    expect(warnings.some(w => w.includes("'enabled' must be a boolean"))).toBeTruthy();
  } finally {
    warnSpy.mockRestore();
  }
});

test('validateConfig: warns and defaults invalid agentName field', async () => {
  const warnings = [];
  const warnSpy = vi.spyOn(console, 'warn').mockImplementation((...args) => {
    warnings.push(args.join(' '));
  });
  
  try {
    const invalidConfig = {
      agentName: 123 // Should be string
    };
    
    const result = validateConfig(invalidConfig);
    
    expect(result.agentName).toBe('opencode', 'Should default to opencode');
    expect(warnings.some(w => w.includes("'agentName' must be a string"))).toBeTruthy();
  } finally {
    warnSpy.mockRestore();
  }
});

test('validateConfig: warns and defaults invalid notifications structure', async () => {
  const warnings = [];
  const warnSpy = vi.spyOn(console, 'warn').mockImplementation((...args) => {
    warnings.push(args.join(' '));
  });
  
  try {
    const invalidConfig = {
      notifications: 'not an object' // Should be object
    };
    
    const result = validateConfig(invalidConfig);
    
    expect(result.notifications).toEqual({}, 'Should default to empty object');
    expect(warnings.some(w => w.includes("'notifications' must be an object"))).toBeTruthy();
  } finally {
    warnSpy.mockRestore();
  }
});

test('validateConfig: warns and defaults notifications as array', async () => {
  const warnings = [];
  const warnSpy = vi.spyOn(console, 'warn').mockImplementation((...args) => {
    warnings.push(args.join(' '));
  });
  
  try {
    const invalidConfig = {
      notifications: ['not', 'valid'] // Arrays should not be allowed
    };
    
    const result = validateConfig(invalidConfig);
    
    expect(result.notifications).toEqual({}, 'Should default to empty object');
    expect(warnings.some(w => w.includes("'notifications' must be an object"))).toBeTruthy();
  } finally {
    warnSpy.mockRestore();
  }
});

test('validateConfig: warns for unknown top-level keys', async () => {
  const warnings = [];
  const warnSpy = vi.spyOn(console, 'warn').mockImplementation((...args) => {
    warnings.push(args.join(' '));
  });
  
  try {
    const configWithUnknownKeys = {
      enabled: true,
      unknownKey: 'value',
      anotherUnknown: 123
    };
    
    const result = validateConfig(configWithUnknownKeys);
    
    // Unknown keys should be preserved but warned about
    expect(result.unknownKey).toBe('value');
    expect(warnings.some(w => w.includes("Unknown configuration key 'unknownKey'"))).toBeTruthy();
    expect(warnings.some(w => w.includes("Unknown configuration key 'anotherUnknown'"))).toBeTruthy();
  } finally {
    warnSpy.mockRestore();
  }
});

test('validateConfig: warns for invalid status values', async () => {
  const warnings = [];
  const warnSpy = vi.spyOn(console, 'warn').mockImplementation((...args) => {
    warnings.push(args.join(' '));
  });
  
  try {
    const invalidConfig = {
      notifications: {
        'session.idle': {
          enabled: true,
          status: 'invalid-status' // Should be success, error, or pending
        }
      }
    };
    
    const result = validateConfig(invalidConfig);
    
    expect(result.notifications['session.idle'].status).toBe('success', 'Should default to success');
    expect(warnings.some(w => w.includes("must be one of success, error, pending"))).toBeTruthy();
  } finally {
    warnSpy.mockRestore();
  }
});

test('validateConfig: returns defaults when config is null', async () => {
  const warnings = [];
  const warnSpy = vi.spyOn(console, 'warn').mockImplementation((...args) => {
    warnings.push(args.join(' '));
  });
  
  try {
    const result = validateConfig(null);
    
    expect(result).toEqual(DEFAULT_CONFIG);
    expect(warnings.some(w => w.includes('Configuration must be an object'))).toBeTruthy();
  } finally {
    warnSpy.mockRestore();
  }
});

test('validateConfig: returns defaults when config is an array', async () => {
  const warnings = [];
  const warnSpy = vi.spyOn(console, 'warn').mockImplementation((...args) => {
    warnings.push(args.join(' '));
  });
  
  try {
    const result = validateConfig([1, 2, 3]);
    
    expect(result).toEqual(DEFAULT_CONFIG);
    expect(warnings.some(w => w.includes('Configuration must be an object'))).toBeTruthy();
  } finally {
    warnSpy.mockRestore();
  }
});

// =============================================================================
// isEventEnabled tests
// =============================================================================

test('isEventEnabled: returns true for enabled default events', () => {
  const config = { ...DEFAULT_CONFIG };
  
  expect(isEventEnabled(config, 'session.idle')).toBe(true);
  expect(isEventEnabled(config, 'session.error')).toBe(true);
  expect(isEventEnabled(config, 'permission.updated')).toBe(true);
});

test('isEventEnabled: returns false when global enabled is false', () => {
  const config = {
    ...DEFAULT_CONFIG,
    enabled: false
  };
  
  // Even normally-enabled events should return false
  expect(isEventEnabled(config, 'session.idle')).toBe(false);
  expect(isEventEnabled(config, 'session.error')).toBe(false);
  expect(isEventEnabled(config, 'permission.updated')).toBe(false);
});

test('isEventEnabled: returns false for disabled events', () => {
  const config = { ...DEFAULT_CONFIG };
  
  // These events are disabled by default
  expect(isEventEnabled(config, 'session.status')).toBe(false);
  expect(isEventEnabled(config, 'session.created')).toBe(false);
  expect(isEventEnabled(config, 'permission.replied')).toBe(false);
});

test('isEventEnabled: returns false for unconfigured events', () => {
  const config = { ...DEFAULT_CONFIG };
  
  // Unknown event types should default to disabled
  expect(isEventEnabled(config, 'unknown.event')).toBe(false);
  expect(isEventEnabled(config, 'custom.notification')).toBe(false);
});

test('isEventEnabled: respects user overrides', () => {
  const config = {
    ...DEFAULT_CONFIG,
    notifications: {
      ...DEFAULT_CONFIG.notifications,
      'session.idle': {
        ...DEFAULT_CONFIG.notifications['session.idle'],
        enabled: false
      },
      'session.status': {
        enabled: true
      }
    }
  };
  
  // User disabled session.idle
  expect(isEventEnabled(config, 'session.idle')).toBe(false);
  // User enabled session.status
  expect(isEventEnabled(config, 'session.status')).toBe(true);
});

// =============================================================================
// getEventConfig tests
// =============================================================================

test('getEventConfig: returns default config for default events', () => {
  const config = { ...DEFAULT_CONFIG };
  
  const idleConfig = getEventConfig(config, 'session.idle');
  
  expect(idleConfig.enabled).toBe(true);
  expect(idleConfig.message).toBe('Task completed');
  expect(idleConfig.status).toBe('success');
});

test('getEventConfig: merges user config with defaults', () => {
  const config = {
    ...DEFAULT_CONFIG,
    notifications: {
      ...DEFAULT_CONFIG.notifications,
      'session.idle': {
        message: 'Custom message'
        // enabled and status not specified, should come from defaults
      }
    }
  };
  
  const idleConfig = getEventConfig(config, 'session.idle');
  
  expect(idleConfig.enabled).toBe(true, 'Should preserve default enabled');
  expect(idleConfig.message).toBe('Custom message', 'Should use user message');
  expect(idleConfig.status).toBe('success', 'Should preserve default status');
});

test('getEventConfig: returns base defaults for unconfigured events', () => {
  const config = { ...DEFAULT_CONFIG };
  
  const unknownConfig = getEventConfig(config, 'unknown.event');
  
  expect(unknownConfig.enabled).toBe(false);
  expect(unknownConfig.message).toBe('');
  expect(unknownConfig.status).toBe('success');
});

test('getEventConfig: applies global enabled=false', () => {
  const config = {
    ...DEFAULT_CONFIG,
    enabled: false
  };
  
  const idleConfig = getEventConfig(config, 'session.idle');
  
  // Global enabled=false should override event-specific enabled
  expect(idleConfig.enabled).toBe(false);
  // Other fields should remain
  expect(idleConfig.message).toBe('Task completed');
  expect(idleConfig.status).toBe('success');
});

test('getEventConfig: handles complete user override', () => {
  const config = {
    ...DEFAULT_CONFIG,
    notifications: {
      ...DEFAULT_CONFIG.notifications,
      'session.idle': {
        enabled: false,
        message: 'Fully custom',
        status: 'pending'
      }
    }
  };
  
  const idleConfig = getEventConfig(config, 'session.idle');
  
  expect(idleConfig.enabled).toBe(false);
  expect(idleConfig.message).toBe('Fully custom');
  expect(idleConfig.status).toBe('pending');
});

// =============================================================================
// substituteTemplate tests
// =============================================================================

test('substituteTemplate: replaces simple placeholders', () => {
  const template = 'Task: {type}';
  const event = { type: 'session.idle' };
  
  const result = substituteTemplate(template, event);
  
  expect(result).toBe('Task: session.idle');
});

test('substituteTemplate: replaces multiple placeholders', () => {
  const template = 'Session {sessionId} has status {status}';
  const event = { sessionId: '123', status: 'completed' };
  
  const result = substituteTemplate(template, event);
  
  expect(result).toBe('Session 123 has status completed');
});

test('substituteTemplate: replaces nested property placeholders', () => {
  const template = 'Status: {properties.status}';
  const event = { properties: { status: 'pending' } };
  
  const result = substituteTemplate(template, event);
  
  expect(result).toBe('Status: pending');
});

test('substituteTemplate: handles deeply nested properties', () => {
  const template = 'Value: {a.b.c.d}';
  const event = { a: { b: { c: { d: 'deep-value' } } } };
  
  const result = substituteTemplate(template, event);
  
  expect(result).toBe('Value: deep-value');
});

test('substituteTemplate: returns empty string for missing properties', () => {
  const template = 'Title: {missing}';
  const event = { type: 'test' };
  
  const result = substituteTemplate(template, event);
  
  expect(result).toBe('Title: ');
});

test('substituteTemplate: returns empty string for missing nested properties', () => {
  const template = 'Value: {missing.nested.prop}';
  const event = { other: 'value' };
  
  const result = substituteTemplate(template, event);
  
  expect(result).toBe('Value: ');
});

test('substituteTemplate: handles null event gracefully', () => {
  const template = 'Hello {name}!';
  
  const result = substituteTemplate(template, null);
  
  expect(result).toBe('Hello {name}!');
});

test('substituteTemplate: handles undefined event gracefully', () => {
  const template = 'Hello {name}!';
  
  const result = substituteTemplate(template, undefined);
  
  expect(result).toBe('Hello {name}!');
});

test('substituteTemplate: handles null template', () => {
  const result = substituteTemplate(null, { name: 'test' });
  
  expect(result).toBe('');
});

test('substituteTemplate: handles undefined template', () => {
  const result = substituteTemplate(undefined, { name: 'test' });
  
  expect(result).toBe('');
});

test('substituteTemplate: handles empty template', () => {
  const result = substituteTemplate('', { name: 'test' });
  
  expect(result).toBe('');
});

test('substituteTemplate: converts non-string values to strings', () => {
  const template = 'Count: {count}, Active: {active}';
  const event = { count: 42, active: true };
  
  const result = substituteTemplate(template, event);
  
  expect(result).toBe('Count: 42, Active: true');
});

test('substituteTemplate: handles null property values', () => {
  const template = 'Value: {nullValue}';
  const event = { nullValue: null };
  
  const result = substituteTemplate(template, event);
  
  expect(result).toBe('Value: ');
});

test('substituteTemplate: preserves text without placeholders', () => {
  const template = 'No placeholders here';
  const event = { name: 'test' };
  
  const result = substituteTemplate(template, event);
  
  expect(result).toBe('No placeholders here');
});
