<script lang="ts">
	import {
		Service,
		Service_ServiceType,
		BoolAttribute,
		TextAttribute
	} from '$lib/api/v1/clients/client_service_pb';
	import { PackageCheck, PackageOpen } from 'lucide-svelte';
	import { cn } from '$lib/utils.js';
	import { validators } from 'tailwind-merge';

	export let title: string | undefined = undefined;
	export let online: boolean;
	export let service: Service;

	$: alias = title ? title + (service.alias !== '' ? ': ' + service.alias : '') : service.alias;
	let attrAvailable: BoolAttribute | undefined;
	let attrCurrentVersion: TextAttribute | undefined;
	let attrUpdateVersion: TextAttribute | undefined;

	$: {
		for (const attr of service.attrs) {
			if (attr.id === 'available') {
				attrAvailable = attr.bool;
			} else if (attr.id === 'current_version') {
				attrCurrentVersion = attr.text;
			} else if (attr.id === 'update_version') {
				attrUpdateVersion = attr.text;
			}
		}
	}
</script>

{#if service.typ === Service_ServiceType.UPDATE}
	<!-- <div class="grid grid-cols-2 gap-4"> -->
	<div
		class={cn(
			'rounded-lg border bg-card p-2 text-card-foreground shadow-sm',
			!online && 'bg-muted'
		)}
	>
		<div class="flex flex-row gap-2">
			<div class="shrink">
				<div class="grid h-full place-content-center">
					{#if attrAvailable?.value}
					<div class="p-2 rounded-full bg-yellow-400 text-black">
						<PackageOpen/>
					</div>
					{:else}
					<div class="p-2 rounded-full bg-secondary text-secondary-foreground">
						<PackageCheck/>
					</div>
					{/if}
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
						{#if attrAvailable !== undefined}
							{#if attrAvailable.value}
							<p>
								Update available
								{#if attrUpdateVersion !== undefined} {attrUpdateVersion.value}{/if}
								{#if attrCurrentVersion !== undefined}(from {attrCurrentVersion.value}){/if}
							</p>
							{:else}
							<p>
								Up to date
								{#if attrCurrentVersion !== undefined}(version {attrCurrentVersion.value}){/if}
							</p>
							{/if}
						{/if}
					</div>
				</div>
			</div>
		</div>
	</div>
	<!-- <div class="p-4 rounded-lg shadow-lg bg-fuchsia-500">02</div>
<div class="p-4 rounded-lg shadow-lg bg-fuchsia-500">03</div>
</div> -->
{:else}
	<p>ERROR Service Type {Service_ServiceType[service.typ]} is not UPDATE</p>
{/if}
