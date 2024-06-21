<script lang="ts">
	import {
		Service,
		Service_ServiceType,
		EnumAttribute
	} from '$lib/api/v1/clients/client_service_pb';
	import { Pointer, Eye, EyeOff } from 'lucide-svelte';
	import { cn } from '$lib/utils.js';

	export let title: string | undefined = undefined;
	export let online: boolean;
	export let service: Service;
	export let expandable: boolean = true;

	$: alias = title ? title + (service.alias !== '' ? ': ' + service.alias : '') : service.alias;
	let attrState: EnumAttribute | undefined;
	let showOptions: boolean = false;

	$: {
		for (const attr of service.attrs) {
			if (attr.id === 'state') {
				attrState = attr.enum;
			}
		}
	}
</script>

{#if service.typ === Service_ServiceType.BUTTON}
	<div
		class={cn(
			'rounded-lg border bg-card p-2 text-card-foreground shadow-sm',
			!online && 'bg-muted'
		)}
	>
		<div class="flex flex-row gap-2">
			<div class="shrink">
				<div class="grid h-full place-content-center">
					<div class="p-2 rounded-full bg-secondary text-secondary-foreground">
					{#if true}
					<Pointer />
					{/if}
					</div>
				</div>
			</div>
			<div class="grow">
				<div class="flex h-full flex-col justify-center gap-0">
					{#if alias !== ''}
						<div class="rounded-lg p-0">
							<p class="font-semibold">{alias}</p>
						</div>
					{/if}
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
			</div>
			{#if expandable}
			<div class="shrink">
				<div class="grid h-full place-content-center">
					<button class="p-2 rounded-full text-secondary-foreground" on:click={() => showOptions = !showOptions}>
					{#if showOptions}
					<EyeOff class="size-4" />
					{:else}
					<Eye class="size-4" />
					{/if}
					</button>
				</div>
			</div>
			{/if}
		</div>
	</div>
{:else}
	<p>ERROR Service Type {Service_ServiceType[service.typ]} is not BUTTON</p>
{/if}
