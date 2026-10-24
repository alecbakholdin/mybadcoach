/**
 * Display helpers for players.
 *
 * A player's `names` list starts with the display name and carries aliases
 * after it; the slug is the fallback when the list is empty.
 */
import type { Player } from '$shared/gen/mybadcoach/players/v1/players_pb';

/** The player's display name, falling back to the slug when unnamed. */
export function displayName(player: Player): string {
	return player.names[0] ?? player.slug;
}

/** The player's aliases as one comma-separated string. */
export function aliases(player: Player): string {
	return player.names.slice(1).join(', ');
}
