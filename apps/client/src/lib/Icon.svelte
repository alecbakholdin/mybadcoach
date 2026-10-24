<script lang="ts">
	// Small inline stroke icons, drawn in `currentColor` so they inherit the
	// surrounding text colour. One component per icon would be 14 near-empty
	// files; a single name-indexed component keeps the set in one place and lets
	// every consumer size/colour it the same way. Icons are decorative and never
	// carry an accessible name of their own (`aria-hidden`), so the text label
	// beside each one stays the source of truth.
	export type IconName =
		| 'team'
		| 'calendar'
		| 'plus'
		| 'close'
		| 'menu'
		| 'chevron-right'
		| 'gear'
		| 'grid'
		| 'chart'
		| 'search'
		| 'trash'
		| 'pencil'
		| 'sort'
		| 'user-plus';

	let {
		name,
		size = 16,
		class: className = '',
	}: { name: IconName; size?: number; class?: string } = $props();
</script>

<svg
	class={className}
	width={size}
	height={size}
	viewBox="0 0 24 24"
	fill="none"
	aria-hidden="true"
>
	{#if name === 'team'}
		<path
			d="M16.5 19.5v-1.3a3.9 3.9 0 0 0-3.9-3.9H6.4a3.9 3.9 0 0 0-3.9 3.9v1.3"
			stroke="currentColor"
			stroke-width="1.6"
			stroke-linecap="round"
		/>
		<circle cx="9.5" cy="7.2" r="3.2" stroke="currentColor" stroke-width="1.6" />
		<path
			d="M21.5 19.5v-1.3a3.9 3.9 0 0 0-2.9-3.77M15.7 4.14a3.2 3.2 0 0 1 0 6.12"
			stroke="currentColor"
			stroke-width="1.6"
			stroke-linecap="round"
		/>
	{:else if name === 'calendar'}
		<rect
			x="3.5"
			y="5"
			width="17"
			height="15.5"
			rx="2.5"
			stroke="currentColor"
			stroke-width="1.6"
		/>
		<path
			d="M3.5 9.6h17M8 3v3.6M16 3v3.6"
			stroke="currentColor"
			stroke-width="1.6"
			stroke-linecap="round"
		/>
	{:else if name === 'plus'}
		<path d="M12 5v14M5 12h14" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" />
	{:else if name === 'close'}
		<path
			d="m6 6 12 12M18 6 6 18"
			stroke="currentColor"
			stroke-width="1.7"
			stroke-linecap="round"
		/>
	{:else if name === 'menu'}
		<path
			d="M4 7h16M4 12h16M4 17h10"
			stroke="currentColor"
			stroke-width="1.7"
			stroke-linecap="round"
		/>
	{:else if name === 'chevron-right'}
		<path
			d="m9 5 7 7-7 7"
			stroke="currentColor"
			stroke-width="1.7"
			stroke-linecap="round"
			stroke-linejoin="round"
		/>
	{:else if name === 'gear'}
		<circle cx="12" cy="12" r="3" stroke="currentColor" stroke-width="1.6" />
		<path
			d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1Z"
			stroke="currentColor"
			stroke-width="1.6"
			stroke-linejoin="round"
		/>
	{:else if name === 'grid'}
		<rect x="3.5" y="3.5" width="7" height="7" rx="1.6" stroke="currentColor" stroke-width="1.6" />
		<rect x="13.5" y="3.5" width="7" height="7" rx="1.6" stroke="currentColor" stroke-width="1.6" />
		<rect x="3.5" y="13.5" width="7" height="7" rx="1.6" stroke="currentColor" stroke-width="1.6" />
		<rect
			x="13.5"
			y="13.5"
			width="7"
			height="7"
			rx="1.6"
			stroke="currentColor"
			stroke-width="1.6"
		/>
	{:else if name === 'chart'}
		<path
			d="M5 20v-7M12 20V5M19 20v-4"
			stroke="currentColor"
			stroke-width="1.8"
			stroke-linecap="round"
		/>
	{:else if name === 'search'}
		<circle cx="10.8" cy="10.8" r="6.3" stroke="currentColor" stroke-width="1.6" />
		<path d="m15.5 15.5 4.5 4.5" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" />
	{:else if name === 'trash'}
		<path
			d="M4 6.5h16M9 6.5V4.8A1.8 1.8 0 0 1 10.8 3h2.4A1.8 1.8 0 0 1 15 4.8v1.7M6.5 6.5l.8 12.1A2.1 2.1 0 0 0 9.4 20.6h5.2a2.1 2.1 0 0 0 2.1-2L17.5 6.5"
			stroke="currentColor"
			stroke-width="1.6"
			stroke-linecap="round"
			stroke-linejoin="round"
		/>
		<path
			d="M10 10.5v6M14 10.5v6"
			stroke="currentColor"
			stroke-width="1.6"
			stroke-linecap="round"
		/>
	{:else if name === 'pencil'}
		<path
			d="M4 20h4l10.5-10.5a2.1 2.1 0 0 0-3-3L5 17v3Z"
			stroke="currentColor"
			stroke-width="1.6"
			stroke-linejoin="round"
		/>
		<path d="m13.5 7.5 3 3" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" />
	{:else if name === 'sort'}
		<path
			d="M12 19V5M6 11l6-6 6 6"
			stroke="currentColor"
			stroke-width="1.8"
			stroke-linecap="round"
			stroke-linejoin="round"
		/>
	{:else}
		<path
			d="M15 19.5v-1.3a3.9 3.9 0 0 0-3.9-3.9H6.4a3.9 3.9 0 0 0-3.9 3.9v1.3"
			stroke="currentColor"
			stroke-width="1.6"
			stroke-linecap="round"
		/>
		<circle cx="9" cy="7.2" r="3.2" stroke="currentColor" stroke-width="1.6" />
		<path d="M18.5 8v6M15.5 11h6" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" />
	{/if}
</svg>
