#!/usr/bin/env node
/**
 * OpenCode Tmux Intray Configuration Generator
 *
 * Generates an initial configuration file with all supported OpenCode events,
 * comments, and examples.
 *
 * Usage:
 *   node generate-config.js [--output <file>] [--stdout] [--help]
 *   ./generate-config.js [--output <file>] [--stdout] [--help]
 *
 * If no output file is specified, writes JSON to ~/.config/opencode-tmux-intray/opencode-config.json. Use --stdout to print instead.
 */

import { mkdir, writeFile } from 'node:fs/promises';
import { join, dirname } from 'node:path';
import { homedir } from 'node:os';
import { parseArgs } from 'node:util';

const DEFAULT_OUTPUT = join(homedir(), '.config', 'opencode-tmux-intray', 'opencode-config.json');

/**
 * Generate a comprehensive configuration object with all supported events.
 * @returns {object} Configuration object
 */
function generateConfig() {
  return {
    _comment: "OpenCode Tmux Intray Plugin Configuration",
    enabled: true,
    agentName: "opencode",
    notifications: {
      "session.idle": {
        _comment: "Session completed successfully",
        enabled: true,
        message: "Task completed",
        status: "success"
      },
      "session.error": {
        _comment: "Session encountered an error",
        enabled: true,
        message: "Session error",
        status: "error"
      },
      "session.status": {
        _comment: "Session status changed (e.g., waiting for input)",
        enabled: false,
        message: "Session status: {status}",
        status: "pending"
      },
      "session.created": {
        _comment: "New session created",
        enabled: false,
        message: "New session created",
        status: "success"
      },
      "session.updated": {
        _comment: "Session updated (e.g., new messages)",
        enabled: false,
        message: "Session updated",
        status: "success"
      },
      "permission.updated": {
        _comment: "AI waiting for user input/permission",
        enabled: true,
        message: "Permission needed",
        status: "pending"
      },
      "permission.replied": {
        _comment: "Permission response received (no notification needed)",
        enabled: false,
        message: "Permission replied",
        status: "success"
      },
      "question.asked": {
        _comment: "AI asked a question",
        enabled: false,
        message: "Question asked: {question}",
        status: "pending"
      },
      "permission.asked": {
        _comment: "AI asked for permission",
        enabled: false,
        message: "Permission asked: {permission}",
        status: "pending"
      }
    },
    sound: {
      _comment: "Sound notification settings (overrides global opencode-tmux-intray sound settings)",
      enabled: true,
      file: "/System/Library/Sounds/Glass.aiff"
    },
    tts: {
      _comment: "Text-to-speech settings (macOS only)",
      enabled: false,
      message: "Agent {agentName} completed with status {status}",
      voice: "Alex"
    }
  };
}

/**
 * Main function
 */
async function main() {
  try {
    const { values } = parseArgs({
      options: {
        output: {
          type: 'string',
          short: 'o',
          description: `Output file path (default: ${DEFAULT_OUTPUT})`
        },
        stdout: {
          type: 'boolean',
          short: 's',
          description: 'Print configuration to stdout instead of writing file'
        },
        help: {
          type: 'boolean',
          short: 'h',
          description: 'Show help message'
        }
      },
      allowPositionals: false
    });

    if (values.help) {
      console.log(`Usage: ${process.argv[1]} [--output <file>] [--stdout] [--help]
Generate an initial configuration file for OpenCode Tmux Intray plugin.

Options:
  -o, --output <file>   Write configuration to file (default: ${DEFAULT_OUTPUT})
  -s, --stdout          Print configuration to stdout instead of writing file
  -h, --help            Show this help message

Examples:
  ${process.argv[1]}                 # Write to ${DEFAULT_OUTPUT}
  ${process.argv[1]} --stdout        # Print to console
  ${process.argv[1]} --output /tmp/opencode-config.json
  ${process.argv[1]} --stdout | jq . # Pretty-print to console
`);
      process.exit(0);
    }

    const config = generateConfig();
    const json = JSON.stringify(config, null, 2);

    const targetPath = values.stdout ? null : (values.output || DEFAULT_OUTPUT);

    if (targetPath === '-') {
      console.log(json);
      return;
    }

    if (targetPath) {
      await mkdir(dirname(targetPath), { recursive: true });
      await writeFile(targetPath, json + '\n');
      console.error(`Configuration written to ${targetPath}`);
    } else {
      console.log(json);
    }
  } catch (error) {
    console.error('Error:', error.message);
    process.exit(1);
  }
}

// Run main if script is executed directly
if (import.meta.url === `file://${process.argv[1]}`) {
  main();
}