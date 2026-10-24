import { teams } from '#lib/api';
import { session } from '#lib/session.svelte';
import type { TeamListData } from '#lib/structure-data';

// `ListMyTeams` is the only team listing (no global directory); each membership
// carries the caller's role, which the detail screen uses to gate destructive
// actions.
export const load = async (): Promise<TeamListData> => {
	// Layout and page loads run in parallel. Wait for the memoized `Me`
	// hydration and skip the RPC when signed out, so the layout guard's clean
	// redirect wins instead of a permission-denied racing it.
	if ((await session.load()) === null) {
		return { memberships: [] };
	}
	const response = await teams.listMyTeams({});
	return { memberships: response.memberships };
};
