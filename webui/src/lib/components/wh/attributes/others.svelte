<script lang="ts">
	import { BoolValueSchema, ValueSchema, AttributeSchema } from '$lib/api/v1/clients/client_service_pb';
	import type { Attribute } from '$lib/api/v1/clients/client_service_pb';
	import { ServiceAction } from '$lib/components/wh/service';
	import { create, toJsonString } from '@bufbuild/protobuf';
	import { BoolContent } from '$lib/components/wh/attributes';

	let {
		others,
		serviceAction
	}: {
		others: Attribute[],
		serviceAction: ServiceAction
	} = $props();

	function toHeadlineCase(input: string): string {
		const minorWords = new Set([
			"a", "an", "the", "and", "but", "or", "for", "nor",
			"on", "in", "at", "to", "from", "by", "with", "of", "over"
		]);

		const words = input.toLowerCase().split(/\s+/);

		return words
			.map((word, index) => {
			if (
				index === 0 ||
				index === words.length - 1 ||
				!minorWords.has(word)
			) {
				return word[0].toUpperCase() + word.slice(1);
			} else {
				return word;
			}
			})
			.join(" ");
	};

	let sendActionBool = async (id: string, val: boolean) => {
		serviceAction.send([
			create(ValueSchema, {
				id: id,
				bool: create(BoolValueSchema, {
					value: val
				})
			})
		]);
	};
</script>

{#if others.length > 0}
	<div class="col-span-3 text-muted-foreground border-t-2 font-semibold"></div>
	<!-- <div class="col-span-3 text-muted-foreground pt-3 border-t-2 font-semibold">Others</div> -->
	{#each others as other}
		{#if other.bool}
			<BoolContent
				class="text-muted-foreground"
				name={toHeadlineCase(other.id)}
				attr={other.bool}
				onaction={(val) => sendActionBool(other.id, val)}
			/>
		<!-- {:else if other.color}
		{:else if other.duration}
		{:else if other.enum}
		{:else if other.float}
		{:else if other.int}
		{:else if other.text}
		{:else if other.time} -->
		{:else}
			<div class="col-span-3 font-mono bg-muted px-4 py-2 rounded-md whitespace-pre overflow-x-auto">
				{toJsonString(AttributeSchema, other)}
			</div>
		{/if}
	{/each}
{/if}
