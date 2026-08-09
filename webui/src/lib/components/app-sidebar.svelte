<script lang="ts">
	import NavUser from './nav-user.svelte';
	import * as Sidebar from '$lib/components/ui/sidebar/index.js';
	import type { ComponentProps } from 'svelte';
	import { MoonIcon, SunIcon } from '@lucide/svelte';
	import { WoodhouseIcon } from '@/components/wh/icons';
	import { toggleMode, mode } from 'mode-watcher';
	import { page } from '$app/state';
	import { type Dashboards, isPathActive } from '$lib/nav';
	import { userData } from '$lib/stores/auth-store';

	let {
		ref = $bindable(null),
		collapsible = 'icon',
		dashboards = [],
		title = 'Woodhouse',
		...restProps
	}: ComponentProps<typeof Sidebar.Root> & { dashboards: Dashboards; title?: string } = $props();

	// NavUser only needs enough to render the footer button; the fullname (for
	// nicer avatar initials) lives in the users stream, not the JWT, and isn't
	// worth subscribing to here just for a footer button - it falls back to
	// username-based initials same as the profile page does before it resolves.
	const user = $derived({ username: $userData.username, fullname: '' });
</script>

<Sidebar.Root {collapsible} {...restProps}>
	<Sidebar.Header>
		<Sidebar.Menu>
			<Sidebar.MenuItem>
				<Sidebar.MenuButton size="lg">
					{#snippet child({ props })}
						<a href="/" {...props}>
							<div
								class="flex aspect-square size-8 items-center justify-center rounded-lg [[data-collapsed=true]_&]:bg-sidebar-primary [[data-collapsed=true]_&]:text-sidebar-primary-foreground transition-[background-color,color] duration-200 ease-linear"
							>
								<WoodhouseIcon class="size-5" />
							</div>
							<span class="truncate text-lg" {title}>{title}</span>
						</a>
					{/snippet}
				</Sidebar.MenuButton>
			</Sidebar.MenuItem>
		</Sidebar.Menu>
	</Sidebar.Header>
	<Sidebar.Content>
		<Sidebar.Group>
			<Sidebar.GroupLabel>Dashboards</Sidebar.GroupLabel>
			<Sidebar.Menu>
				{#each dashboards as item (item.name)}
					<Sidebar.MenuItem>
						<Sidebar.MenuButton isActive={isPathActive(page.url.pathname, item.url)}>
							{#snippet child({ props })}
								<a href={item.url} {...props}>
									<item.icon />
									<span>{item.name}</span>
								</a>
							{/snippet}
						</Sidebar.MenuButton>
					</Sidebar.MenuItem>
				{/each}
			</Sidebar.Menu>
		</Sidebar.Group>
	</Sidebar.Content>
	<Sidebar.Separator />
	<Sidebar.Footer>
		<Sidebar.Menu>
			<Sidebar.MenuItem>
				<Sidebar.MenuButton onclick={toggleMode} class="cursor-pointer">
					{#if mode.current == 'dark'}
						<MoonIcon />
					{:else}
						<SunIcon />
					{/if}
					<span>Toggle Theme</span>
				</Sidebar.MenuButton>
			</Sidebar.MenuItem>
		</Sidebar.Menu>
		<NavUser {user} />
	</Sidebar.Footer>
	<Sidebar.Rail />
</Sidebar.Root>
