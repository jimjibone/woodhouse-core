<script lang="ts">
	// Headless: shows a toast with a link to the device whenever a pairing
	// service reports a live `paired` event. Mounted once in the app layout.
	import { onMount } from 'svelte';
	import { toast } from 'svelte-sonner';
	import { goto } from '$app/navigation';
	import { Service_ServiceType } from '$lib/api/v2/clients/client_service_pb';
	import { DevicesStore, subscribeLiveEvents, type DevicesStoreDevice } from '$lib/stores/devices-stream';
	import { asText, fieldValue, findService } from '$lib/device';
	import { Pairing } from '$lib/schema/zz_services';

	let devices: DevicesStoreDevice[] = [];

	onMount(() => {
		// Live events only flow while the devices stream is subscribed, so hold
		// a subscription here for as long as the app is shown.
		const unsubDevices = DevicesStore.subscribe((update) => (devices = update.devices));
		const unsubEvents = subscribeLiveEvents((deviceID, event) => {
			if (event.id !== Pairing.events.paired) return;
			const dev = devices.find((d) => d.id === deviceID);
			if (dev !== undefined && findService(dev, event.endpointId, event.serviceId)?.typ !== Service_ServiceType.PAIRING) {
				return;
			}
			const pairedID = asText(fieldValue(event.fields, 'device_id'));
			if (pairedID === undefined || pairedID === '') return;
			const name = asText(fieldValue(event.fields, 'name'));
			toast.success('Device paired: ' + (name ? name : pairedID), {
				action: {
					label: 'Go to device',
					onClick: () => goto('/devices/' + encodeURIComponent(pairedID))
				}
			});
		});
		return () => {
			unsubEvents();
			unsubDevices();
		};
	});
</script>
