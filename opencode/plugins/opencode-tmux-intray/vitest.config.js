import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    // Use globals for compatibility with test files that use globals (like describe, it, etc.)
    globals: true,
    // Use happy-dom for DOM APIs if needed (not required for Node.js tests)
    // environment: 'happy-dom',
    // Setup files to run before each test file
    setupFiles: [],
    // Match test files
    include: ['tests/**/*.js'],
    // Exclude files
    exclude: ['node_modules', 'dist', '.tmp'],
    // Coverage configuration
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html'],
      exclude: [
        'node_modules/',
        'dist/',
        '.tmp/',
        'tests/',
        '**/*.config.js'
      ]
    }
  }
});