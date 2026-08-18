// toPreciseDuration formats a duration with its two largest units, e.g.
// "4m 14s" or "1h 5m". Useful for countdowns where toHumanDuration is too
// coarse.
export function toPreciseDuration(millis: number): string {
	let seconds = Math.max(Math.round(millis / 1000), 0);
	const days = Math.floor(seconds / 86400);
	seconds -= days * 86400;
	const hours = Math.floor(seconds / 3600);
	seconds -= hours * 3600;
	const minutes = Math.floor(seconds / 60);
	seconds -= minutes * 60;

	const parts: string[] = [];
	if (days > 0) parts.push(`${days}d`);
	if (hours > 0) parts.push(`${hours}h`);
	if (minutes > 0) parts.push(`${minutes}m`);
	if (seconds > 0 || parts.length === 0) parts.push(`${seconds}s`);
	return parts.slice(0, 2).join(' ');
}

// toHumanDuration formats a duration with its largest unit, e.g. "15s", "4m",
// or "1h". Useful for coarse time measurements such as when something last
// happened.
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
