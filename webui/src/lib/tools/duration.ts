export function toHumanDuration(millis: number): string {
	const seconds = Math.max(millis / 1000, 0);
	if (seconds < 60) {
		return `${seconds}s`;
	}
	if (seconds < 3600) {
		return `${Math.floor(seconds / 60)}m`;
	}
	if (seconds < 86400) {
		return `${Math.floor(seconds / 3600)}h`;
	}
	const days = Math.floor(seconds / 86400);
	if (days <= 7) {
		return `${days}d`;
	}
	const weeks = Math.floor(days / 7);
	return `${weeks}w`;
}
