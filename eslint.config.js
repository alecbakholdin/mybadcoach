// Strict, shared ESLint configuration for every workspace (see docs/testing.md).
//
// Policy notes:
// - Every enabled rule is error-level or off. Plugin presets that ship warnings
//   are normalised below, so `npm run lint` never reports warning-only noise.
// - Escape hatches must name the rule and state the reason inline:
//   `// eslint-disable-next-line <rule> -- <reason>` and
//   `// @ts-expect-error <reason>`. Unused disable directives fail the run.
// - Generated code (buf `gen/`) is excluded; its inputs are the source of
//   record and it carries its own `eslint-disable` header.
import js from '@eslint/js';
import importX from 'eslint-plugin-import-x';
import playwright from 'eslint-plugin-playwright';
import svelte from 'eslint-plugin-svelte';
import globals from 'globals';
import tseslint from 'typescript-eslint';

/** Presets that ship `warn` rules are promoted to errors (no warning noise). */
function promoteWarnings(rules) {
	return Object.fromEntries(
		Object.entries(rules).map(([name, value]) => {
			const severity = Array.isArray(value) ? value[0] : value;
			if (severity === 'warn' || severity === 1) {
				return [name, Array.isArray(value) ? ['error', ...value.slice(1)] : 'error'];
			}
			return [name, value];
		}),
	);
}

const importOrder = [
	'error',
	{
		groups: ['builtin', 'external', 'internal', ['parent', 'sibling', 'index']],
		pathGroups: [
			{ pattern: '$app/**', group: 'internal', position: 'before' },
			{ pattern: '$shared', group: 'internal' },
			{ pattern: '$shared/**', group: 'internal' },
			{ pattern: '#lib', group: 'internal' },
			{ pattern: '#lib/**', group: 'internal' },
		],
		pathGroupsExcludedImportTypes: ['builtin'],
		'newlines-between': 'always',
		alphabetize: { order: 'asc', caseInsensitive: true },
	},
];

export default tseslint.config(
	// Global ignores: this object may contain only `ignores` (and `name`), or
	// ESLint treats it as a per-config filter instead of excluding the files.
	{
		ignores: [
			'**/node_modules/**',
			'**/dist/**',
			'**/build/**',
			// The Go server embeds the client build here (populated by
			// scripts/copy-client-build.sh); it is build output, not source.
			'server/web/static/**',
			// Claude Code's own harness directory (agent worktrees, settings) —
			// never application source.
			'.claude/**',
			'**/.svelte-kit/**',
			'**/coverage/**',
			'**/test-results/**',
			'**/playwright-report/**',
			'**/blob-report/**',
			'server/internal/gen/**',
			'packages/shared/src/gen/**',
			// The lint config itself is configuration, not application code.
			'eslint.config.js',
		],
	},
	{
		linterOptions: {
			reportUnusedDisableDirectives: 'error',
		},
	},

	js.configs.recommended,
	...tseslint.configs.strictTypeChecked,

	// Svelte parser setup for .svelte files (records include the script block).
	...svelte.configs.recommended,

	// These ship as warnings in the Svelte preset; the policy is error-or-off
	// everywhere. This block must follow the preset (later configs win) and is
	// unscoped so no config inherits the warning severity.
	{
		rules: {
			'svelte/no-at-debug-tags': 'error',
			'svelte/no-inspect': 'error',
		},
	},

	{
		files: ['**/*.ts', '**/*.svelte'],
		languageOptions: {
			parserOptions: {
				projectService: true,
				tsconfigRootDir: import.meta.dirname,
				extraFileExtensions: ['.svelte'],
			},
		},
		rules: {
			// Policy from docs/testing.md, stated explicitly even where the
			// preset already covers it.
			'@typescript-eslint/no-explicit-any': 'error',
			'@typescript-eslint/no-unused-vars': [
				'error',
				{
					args: 'all',
					argsIgnorePattern: '^_',
					varsIgnorePattern: '^_',
					caughtErrors: 'all',
					caughtErrorsIgnorePattern: '^_',
					destructuredArrayIgnorePattern: '^_',
					ignoreRestSiblings: true,
				},
			],
			'@typescript-eslint/ban-ts-comment': [
				'error',
				{
					'ts-expect-error': 'allow-with-description',
					'ts-ignore': true,
					'ts-nocheck': true,
					minimumDescriptionLength: 10,
				},
			],
			'no-console': 'error',
			// Numbers are safe to interpolate; objects/unknowns are not.
			'@typescript-eslint/restrict-template-expressions': ['error', { allowNumber: true }],
		},
	},

	// Svelte 5 runes: parse components (and `.svelte.ts` rune modules) in runes
	// mode and type the script block with the TypeScript parser.
	{
		files: ['**/*.svelte', '**/*.svelte.ts'],
		languageOptions: {
			globals: { ...globals.browser },
			parserOptions: {
				parser: tseslint.parser,
				svelteFeatures: { runes: true },
			},
		},
	},

	// Import sorting/grouping: builtins, external, internal aliases, relative.
	{
		files: ['**/*.ts', '**/*.svelte'],
		plugins: { 'import-x': importX },
		rules: {
			'import-x/order': importOrder,
			'import-x/no-duplicates': 'error',
			// Module resolution is TypeScript's and Vite's job (including the
			// native `.ts` specifiers in generated code); import-x only orders.
			'import-x/no-unresolved': 'off',
		},
	},

	// Runtime globals per workspace.
	{
		files: ['apps/client/src/**/*.ts', 'apps/client/src/**/*.svelte'],
		languageOptions: { globals: { ...globals.browser } },
	},
	{
		files: ['packages/shared/**/*.ts'],
		languageOptions: { globals: { ...globals.node } },
	},
	{
		files: ['e2e/**/*.ts', 'apps/client/test-server/**/*.ts', '**/*.config.ts'],
		languageOptions: { globals: { ...globals.node } },
	},

	// Playwright e2e specs: recommended preset with warnings promoted to errors.
	{
		files: ['e2e/**/*.ts'],
		plugins: playwright.configs['flat/recommended'].plugins,
		rules: promoteWarnings(playwright.configs['flat/recommended'].rules),
	},

	// The stub Connect server is a Node startup process; logging is explicit.
	{
		files: ['apps/client/test-server/**/*.ts'],
		rules: { 'no-console': 'off' },
	},
);
