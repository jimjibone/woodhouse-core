<script lang="ts">
	import { HeartIcon, LampIcon, UsersIcon } from '@lucide/svelte';
	import AppSidebar, { type Dashboards } from '$lib/components/app-sidebar.svelte';
	import AppMobilebar from '$lib/components/app-mobilebar.svelte';
	import * as Breadcrumb from '$lib/components/ui/breadcrumb';
	import { Separator } from '$lib/components/ui/separator';
	import * as Sidebar from '$lib/components/ui/sidebar';
	import { page } from '$app/state';
	import { loggedIn } from '$lib/stores/auth-store';
	import { goto } from '$app/navigation';

	let { children } = $props();

	const dashboards: Dashboards = [
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
			name: 'Users',
			url: '/users',
			icon: UsersIcon
		}
	];

	let activeDashboard: string = $derived.by(() => {
		return dashboards.find((item) => item.url === page.url.pathname)?.name ?? 'Unknown';
	});

	$effect(() => {
		if (!$loggedIn) {
			const redirectTo = encodeURIComponent(page.url.pathname + page.url.search);
			goto(`/login?redirect=${redirectTo}`);
		}
	});
</script>

<Sidebar.Provider>
	<AppSidebar {dashboards} />
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
		</header>

		<div class="p-4">
			{@render children()}
		</div>

		<AppMobilebar {dashboards} />
	</Sidebar.Inset>
</Sidebar.Provider>
