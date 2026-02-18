#!/usr/bin/env node

/**
 * Build script for compiling the OpenCode agent to WASM
 * 
 * This script uses AssemblyScript to compile TypeScript to WebAssembly.
 * The output is a .wasm file that can be loaded by the Go runtime.
 */

const { execSync } = require('child_process');
const fs = require('fs');
const path = require('path');

const isDebug = process.argv.includes('--debug');
const srcDir = path.join(__dirname, '..', 'src');
const distDir = path.join(__dirname, '..', 'dist');

// Ensure dist directory exists
if (!fs.existsSync(distDir)) {
  fs.mkdirSync(distDir, { recursive: true });
}

console.log(`Building Forge Agent WASM (${isDebug ? 'debug' : 'release'})...`);

try {
  // Check if AssemblyScript is installed
  const ascPath = path.join(__dirname, '..', 'node_modules', '.bin', 'asc');
  
  if (!fs.existsSync(ascPath)) {
    console.error('AssemblyScript not found. Run npm install first.');
    process.exit(1);
  }

  // Build flags
  const flags = [
    '--target', isDebug ? 'debug' : 'release',
    '--outFile', path.join(distDir, 'agent.wasm'),
    '--textFile', path.join(distDir, 'agent.wat'),
    '--sourceMap',
    '--exportRuntime'
  ];

  if (!isDebug) {
    flags.push('--optimize');
  }

  // Run AssemblyScript compiler
  const entryFile = path.join(srcDir, 'index.ts');
  const command = `${ascPath} ${entryFile} ${flags.join(' ')}`;
  
  console.log(`Running: ${command}`);
  execSync(command, { stdio: 'inherit', cwd: path.join(__dirname, '..') });

  // Verify output
  const wasmPath = path.join(distDir, 'agent.wasm');
  if (fs.existsSync(wasmPath)) {
    const stats = fs.statSync(wasmPath);
    console.log(`✓ Build successful: ${wasmPath} (${(stats.size / 1024).toFixed(2)} KB)`);
  } else {
    console.error('✗ Build failed: WASM file not created');
    process.exit(1);
  }

} catch (error) {
  console.error('Build failed:', error.message);
  process.exit(1);
}
