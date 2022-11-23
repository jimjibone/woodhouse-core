import { differenceInMilliseconds } from 'date-fns';
import { notifications } from './notifications';

export const defaultMinBackoffMs = 1000;
export const defaultMaxBackoffMs = 8000;

// Run a function which should be restarted occasionally with some exponential
// backoff.
export function createBackoff(debug: string, minBackoffMs: number, maxBackoffMs: number, runner: (restart: VoidFunction) => void, stopper: VoidFunction) {
	let backoffDurationMs = 0;
	let lastBackoffTime = new Date();
	let lastRestartTime = new Date();
	let running = false;

	function doStart() {
		running = true;
		runner(doRestart);
	}

	function doStop() {
		running = false;
		stopper();
	}

	function doRestart() {
		if (running) {
			// Reset the backoff duration if the backoff has not been used for a
			// suitable amount of time.
			const now = new Date();
			const dt = differenceInMilliseconds(now, lastRestartTime);
			if (dt > backoffDurationMs) {
				if (debug !== "") console.log(`${debug}: backoff reset after ${dt} ms`);
				backoffDurationMs = minBackoffMs;
			}
			lastBackoffTime = now;
			if (debug !== "") console.log(`${debug}: starting backoff for ${backoffDurationMs} ms`);
			notifications.danger(`Disconnected. Retrying in ${backoffDurationMs/1000} seconds.`, backoffDurationMs);
			setTimeout(() => {
				if (debug !== "") console.log(`${debug}: backoff finished`);
				// notifications.info("Connected!", 2000);
				backoffDurationMs = backoffDurationMs * 2
				if (backoffDurationMs > maxBackoffMs) {
					backoffDurationMs = maxBackoffMs;
				}
				if (running) {
					lastRestartTime = new Date();
					runner(doRestart);
				}
			}, backoffDurationMs);
		}
	}

	return {
		start: doStart,
		stop: doStop,
	}
}
