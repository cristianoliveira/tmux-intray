#!/usr/bin/env node

/**
 * tmux-intray - Platform-aware binary wrapper
 * 
 * This script detects the current platform and architecture,
 * then executes the appropriate pre-built binary.
 */

const path = require('path');
const { spawn } = require('child_process');

function getPlatform() {
  switch (process.platform) {
    case 'darwin':
      return 'darwin';
    case 'linux':
      return 'linux';
    case 'win32':
      return 'windows';
    default:
      throw new Error(`Unsupported platform: ${process.platform}`);
  }
}

function getArch() {
  switch (process.arch) {
    case 'x64':
      return 'amd64';
    case 'arm64':
      return 'arm64';
    default:
      throw new Error(`Unsupported architecture: ${process.arch}`);
  }
}

function getBinaryName() {
  const platform = getPlatform();
  const arch = getArch();
  const ext = platform === 'windows' ? '.exe' : '';
  return `tmux-intray_${platform}_${arch}${ext}`;
}

function getBinaryPath() {
  return path.join(__dirname, '..', 'dist', getBinaryName());
}

function main() {
  const binaryPath = getBinaryPath();
  const args = process.argv.slice(2);

  const child = spawn(binaryPath, args, {
    stdio: 'inherit',
    env: process.env,
    shell: false
  });

  child.on('error', (err) => {
    if (err.code === 'ENOENT') {
      console.error(`Error: Binary not found at ${binaryPath}`);
      console.error('');
      console.error('This may happen if:');
      console.error('  - The npm package was not published correctly');
      console.error('  - Your platform/architecture is not supported');
      console.error('');
      console.error(`Detected: ${process.platform}/${process.arch}`);
      process.exit(1);
    } else {
      console.error(`Error executing binary: ${err.message}`);
      process.exit(1);
    }
  });

  child.on('close', (code) => {
    process.exit(code || 0);
  });
}

main();
