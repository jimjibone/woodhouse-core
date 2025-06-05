<script lang="ts">
	import { tick } from "svelte";
	import { ServiceRoot } from '$lib/components/wh/service'
	import { Rows3, Check, ChevronsUpDown } from 'lucide-svelte';
	import { cn } from "$lib/utils.js";

	import * as Command from "$lib/components/ui/command/index.js";
	import * as Popover from "$lib/components/ui/popover/index.js";
	import { Button } from "$lib/components/ui/button/index.js";

	import {
		Service,
		Service_ServiceType,
		ImageAttribute,
		ImageValue,
		Value,
		ImageResponse_ImageStatus,
		ActionResponse,
	} from '$lib/api/v1/clients/client_service_pb';

	import { SendImageRequest } from '$lib/stores';

	export let deviceID: string;
	export let title: string | undefined = undefined;
	export let online: boolean;
	export let service: Service;
	export let onSetFavorite: ((serviceID: string, fave: boolean) => Promise<void>) | undefined;
	export let onAction: ((serviceID: string, vals: Value[], responseHandler: (response: ActionResponse) => void) => Promise<void>) | undefined;

	let attrImage: ImageAttribute | undefined;
	let showOptions: boolean = false;
	let imageSrc = '';

	$: {
		for (const attr of service.attrs) {
			if (attr.id === 'image') {
				attrImage = attr.image;
			}
		}
	}

	let getImage = async (ev: MouseEvent) => {
		ev.stopPropagation();
		await SendImageRequest(deviceID, service.id, "image", (data) => {
			console.log("ServiceCamera data: " + data.length);
			const blob = new Blob([data], { type: 'image/jpeg' });
			imageSrc = URL.createObjectURL(blob);
		}, (status, details) => {
			console.log("ServiceCamera error: " + ImageResponse_ImageStatus[status] + ", details: " + details);
		});
	};

	let action = async (val: string) => {
		// if (onAction) {
		// 	// actionPending = true;
		// 	await onAction(service.id, [
		// 		new Value({
		// 			id: 'value',
		// 			enum: new EnumValue({
		// 				value: val
		// 			})
		// 		})
		// 	]);
		// 	// actionPending = false;
		// }
	};

	let comboOpen = false;

	// We want to refocus the trigger button when the user selects
	// an item from the list so users can continue navigating the
	// rest of the form with the keyboard.
	function closeAndFocusTrigger(currentValue: any, triggerId: string) {
		action(currentValue);
		comboOpen = false;
		tick().then(() => {
			document.getElementById(triggerId)?.focus();
		});
	}
</script>

<style>
	/* Add any necessary styles */
	.image-container {
	  text-align: center;
	  margin: 20px;
	}

	img {
	  max-width: 100%;
	  height: auto;
	}
  </style>

{#if service.typ === Service_ServiceType.CAMERA}
	<ServiceRoot deviceName={title} online={online} service={service} {onSetFavorite}>
		<span slot="icon">
			<div class="p-2 rounded-full bg-secondary text-secondary-foreground">
				{#if true}
				<Rows3 />
				{/if}
			</div>
		</span>
		<span slot="details">
			{#if attrImage !== undefined}
				<Button on:click={getImage}>Get</Button>
				<div class="image-container">
					{#if imageSrc}
						<img src={imageSrc} alt="Your Thing" />
					{:else}
						<p>Image</p>
					{/if}
				</div>
			{:else}
				<p>no image</p>
			{/if}
		</span>
	</ServiceRoot>
{:else}
	<p>ERROR Service Type {Service_ServiceType[service.typ]} is not CAMERA</p>
{/if}
