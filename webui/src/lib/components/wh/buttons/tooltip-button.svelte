<script lang="ts">
	import { buttonVariants, type ButtonSize, type ButtonVariant } from "$lib/components/ui/button";
	import * as Tooltip from "$lib/components/ui/tooltip";
	import { type Snippet } from "svelte";

	let {
		size = "default",
		variant = "default",
		disabled = false,
		tooltip,
		children,
	} : {
		size: ButtonSize
		variant: ButtonVariant,
		disabled?: boolean,
		tooltip: string,
		children: Snippet
	} = $props();
</script>

<Tooltip.Provider>
	<Tooltip.Root ignoreNonKeyboardFocus={true}>
		<Tooltip.Trigger class={buttonVariants({ size: size, variant: variant })} {disabled}>
			{@render children()}
		</Tooltip.Trigger>
		<Tooltip.Content>
			<p>{tooltip}</p>
		</Tooltip.Content>
	</Tooltip.Root>
</Tooltip.Provider>
