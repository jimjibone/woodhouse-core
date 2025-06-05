<script lang="ts">
	import {
		Service,
		Service_ServiceType,
		EnumAttribute,
		DurationAttribute
	} from '$lib/api/v1/clients/client_service_pb';
	import { ServiceRoot } from '$lib/components/wh/service';
	import { Pointer, Eye, EyeOff } from 'lucide-svelte';
	import { cn } from '$lib/utils.js';

	export let title: string | undefined = undefined;
	export let online: boolean;
	export let service: Service;
	export let expandable: boolean = true;
	export let onSetFavorite: ((serviceID: string, fave: boolean) => Promise<void>) | undefined;

	$: alias = title ? title + (service.alias !== '' ? ': ' + service.alias : '') : service.alias;
	let attrState: EnumAttribute | undefined;
	let attrDuration: DurationAttribute | undefined;
	let showOptions: boolean = false;

	$: {
		for (const attr of service.attrs) {
			if (attr.id === 'state') {
				attrState = attr.enum;
			} else if (attr.id === 'duration') {
				attrDuration = attr.duration;
			}
		}
	}
</script>

{#if service.typ === Service_ServiceType.BUTTON}
	<ServiceRoot deviceName={title} online={online} service={service} {onSetFavorite}>
		<span slot="icon">
			<div class="p-2 rounded-full bg-secondary text-secondary-foreground">
				<Pointer />
			</div>
		</span>
		<span slot="details" let:expanded={showOptions}>
			<div class="flex h-full flex-col justify-center gap-0">
				<div class="flex flex-row gap-2 rounded-lg p-0">
					{#if attrState !== undefined}
						{#if attrState.value === ""}
						<p>
							No state
						</p>
						{:else}
						<p>
							{attrState.value}
						</p>
						{/if}
					{/if}
					{#if attrDuration !== undefined}
						{#if attrDuration.value > 0}
						<p class="text-muted-foreground">
							{attrDuration.value}ms
						</p>
						{/if}
					{/if}
					{#if attrState !== undefined && showOptions}
						<p class="text-muted-foreground">
							{"["}
							{#each attrState.options as opt, index}
								{#if index > 0}{", "}{/if}
								{opt}
							{/each}
							{"]"}
						</p>
					{/if}
				</div>
			</div>
		</span>
	</ServiceRoot>
{:else}
	<p>ERROR Service Type {Service_ServiceType[service.typ]} is not BUTTON</p>
{/if}
