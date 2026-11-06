<script lang="ts">
	import ServiceRoot, { type StandardProps } from './service-root.svelte';
	import ServiceAction from './service-action.svelte';
	import { ListMusicIcon, PlayIcon } from '@lucide/svelte';
	import { Button } from '$lib/components/ui/button';
	import { OthersContent } from '$lib/components/wh/attributes';
	import { asList, asStruct, asText, field, fieldValue, findAttr, textValue } from '$lib/device';
	import { MediaSources } from '$lib/schema/zz_services';

	let { deviceID, endpointID, service, ...rest }: StandardProps = $props();

	const sources = $derived(
		(asList(findAttr(service, MediaSources.attributes.sources)?.value)?.items ?? []).flatMap((item) => {
			const fields = asStruct(item)?.fields;
			const id = asText(fieldValue(fields, 'id'));
			if (id === undefined) return [];
			return [{ id, name: asText(fieldValue(fields, 'name')) || id }];
		})
	);
	const canPlay = $derived(service.commands.some((c) => c.id === MediaSources.commands.play));

	// svelte-ignore state_referenced_locally
	const serviceAction = new ServiceAction(deviceID, endpointID, service.id);

	const play = (id: string) => serviceAction.command(MediaSources.commands.play, [field('id', textValue(id))]);
</script>

{#snippet icon()}
	<ListMusicIcon />
{/snippet}

{#snippet details()}
	<p>{sources.length} {sources.length === 1 ? 'source' : 'sources'}</p>
{/snippet}

<ServiceRoot
	{deviceID}
	{endpointID}
	{...rest}
	{service}
	actionPending={serviceAction.pending}
	errorSignal={serviceAction.error}
	{icon}
	{details}
>
	{#if sources.length === 0}
		<p class="text-muted-foreground">No sources</p>
	{:else}
		<ul class="grid gap-2">
			{#each sources as source (source.id)}
				<li class="flex flex-row items-center gap-2">
					<span class="grow break-words">{source.name}</span>
					{#if canPlay}
						<Button
							variant="secondary"
							size="icon"
							class="cursor-pointer"
							aria-label={'Play ' + source.name}
							onclick={() => play(source.id)}
						>
							<PlayIcon />
						</Button>
					{/if}
				</li>
			{/each}
		</ul>
	{/if}
	<OthersContent
		{deviceID}
		{endpointID}
		{service}
		action={serviceAction}
		exclude={[MediaSources.attributes.sources]}
		excludeCommands={[MediaSources.commands.play]}
	/>
</ServiceRoot>
