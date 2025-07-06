<script lang="ts">
	import "../app.css";
	import { ModeWatcher } from "mode-watcher";
	import { HeartIcon, Rows3Icon, LampIcon, BugIcon, LightbulbIcon } from "@lucide/svelte";
	import { WoodhouseIcon } from "$lib/components/wh/icons";
	import AppSidebar, { type Dashboards } from "$lib/components/app-sidebar.svelte";
	import AppMobilebar from "$lib/components/app-mobilebar.svelte";
	import * as Breadcrumb from "$lib/components/ui/breadcrumb";
	import { Separator } from "$lib/components/ui/separator";
	import * as Sidebar from "$lib/components/ui/sidebar";
	import { page } from "$app/state";
	import { Toaster } from "$lib/components/ui/sonner";
	import type { Snippet } from "svelte";

	let { children } = $props();

	const dashboards: Dashboards = [
		{
			name: "Favorites",
			url: "/favorites",
			icon: HeartIcon,
		},
		// {
		// 	name: "Services",
		// 	url: "/services",
		// 	icon: Rows3Icon,
		// },
		{
			name: "Devices",
			url: "/devices",
			icon: LampIcon,
		},
		// {
		// 	name: "Debug",
		// 	url: "/debug",
		// 	icon: BugIcon,
		// },
	];

	let activeDashboard: string = $derived.by(() => {
		return dashboards.find(item => item.url === page.url.pathname)?.name ?? 'Unknown';
	});
</script>

<ModeWatcher themeColors={{ dark: "#09090b", light: "#ffffff" }} />

<Toaster closeButton richColors position="top-right" expand={true} />

<!-- <Menubar.Root class="fixed bottom-5 self-center shadow-lg rounded-full h-12">
	<Menubar.Menu>
		<Menubar.Item on:click={filterAll} class={cn("rounded-full px-1.5 py-1.5 hover:bg-muted cursor-pointer", filterServiceTypes.length === 0 && "bg-secondary")}>
			<Asterisk class="size-6"/>
		</Menubar.Item>
		<Menubar.Item on:click={filterLightbulb} class={cn("rounded-full px-1.5 py-1.5 hover:bg-muted cursor-pointer", showServiceType(false, Service_ServiceType.LIGHTBULB) && "bg-secondary")}>
			<Lightbulb class="size-6"/>
		</Menubar.Item>
		<Menubar.Item on:click={filterClimate} class={cn("rounded-full px-1.5 py-1.5 hover:bg-muted cursor-pointer", showServiceType(false, Service_ServiceType.CLIMATE) && "bg-secondary")}>
			<Thermometer class="size-6"/>
		</Menubar.Item>
	</Menubar.Menu>
</Menubar.Root> -->

<Sidebar.Provider>
	<AppSidebar {dashboards}/>
	<Sidebar.Inset>
		<header class="group-has-data-[collapsible=icon]/sidebar-wrapper:h-12 flex h-16 shrink-0 items-center gap-2 transition-[width,height] ease-linear">
			<div class="flex items-center gap-2 px-4">
				<Sidebar.Trigger class="-ml-1" />
				<Separator orientation="vertical" class="mr-2 data-[orientation=vertical]:h-4" />
				<Breadcrumb.Root>
					<Breadcrumb.List>
						<!-- <Breadcrumb.Item class="hidden md:block">
							<Breadcrumb.Link href="#">Building Your Application</Breadcrumb.Link>
						</Breadcrumb.Item>
						<Breadcrumb.Separator class="hidden md:block" /> -->
						<Breadcrumb.Item>
							<Breadcrumb.Page>{activeDashboard}</Breadcrumb.Page>
						</Breadcrumb.Item>
					</Breadcrumb.List>
				</Breadcrumb.Root>
			</div>
		</header>
		<!-- <div class="flex flex-1 flex-col gap-4 p-4 pt-0">
			<div class="grid auto-rows-min gap-4 md:grid-cols-3">
				<div class="bg-muted/50 aspect-video rounded-xl"></div>
				<div class="bg-muted/50 aspect-video rounded-xl"></div>
				<div class="bg-muted/50 aspect-video rounded-xl"></div>
			</div>
			<div class="bg-muted/50 min-h-[100vh] flex-1 rounded-xl md:min-h-min"></div>
		</div> -->
		<div class="p-4">
			{@render children()}
		</div>

		<AppMobilebar {dashboards}/>
	</Sidebar.Inset>
</Sidebar.Provider>
