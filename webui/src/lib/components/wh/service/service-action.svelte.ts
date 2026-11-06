import type { FieldValue, Value } from '$lib/api/v2/clients/client_service_pb';
import { sendCommand, sendWrite } from '$lib/stores/requests';
import { field } from '$lib/device';

// Sends writes and commands to one service of one endpoint and tracks whether
// one is in flight (for a pending indicator) and when one last failed (for the
// error flash). Failures are also shown as a toast by sendWrite/sendCommand.
class ServiceAction {
	#deviceID: string;
	#endpointID: string;
	#serviceID: string;
	pending: boolean = $state(false);
	error: number | null = $state(null);
	#inFlight = 0;

	constructor(deviceID: string, endpointID: string, serviceID: string) {
		this.#deviceID = deviceID;
		this.#endpointID = endpointID;
		this.#serviceID = serviceID;
	}

	// Runs an action, showing the pending state if it takes over 500ms.
	async #run(fn: () => Promise<FieldValue[] | undefined>): Promise<FieldValue[] | undefined> {
		this.#inFlight++;
		const timeout = setTimeout(() => (this.pending = true), 500);
		try {
			const result = await fn();
			if (result === undefined) {
				this.error = Date.now();
			}
			return result;
		} finally {
			clearTimeout(timeout);
			this.#inFlight--;
			if (this.#inFlight === 0) {
				this.pending = false;
			}
		}
	}

	/** Writes attribute values. Resolves with the result fields, undefined on failure. */
	write(fields: FieldValue[]): Promise<FieldValue[] | undefined> {
		return this.#run(() => sendWrite(this.#deviceID, this.#endpointID, this.#serviceID, fields));
	}

	/** Writes one attribute. */
	writeValue(id: string, value: Value): Promise<FieldValue[] | undefined> {
		return this.write([field(id, value)]);
	}

	/** Invokes a command. Resolves with the result fields, undefined on failure. */
	command(id: string, params: FieldValue[] = []): Promise<FieldValue[] | undefined> {
		return this.#run(() => sendCommand(this.#deviceID, this.#endpointID, this.#serviceID, id, params));
	}
}

export default ServiceAction;
