<script lang="ts">
	import type { TeamRole } from '#lib/structure-data';

	let { role }: { role: TeamRole } = $props();

	const isOwner = $derived(role === 'owner');
</script>

<span class="badge" class:badge-owner={isOwner}>{isOwner ? 'Owner' : 'Member'}</span>
