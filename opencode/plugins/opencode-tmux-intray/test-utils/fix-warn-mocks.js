import fs from 'fs/promises';
import path from 'path';

async function fixFile(filePath) {
  console.log(`Fixing ${filePath}`);
  let content = await fs.readFile(filePath, 'utf8');
  
  // Replace the pattern: const warnings = [];\n  const originalWarn = console.warn;\n  console.warn = (...args) => warnings.push(args.join(' '));
  // with spy pattern
  content = content.replace(
    /const warnings = \[\];\s*\n\s*const originalWarn = console\.warn;\s*\n\s*console\.warn = \(\.\.\.args\) => warnings\.push\(args\.join\(' '\)\);/g,
    `const warnings = [];\n  const warnSpy = vi.spyOn(console, 'warn').mockImplementation((...args) => {\n    warnings.push(args.join(' '));\n  });`
  );
  
  // Replace restore pattern: console.warn = originalWarn;
  content = content.replace(
    /console\.warn = originalWarn;/g,
    `warnSpy.mockRestore();`
  );
  
  // Remove the (t) parameter from test functions
  content = content.replace(
    /test\('([^']+)', async \(t\) =>/g,
    `test('$1', async () =>`
  );
  
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
    await fixFile(path.join(process.cwd(), '..', file));
  }
  
  console.log('Fix complete');
}

main().catch(console.error);