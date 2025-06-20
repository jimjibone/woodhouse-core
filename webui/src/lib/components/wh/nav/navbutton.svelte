<script lang="ts">
	import { buttonVariants } from "$lib/components/ui/button";
	import * as Tooltip from "$lib/components/ui/tooltip";
	import { Button } from "$lib/components/ui/button";
	import { type Snippet } from "svelte";

	let {
		onclick,
		label,
		children
	} : {
		onclick: ()=>void,
		label: string
		children: Snippet<[]>
	} = $props();
</script>

<Tooltip.Provider>
	<Tooltip.Root>
		<Tooltip.Trigger {onclick} class={buttonVariants({
			variant: "ghost",
			size: "icon",
			class: "rounded-lg",
		})}>
			{@render children()}
		</Tooltip.Trigger>
		<Tooltip.Content side="right">
			<p>{label}</p>
		</Tooltip.Content>
	</Tooltip.Root>
</Tooltip.Provider>
