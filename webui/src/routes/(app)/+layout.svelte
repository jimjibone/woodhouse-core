<script lang="ts">
	import { HeartIcon, LampIcon, SettingsIcon } from '@lucide/svelte';
	import AppSidebar from '$lib/components/app-sidebar.svelte';
	import { type Dashboards, settingsNav, isPathActive } from '$lib/nav';
	import AppMobilebar from '$lib/components/app-mobilebar.svelte';
	import * as Breadcrumb from '$lib/components/ui/breadcrumb';
	import { Separator } from '$lib/components/ui/separator';
	import * as Sidebar from '$lib/components/ui/sidebar';
	import { page } from '$app/state';
	import { doLogout, loggedIn, userData } from '$lib/stores/auth-store';
	import ChangePasswordForm from '$lib/components/change-password-form.svelte';
	import Button from '$lib/components/ui/button/button.svelte';
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

	type Crumb = { name: string; url?: string };

	// Settings sub-pages get a two-level trail so the header says where you are
	// when the sidebar can only say "Settings".
	const crumbs: Crumb[] = $derived.by(() => {
		const path = page.url.pathname;

		if (isPathActive(path, '/settings')) {
			const section = settingsNav.find((item) => isPathActive(path, item.url));
			return section ? [{ name: 'Settings', url: '/settings' }, { name: section.name }] : [{ name: 'Settings' }];
		}

		// /profile is reached from the nav-user menu, not the dashboards array
		// (it's for any logged-in user, not just a sidebar destination), so it
		// needs its own case rather than falling through to "Unknown".
		if (isPathActive(path, '/profile')) {
			return [{ name: 'Profile' }];
		}

		const dashboard = dashboards.find((item) => isPathActive(path, item.url));
		return [{ name: dashboard?.name ?? 'Unknown' }];
	});

	// The tab title names the page you are on, not the whole trail.
	const activeDashboard: string = $derived(crumbs[crumbs.length - 1].name);

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

	// Set while the user is still on a password somebody else chose for them
	// - a new account, or one an admin has reset. Gating here rather than on
	// /profile is deliberate: the temporary password was handed over out of
	// band (spoken, messaged, written down), so it should not be enough to
	// go on using the instance indefinitely. Cleared by the fresh access
	// token the change-password form fetches on success.
	const mustResetPassword = $derived($loggedIn && $userData.reset_password);
</script>

<svelte:head>
	<title>{mustResetPassword ? 'Choose a password' : activeDashboard} · {title}</title>
</svelte:head>

{#if mustResetPassword}
	<!-- Rendered instead of the app, not over it: no sidebar, no nav, no
	page content. The only ways out are changing the password or logging
	out. -->
	<main class="mx-auto grid min-h-svh max-w-md content-center gap-6 p-4">
		<div class="grid gap-1">
			<h1 class="text-xl font-semibold">Choose a password</h1>
			<p class="text-muted-foreground text-sm">
				You are signed in as {$userData.username} with a temporary password. Choose your own before continuing.
			</p>
		</div>

		<ChangePasswordForm
			legend=""
			description=""
			currentLabel="Temporary password"
			currentHelp="The one you signed in with just now."
			submitLabel="Save and continue"
		/>

		<div class="flex justify-end">
			<Button variant="ghost" class="cursor-pointer" onclick={() => doLogout()}>Log out</Button>
		</div>
	</main>
{:else}
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
							{#each crumbs as crumb, i (crumb.name)}
								{#if i > 0}
									<Breadcrumb.Separator />
								{/if}
								<Breadcrumb.Item>
									{#if crumb.url && i < crumbs.length - 1}
										<Breadcrumb.Link href={crumb.url}>{crumb.name}</Breadcrumb.Link>
									{:else}
										<Breadcrumb.Page>{crumb.name}</Breadcrumb.Page>
									{/if}
								</Breadcrumb.Item>
							{/each}
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
{/if}
