<script lang="ts">
	import { buttonVariants } from "$lib/components/ui/button";
	import * as Tooltip from "$lib/components/ui/tooltip";
	import { type Snippet } from "svelte";
	import { page } from '$app/stores';

	let {
		href,
		label,
		children
	} : {
		href: string,
		label: string
		children: Snippet<[]>
	} = $props();
</script>

<Tooltip.Provider>
	<Tooltip.Root>
		<Tooltip.Trigger>
			<a
				{href}
				aria-label="Favorites"
				class={buttonVariants({
					variant: "ghost",
					size: "icon",
					class: "rounded-lg",
				})}
				class:bg-muted={$page.url.pathname === href}
			>
				{@render children()}
			</a>
		</Tooltip.Trigger>
		<Tooltip.Content side="right">
			<p>{label}</p>
		</Tooltip.Content>
	</Tooltip.Root>
</Tooltip.Provider>
