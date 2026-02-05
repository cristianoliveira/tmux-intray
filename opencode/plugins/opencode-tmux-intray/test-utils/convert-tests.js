import fs from 'fs/promises';
import path from 'path';

async function convertFile(filePath) {
  console.log(`Converting ${filePath}`);
  let content = await fs.readFile(filePath, 'utf8');
  
  // Replace import lines
  content = content.replace(
    /import { test } from 'node:test';\s*\nimport assert from 'node:assert\/strict';/,
    `import { describe, test, expect, afterAll, vi } from 'vitest';`
  );
  
  // Replace test.after with afterAll
  content = content.replace(/test\.after\((\w+)\);/g, 'afterAll($1);');
  
  // Replace assert.deepEqual(a, b) -> expect(a).toEqual(b)
  content = content.replace(/assert\.deepEqual\(([^,]+),\s*([^)]+)\)/g, 'expect($1).toEqual($2)');
  
  // Replace assert.equal(a, b) -> expect(a).toBe(b)
  content = content.replace(/assert\.equal\(([^,]+),\s*([^)]+)\)/g, 'expect($1).toBe($2)');
  
  // Replace assert.ok(expr, msg) -> expect(expr).toBeTruthy()
  content = content.replace(/assert\.ok\(([^,]+)(?:,\s*[^)]+)?\)/g, 'expect($1).toBeTruthy()');
  
  // Replace console.warn mocking with vi.spyOn
  // This is more complex, we'll need to handle each case manually
  // For now, we'll keep as is and later adjust manually
  
  await fs.writeFile(filePath, content);
}

async function main() {
  const testFiles = [
    'tests/test-config-loader.js',
    'tests/test-integration.js',
    'tests/test-plugin.js',
    'tests/test-real.js'
  ];
  
  for (const file of testFiles) {
    await convertFile(path.join(process.cwd(), '..', file));
  }
  
  console.log('Conversion complete');
}

main().catch(console.error);