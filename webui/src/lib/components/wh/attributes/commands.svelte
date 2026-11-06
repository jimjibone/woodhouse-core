<script lang="ts">
	// The commands of a service as buttons. A command without parameters runs
	// on click; one with parameters opens a form built from its spec. Result
	// fields, if the command returns any, are shown under the buttons.
	import type { CommandSpec, FieldValue, Service } from '$lib/api/v2/clients/client_service_pb';
	import type ServiceAction from '$lib/components/wh/service/service-action.svelte';
	import { toHeadlineCase } from '$lib/tools/headline-case';
	import { Button } from '$lib/components/ui/button';
	import Dialog from '$lib/components/wh/ui/dialog.svelte';
	import CommandForm from './command-form.svelte';
	import ValueView from './value-view.svelte';

	let {
		service,
		action,
		exclude = []
	}: {
		service: Service;
		action: ServiceAction;
		/** Command IDs the caller renders itself. */
		exclude?: string[];
	} = $props();

	const commands = $derived(service.commands.filter((c) => !exclude.includes(c.id)));

	let formFor = $state<CommandSpec | undefined>(undefined);
	let formOpen = $state(false);
	let running = $state<string | undefined>(undefined);
	let results = $state.raw<Record<string, { command: CommandSpec; fields: FieldValue[] }>>({});

	async function run(command: CommandSpec, params: FieldValue[]): Promise<boolean> {
		running = command.id;
		const fields = await action.command(command.id, params);
		running = undefined;
		if (fields === undefined) return false;
		const { [command.id]: _old, ...rest } = results;
		results = fields.length > 0 ? { ...rest, [command.id]: { command, fields } } : rest;
		return true;
	}

	function click(command: CommandSpec) {
		if (command.params.length === 0) {
			run(command, []);
		} else {
			formFor = command;
			formOpen = true;
		}
	}

	const resultSpec = (command: CommandSpec, id: string) => command.result.find((f) => f.id === id)?.spec;
</script>

{#if commands.length > 0}
	<div class="flex flex-col gap-3 pt-4">
		<div class="text-muted-foreground border-t-2 font-semibold"></div>
		<div class="flex flex-row flex-wrap gap-2">
			{#each commands as command (command.id)}
				<Button
					variant="secondary"
					class="cursor-pointer"
					disabled={running !== undefined}
					onclick={() => click(command)}
				>
					{toHeadlineCase(command.id)}{command.params.length > 0 ? '...' : ''}
				</Button>
			{/each}
		</div>

		{#each Object.values(results) as res (res.command.id)}
			<div class="rounded-md border p-2 text-sm grid gap-1">
				<div class="font-medium">{toHeadlineCase(res.command.id)} result</div>
				<div class="grid grid-cols-[auto_1fr] gap-x-3 gap-y-0.5">
					{#each res.fields as f (f.id)}
						<div class="text-muted-foreground">{toHeadlineCase(f.id)}</div>
						<div><ValueView value={f.value} spec={resultSpec(res.command, f.id)} /></div>
					{/each}
				</div>
			</div>
		{/each}
	</div>

	<Dialog bind:open={formOpen} title={formFor ? toHeadlineCase(formFor.id) : undefined}>
		{#if formOpen && formFor}
			{@const command = formFor}
			<CommandForm
				{command}
				running={running === command.id}
				oncancel={() => (formOpen = false)}
				onsubmit={async (params) => {
					if (await run(command, params)) {
						formOpen = false;
					}
				}}
			/>
		{/if}
	</Dialog>
{/if}
