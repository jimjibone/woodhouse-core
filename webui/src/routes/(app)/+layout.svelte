<script lang="ts">
	import {
		HeartIcon,
		LampIcon,
		UsersIcon,
		ChevronsLeftRightEllipsisIcon,
		LayersIcon,
		SettingsIcon
	} from '@lucide/svelte';
	import AppSidebar, { type Dashboards } from '$lib/components/app-sidebar.svelte';
	import AppMobilebar from '$lib/components/app-mobilebar.svelte';
	import * as Breadcrumb from '$lib/components/ui/breadcrumb';
	import { Separator } from '$lib/components/ui/separator';
	import * as Sidebar from '$lib/components/ui/sidebar';
	import { page } from '$app/state';
	import { loggedIn, userData } from '$lib/stores/auth-store';
	import { goto } from '$app/navigation';
	import { createConnectionContext } from '$lib/stores/connection-status.svelte';
	import { settings, loadSettings, displayName } from '$lib/stores/settings-store';

	let { children } = $props();

	// What the interface calls this server feeds the sidebar and the tab title,
	// so load it as soon as we have a session rather than only on /settings.
	$effect(() => {
		if ($loggedIn) {
			loadSettings();
		}
	});

	// "Woodhouse" unless an admin has opted into showing the instance name.
	const title = $derived(displayName($settings));

	// Settings is admin-only. The server is the enforcement point; hiding the
	// entry just keeps a page a user cannot act on out of their way.
	const dashboards: Dashboards = $derived([
		{
			name: 'Favorites',
			url: '/favorites',
			icon: HeartIcon
		},
		{
			name: 'Devices',
			url: '/devices',
			icon: LampIcon
		},
		{
			name: 'Clients',
			url: '/clients',
			icon: ChevronsLeftRightEllipsisIcon
		},
		{
			name: 'Users',
			url: '/users',
			icon: UsersIcon
		},
		{
			name: 'Groups',
			url: '/groups',
			icon: LayersIcon
		},
		...($userData.role === 'admin'
			? [
					{
						name: 'Settings',
						url: '/settings',
						icon: SettingsIcon
					}
				]
			: [])
	]);

	let activeDashboard: string = $derived.by(() => {
		return dashboards.find((item) => item.url === page.url.pathname)?.name ?? 'Unknown';
	});

	const connStatus = createConnectionContext();

	// Controls whether the indicator is in the DOM and whether it is fading out.
	let shown = $state(false);
	let fading = $state(false);

	$effect(() => {
		let fadeTimer: ReturnType<typeof setTimeout>;
		let removeTimer: ReturnType<typeof setTimeout>;

		if (connStatus.connected || connStatus.reconnecting) {
			shown = true;
			fading = false;

			if (connStatus.connected) {
				// Begin fade-out after 3s of being connected.
				fadeTimer = setTimeout(() => {
					fading = true;
					// Remove from DOM after the 1s CSS transition completes.
					removeTimer = setTimeout(() => (shown = false), 1000);
				}, 3000);
			}
		} else {
			shown = false;
			fading = false;
		}

		return () => {
			clearTimeout(fadeTimer);
			clearTimeout(removeTimer);
		};
	});

	$effect(() => {
		if (!$loggedIn) {
			const redirectTo = encodeURIComponent(page.url.pathname + page.url.search);
			goto(`/login?redirect=${redirectTo}`);
		}
	});
</script>

<svelte:head>
	<title>{activeDashboard} · {title}</title>
</svelte:head>

<Sidebar.Provider>
	<AppSidebar {dashboards} {title} />
	<Sidebar.Inset>
		<header
			class="group-has-data-[collapsible=icon]/sidebar-wrapper:h-12 flex h-16 shrink-0 items-center gap-2 transition-[width,height] ease-linear"
		>
			<div class="flex items-center gap-2 px-4">
				<Sidebar.Trigger class="-ml-1" />
				<Separator orientation="vertical" class="mr-2 data-[orientation=vertical]:h-4" />
				<Breadcrumb.Root>
					<Breadcrumb.List>
						<Breadcrumb.Item>
							<Breadcrumb.Page>{activeDashboard}</Breadcrumb.Page>
						</Breadcrumb.Item>
					</Breadcrumb.List>
				</Breadcrumb.Root>
			</div>

			{#if shown}
				<div class="ml-auto px-4 transition-opacity duration-1000" class:opacity-0={fading}>
					{#if connStatus.connected}
						<span class="flex items-center gap-1.5 text-xs text-green-600 dark:text-green-500">
							<span class="relative flex size-2">
								<span class="absolute inline-flex h-full w-full animate-ping rounded-full bg-green-500 opacity-75"
								></span>
								<span class="relative inline-flex size-2 rounded-full bg-green-500"></span>
							</span>
							Live
						</span>
					{:else}
						<span class="flex items-center gap-1.5 text-xs text-amber-500">
							<span class="relative flex size-2">
								<span class="absolute inline-flex h-full w-full animate-ping rounded-full bg-amber-400 opacity-75"
								></span>
								<span class="relative inline-flex size-2 rounded-full bg-amber-400"></span>
							</span>
							Reconnecting…
						</span>
					{/if}
				</div>
			{/if}
		</header>

		<div class="p-2">
			{@render children()}
		</div>

		<AppMobilebar {dashboards} />
	</Sidebar.Inset>
</Sidebar.Provider>
