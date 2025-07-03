import type { Value, ActionResponse } from '$lib/api/v1/clients/client_service_pb';
import { ActionResponse_ActionStatus } from '$lib/api/v1/clients/client_service_pb';
import { SendActionRequest } from '$lib/stores/requests';
import { toast } from "svelte-sonner";

class ServiceAction {
	#deviceID: string;
	#serviceID: string;
	pending: boolean = $state(false);
	error: number | null = $state(null);

	constructor(deviceID: string, serviceID: string) {
		this.#deviceID = deviceID;
		this.#serviceID = serviceID;
	}

	async send(vals: Value[]) {
		let timeout = setTimeout(() => this.pending = true, 500);
		SendActionRequest(this.#deviceID, this.#serviceID, vals, (resp: ActionResponse) => {
			clearTimeout(timeout);
			this.pending = false;
			if (resp.status >= ActionResponse_ActionStatus.TIMEOUT) {
				console.error("action failed:", resp);
				this.error = Date.now();
				toast.error("Action Failed", {
					description: resp.details
				});
			}
		});
	}

	async delayedSend(delayms: number, vals: Value[]) {
		let timeout = setTimeout(() => this.pending = true, 500);
		setTimeout(() => SendActionRequest(this.#deviceID, this.#serviceID, vals, (resp: ActionResponse) => {
			clearTimeout(timeout);
			this.pending = false;
			if (resp.status >= ActionResponse_ActionStatus.TIMEOUT) {
				console.error("action failed:", resp);
				this.error = Date.now();
				toast.error("Action Failed", {
					description: resp.details
				});
			}
		}), delayms);
	}
}

export default ServiceAction;
