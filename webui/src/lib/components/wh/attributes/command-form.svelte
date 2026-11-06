<script lang="ts">
	// A form for the parameters of a command, built from its CommandSpec.
	import type { CommandSpec, FieldValue, Value } from '$lib/api/v2/clients/client_service_pb';
	import { field, specEnumOptions, specKind, boolValue, enumValue } from '$lib/device';
	import { toHeadlineCase } from '$lib/tools/headline-case';
	import { Button } from '$lib/components/ui/button';
	import FieldInput from './field-input.svelte';

	let {
		command,
		running = false,
		onsubmit,
		oncancel
	}: {
		command: CommandSpec;
		running?: boolean;
		onsubmit: (params: FieldValue[]) => void;
		oncancel: () => void;
	} = $props();

	// Required booleans start false and required enums on their first option,
	// so the form can be submitted without touching them. Everything else
	// starts empty.
	const initial = (): Record<string, Value | undefined> => {
		const out: Record<string, Value | undefined> = {};
		for (const p of command.params) {
			if (!p.required) continue;
			const k = specKind(p.spec);
			if (k === 'bool') out[p.id] = boolValue(false);
			if (k === 'enum') {
				const first = specEnumOptions(p.spec)[0];
				if (first !== undefined) out[p.id] = enumValue(first);
			}
		}
		return out;
	};
	let drafts = $state.raw<Record<string, Value | undefined>>(initial());

	const missing = $derived(command.params.some((p) => p.required && drafts[p.id] === undefined));

	const submit = (event: SubmitEvent) => {
		event.preventDefault();
		if (missing || running) return;
		onsubmit(
			command.params.flatMap((p) => {
				const v = drafts[p.id];
				return v === undefined ? [] : [field(p.id, v)];
			})
		);
	};
</script>

<form onsubmit={submit} class="flex flex-col gap-4 pt-2">
	{#each command.params as p (p.id)}
		<div class="grid gap-1.5">
			<label class="text-sm font-medium" for={'cmd-' + command.id + '-' + p.id}>
				{toHeadlineCase(p.id)}{p.required ? ' *' : ''}
			</label>
			<FieldInput
				id={'cmd-' + command.id + '-' + p.id}
				spec={p.spec}
				value={drafts[p.id]}
				onchange={(v) => (drafts = { ...drafts, [p.id]: v })}
			/>
		</div>
	{/each}

	<div class="flex gap-2 justify-end">
		<Button type="button" variant="secondary" class="cursor-pointer" onclick={oncancel} disabled={running}>
			Cancel
		</Button>
		<Button type="submit" class="cursor-pointer" disabled={missing || running}>
			{running ? 'Running...' : 'Run'}
		</Button>
	</div>
</form>
