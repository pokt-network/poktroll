#!/usr/bin/env node
// Ensures Docusaurus runs under Node 18 by relaxing the engine constraint.
// This is a workaround while the docs toolchain still uses Node 18 locally.

'use strict';

const fs = require('fs');
const path = require('path');

const PACKAGE_RELATIVE_PATH = path.join(
  '..',
  '..',
  '..',
  'docusaurus',
  'node_modules',
  '@docusaurus',
  'core',
  'package.json',
);

const ABSOLUTE_PACKAGE_PATH = path.resolve(__dirname, PACKAGE_RELATIVE_PATH);
const MIN_SUPPORTED_VERSION = '>=18.0.0';

if (!fs.existsSync(ABSOLUTE_PACKAGE_PATH)) {
  console.error(
    `Skipped Docusaurus engine patch: missing ${ABSOLUTE_PACKAGE_PATH}. ` +
      'Run `yarn install` inside the docusaurus workspace first.',
  );
  process.exit(0);
}

try {
  const rawPackageJson = fs.readFileSync(ABSOLUTE_PACKAGE_PATH, 'utf8');
  const packageJson = JSON.parse(rawPackageJson);

  if (!packageJson.engines || typeof packageJson.engines.node !== 'string') {
    // Nothing to update.
    process.exit(0);
  }

  const currentRequirement = packageJson.engines.node;
  if (currentRequirement === MIN_SUPPORTED_VERSION) {
    process.exit(0);
  }

  if (!currentRequirement.trim().startsWith('>=20')) {
    // Unexpected requirement; avoid overwriting.
    process.exit(0);
  }

  packageJson.engines.node = MIN_SUPPORTED_VERSION;
  const formattedJson = `${JSON.stringify(packageJson, null, 2)}\n`;
  fs.writeFileSync(ABSOLUTE_PACKAGE_PATH, formattedJson, 'utf8');
  console.log(
    `Patched @docusaurus/core engine constraint from ${currentRequirement} to ${MIN_SUPPORTED_VERSION}.`,
  );
} catch (error) {
  console.error('Failed to patch @docusaurus/core package.json:', error);
  process.exitCode = 1;
}
