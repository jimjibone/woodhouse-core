import type { TimeAttribute, TimeValue } from '$lib/api/v1/clients/client_service_pb';

export function attributeToDate(time: TimeAttribute): Date {
	const ms = Number(time.seconds) * 1000 + Math.floor(Number(time.nanos) / 1_000_000);
	return new Date(ms);
}

export function valueToDate(time: TimeValue): Date {
	const ms = Number(time.seconds) * 1000 + Math.floor(Number(time.nanos) / 1_000_000);
	return new Date(ms);
}

export function toHumanDate(date: Date): string {
	return date.toLocaleString();
}

const RELATIVE_UNITS: [Intl.RelativeTimeFormatUnit, number][] = [
	['year', 365 * 24 * 60 * 60 * 1000],
	['month', 30 * 24 * 60 * 60 * 1000],
	['day', 24 * 60 * 60 * 1000],
	['hour', 60 * 60 * 1000],
	['minute', 60 * 1000]
];

// toRelativeTime renders a date as "3 minutes ago". Pass `now` from the clock
// store so it re-renders as time passes rather than freezing at first paint.
export function toRelativeTime(date: Date, now: number): string {
	const elapsed = date.getTime() - now;
	const magnitude = Math.abs(elapsed);

	const formatter = new Intl.RelativeTimeFormat(undefined, { numeric: 'auto' });
	for (const [unit, ms] of RELATIVE_UNITS) {
		if (magnitude >= ms) {
			return formatter.format(Math.round(elapsed / ms), unit);
		}
	}
	return formatter.format(0, 'second'); // "now", rather than a jittery seconds count.
}
