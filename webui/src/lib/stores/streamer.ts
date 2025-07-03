import type { Client } from '@connectrpc/connect';
import type { DescService } from '@bufbuild/protobuf';

export type StreamerHandler<ServiceT extends DescService> = (client: Client<ServiceT>, abortSignal: AbortSignal, heartbeat: HeartbeatHandler) => Promise<boolean>;
export type BackoffHandler = (backoff: number) => void;
export type HeartbeatHandler = () => void;

export class Streamer<ServiceT extends DescService> {
	/**
	 *
	 * @param client The Client to use for the streaming RPC.
	 * @param streamerFunc A function to run the streaming call. The function should use the abortSignal with the streaming RPC to cancel it early. The heartbeat function should be called to let the Streamer know that the connection is still alive and not to trigger an abort due to timeout. When the stream finishes the function should return true if a connection was established (to reset the backoff).
	 * @param backoffFunc A callback function which can be used to update the UI with the current backoff duration.
	 */
	constructor(client: Client<ServiceT>, streamerFunc: StreamerHandler<ServiceT>, backoffFunc: BackoffHandler) {
		this.#streamFunc = streamerFunc;
		this.#backoffFunc = backoffFunc;
		this.#retry(client);
	}

	#backoff = 1000;
	#minBackoff = 1000;
	#maxBackoff = 4000;
	#streamFunc = async (client: Client<ServiceT>, abortSignal: AbortSignal, heartbeat: HeartbeatHandler) => { return false; };
	#backoffFunc = (backoff: number) => {};
	#controller = new AbortController();

	#retry = async (client: Client<ServiceT>) => {
		// Use a timer to periodically check if the connection has died. If it
		// has then trigger the abort controller to cancel the stream and then
		// fire up another one after a backoff delay.
		let lastrx = Date.now();
		let didConnect = false;
		this.#controller = new AbortController();
		const interval = setInterval(() => {
			const now = Date.now();
			if (now - lastrx > 11000) {
				this.#controller.abort();
			}
		}, 1000);

		const onHeartbeat = () => {
			lastrx = Date.now();
		};

		const resetBackoff = await this.#streamFunc(client, this.#controller.signal, onHeartbeat);

		if (resetBackoff) {
			this.#backoff = 0;
		} else {
			if (this.#backoff === 0) {
				this.#backoff = this.#minBackoff;
			} else {
				this.#backoff = this.#backoff * 2;
				if (this.#backoff > this.#maxBackoff) {
					this.#backoff = this.#maxBackoff;
				}
			}
		}
		this.#backoffFunc(this.#backoff);
		if (this.#backoff === 0) {
			this.#retry(client);
		} else {
			setTimeout(() => {
				this.#retry(client);
			}, this.#backoff);
		}

		clearInterval(interval);
	};

	/** Triggers the streamer to stop via the abort handler. */
	stop = () => {
		this.#controller.abort();
	}
}
