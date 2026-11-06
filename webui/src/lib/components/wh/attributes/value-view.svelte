<script lang="ts">
	// Read-only rendering of any Value, nested for lists and structs. The spec
	// is optional: when given it supplies units, field order and enum context.
	import type { Spec, Value } from '$lib/api/v2/clients/client_service_pb';
	import { asColor, formatValue, specListItem, specStructFields } from '$lib/device';
	import { toHeadlineCase } from '$lib/tools/headline-case';
	import chroma from 'chroma-js';
	import Self from './value-view.svelte';

	let {
		value,
		spec = undefined,
		depth = 0
	}: {
		value: Value | undefined;
		spec?: Spec;
		depth?: number;
	} = $props();

	const kind = $derived(value?.kind.case);
	const list = $derived(value?.kind.case === 'list' ? value.kind.value.items : []);
	const itemSpec = $derived(specListItem(spec));
	const struct = $derived(value?.kind.case === 'struct' ? value.kind.value.fields : []);
	const structSpecs = $derived(specStructFields(spec));

	// A list of structs is shown as a table, one row per item, with the
	// columns taken from the spec when there is one and from the items
	// otherwise.
	const isTable = $derived(list.length > 0 && list.every((item) => item.kind.case === 'struct'));
	const columns = $derived.by(() => {
		if (!isTable) return [];
		const fromSpec = specStructFields(itemSpec).map((f) => f.id);
		const seen = new Set(fromSpec);
		const cols = [...fromSpec];
		for (const item of list) {
			if (item.kind.case !== 'struct') continue;
			for (const f of item.kind.value.fields) {
				if (!seen.has(f.id)) {
					seen.add(f.id);
					cols.push(f.id);
				}
			}
		}
		return cols;
	});

	const cell = (item: Value, col: string): Value | undefined =>
		item.kind.case === 'struct' ? item.kind.value.fields.find((f) => f.id === col)?.value : undefined;

	const colSpec = (col: string): Spec | undefined => specStructFields(itemSpec).find((f) => f.id === col)?.spec;

	const swatch = $derived.by(() => {
		const c = asColor(value);
		if (c?.hueSat) {
			return chroma.hsv(c.hueSat.hue, Math.max(0, Math.min(1, c.hueSat.sat / 100)), 1).css();
		}
		return undefined;
	});
</script>

{#if kind === undefined}
	<span class="text-muted-foreground">Unknown</span>
{:else if kind === 'list'}
	{#if list.length === 0}
		<span class="text-muted-foreground">Empty</span>
	{:else if isTable}
		<div class="overflow-x-auto rounded-md border">
			<table class="w-full text-sm">
				<thead class="bg-muted/50 text-left text-muted-foreground">
					<tr>
						{#each columns as col (col)}
							<th class="px-2 py-1 font-medium whitespace-nowrap">{toHeadlineCase(col)}</th>
						{/each}
					</tr>
				</thead>
				<tbody>
					{#each list as item, i (i)}
						<tr class="border-t">
							{#each columns as col (col)}
								<td class="px-2 py-1 align-top">
									<Self value={cell(item, col)} spec={colSpec(col)} depth={depth + 1} />
								</td>
							{/each}
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{:else}
		<ol class="grid gap-1">
			{#each list as item, i (i)}
				<li><Self value={item} spec={itemSpec} depth={depth + 1} /></li>
			{/each}
		</ol>
	{/if}
{:else if kind === 'struct'}
	{#if struct.length === 0}
		<span class="text-muted-foreground">Empty</span>
	{:else}
		<div class="grid grid-cols-[auto_1fr] gap-x-3 gap-y-0.5 text-sm">
			{#each struct as f (f.id)}
				<div class="text-muted-foreground">{toHeadlineCase(f.id)}</div>
				<div><Self value={f.value} spec={structSpecs.find((fs) => fs.id === f.id)?.spec} depth={depth + 1} /></div>
			{/each}
		</div>
	{/if}
{:else if kind === 'color' && swatch !== undefined}
	<span class="inline-flex items-center gap-2">
		<span class="inline-block size-4 rounded-full border" style={'background-color: ' + swatch}></span>
		{formatValue(value, spec)}
	</span>
{:else}
	<span class="break-words">{formatValue(value, spec)}</span>
{/if}
