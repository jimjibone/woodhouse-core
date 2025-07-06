<script lang="ts">
	import { cn } from "$lib/utils";
	import { page } from "$app/state";
	import type { Snippet } from "svelte";

	let {
		children,
		href
	} : {
		children: Snippet<[]>,
		href: string
	} = $props();
</script>

<a class={cn("h-10 rounded-full aspect-square hover:bg-muted/90 cursor-pointer flex items-center justify-center", page.url.pathname === href && "bg-muted")} href={href}>
	{@render children()}
</a>
