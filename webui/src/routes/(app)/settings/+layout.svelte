<script lang="ts">
	import { page } from '$app/state';
	import { userData } from '$lib/stores/auth-store';
	import { buttonVariants } from '$lib/components/ui/button';
	import { cn } from '$lib/utils';
	import { settingsNav, isPathActive } from '$lib/nav';

	let { children } = $props();

	// The whole section is admin-only, matching the sidebar entry. The server is
	// still the enforcement point; this just keeps pages a user cannot act on
	// out of their way.
	const isAdmin = $derived($userData.role === 'admin');
</script>

{#if isAdmin}
	<div class="flex flex-col gap-4 md:flex-row md:gap-6">
		<!--
			A column beside the content on md+, a horizontally scrollable row above
			it below that. The negative margin lets the row bleed to the edges of
			the app layout's p-2 padding so a scrolled item is not clipped.
		-->
		<nav
			aria-label="Settings"
			class="-mx-2 flex shrink-0 flex-row gap-1 overflow-x-auto px-2 pb-1 md:mx-0 md:w-48 md:flex-col md:self-start md:overflow-visible md:px-0 md:pb-0"
		>
			{#each settingsNav as item (item.url)}
				{@const active = isPathActive(page.url.pathname, item.url)}
				<a
					href={item.url}
					aria-current={active ? 'page' : undefined}
					class={cn(
						buttonVariants({ variant: 'ghost' }),
						'shrink-0 justify-start gap-2',
						active && 'bg-muted hover:bg-muted'
					)}
				>
					<item.icon class="size-4" />
					{item.name}
				</a>
			{/each}
		</nav>

		<!--
			min-w-0 stops long client IDs blowing out the flex row. The bottom
			padding clears the fixed mobile nav pill, matching the mb-20 used by
			the favorites and devices pages.
		-->
		<div class="min-w-0 flex-1 pb-20 md:pb-0">
			{@render children()}
		</div>
	</div>
{:else}
	<main class="max-w-2xl">
		<p class="text-muted-foreground text-sm">You need to be an admin to change server settings.</p>
	</main>
{/if}
