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
		restart: doRestart,
	}
}

// Run a function which should be restarted occasionally with some exponential
// backoff.
export function createBackoffWithHeartbeat(debug: string, minBackoffMs: number, maxBackoffMs: number, heartbeatTimeoutMs: number, runner: (resetHeartbeat: VoidFunction, restartConnection: VoidFunction) => void, stopper: VoidFunction) {
	let backoffDurationMs = 0;
	let lastBackoffTime = new Date();
	let lastRestartTime = new Date();
	let running = false;
	let interval: number = 0;
	let lastHeartbeatTime = new Date();

	function doStart() {
		running = true;
		interval = setInterval(checkTimeout, 1000);
		lastHeartbeatTime = new Date();
		runner(doResetHeartbeat, doRestart);
	}

	function doStop() {
		running = false;
		if (interval != 0) {
			clearInterval(interval);
			interval = 0;
		}
		stopper();
	}

	function doResetHeartbeat() {
		lastHeartbeatTime = new Date();
	}

	function checkTimeout() {
		const now = new Date();
		const dt = differenceInMilliseconds(now, lastHeartbeatTime);
		if (dt > heartbeatTimeoutMs) {
			if (debug) console.log(`${debug}: heartbeat timeout after ${dt} ms`);
			doStop();
			doStart();
		}
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
					runner(doResetHeartbeat, doRestart);
				}
			}, backoffDurationMs);
		}
	}

	return {
		start: doStart,
		stop: doStop,
		restart: doRestart,
	}
}
