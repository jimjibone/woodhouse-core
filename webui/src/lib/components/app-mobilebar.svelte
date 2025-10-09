<script lang="ts">
	import * as Sidebar from "$lib/components/ui/sidebar/index.js";
	import { type Dashboards } from "./app-sidebar.svelte";
	import { MobileNavButton, MobileNavItem } from "$lib/components/wh/nav";
	import { SearchIcon, XIcon } from "@lucide/svelte";
	import { search } from "$lib/stores/search";
	import { onMount, onDestroy } from "svelte";
    import { cn } from "$lib/utils";

	let {
		dashboards = [],
	} : {
		dashboards: Dashboards
	} = $props();

	// When showSearch becomes true, focus the input
	let inputEl: HTMLInputElement | null = $state(null);
	$effect(() => {
		if ($search.active && inputEl) {
			inputEl.focus();
		}
	});

	let bottomOffset = $state(0);

	onMount(() => {
		const viewport = window.visualViewport;
		if (!viewport) return;

		const updateOffset = () => {
			const keyboardHeight = window.innerHeight - viewport.height - viewport.offsetTop;
			bottomOffset = keyboardHeight > 0 ? keyboardHeight : 0;
		};

		viewport.addEventListener('resize', updateOffset);
		viewport.addEventListener('scroll', updateOffset);
		updateOffset();

		onDestroy(() => {
			viewport.removeEventListener('resize', updateOffset);
			viewport.removeEventListener('scroll', updateOffset);
		});
	});
</script>

{#if Sidebar.useSidebar().isMobile}
	<!-- <div class="fixed bottom-5 self-center shadow-lg rounded-full min-w-12 max-w-[90%] p-1 backdrop-blur-md border"> -->
	<!-- <div class="fixed bottom-0 self-center shadow-lg w-full p-1 backdrop-blur-md border"> -->
	<div class="fixed self-center shadow-lg rounded-full min-w-12 max-w-[90%] p-1 backdrop-blur-md border" style="bottom: {20+bottomOffset}px;">
	<!-- <div class="fixed bottom-[{bottomOffset}px] self-center shadow-lg rounded-full min-w-12 max-w-[90%] p-1 backdrop-blur-md border"> -->
		{#if !$search.active}
			<div class="h-12 flex flex-row gap-1 items-center">
				{#each dashboards as item (item.name)}
					<MobileNavItem href={item.url}>
						<item.icon class="size-6"/>
					</MobileNavItem>
				{/each}
				<MobileNavButton onclick={() => $search.active = true}>
					<SearchIcon class="size-6"/>
				</MobileNavButton>
			</div>
		{:else}
			<div class="h-12 flex flex-row gap-1 items-center">
				<input
					bind:this={inputEl}
					type="search"
					placeholder="Search…"
					class="h-12 rounded-full aspect-square bg-transparent backdrop-blur-md border border-accent flex items-center justify-center"
					bind:value={$search.query}
					onkeydown={(e) => {
						if (e.key === 'Escape') {
							if ($search.query !== "") {
								$search.query = ""
							} else {
								$search.active = false
							}
						} else if (e.key === 'Enter') {
							if ($search.query !== "") {
								inputEl?.blur();
							} else {
								$search.active = false
							}
						}
					}}
				/>
				<MobileNavButton onclick={() => {
					$search.active = false;
					$search.query = "";
				}}>
					<XIcon class="size-6"/>
				</MobileNavButton>
			</div>
		{/if}
	</div>
{/if}
