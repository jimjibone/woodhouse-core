import { ChevronsLeftRightEllipsisIcon, LayersIcon, SlidersHorizontalIcon, UsersIcon } from '@lucide/svelte';

export type NavItem = {
	name: string;
	url: string;
	icon: any;
};

// Aliased so the sidebar and mobile bar props read the same as before.
export type Dashboards = NavItem[];

// The Settings section's own navigation. Shared between the settings layout,
// which renders it, and the app layout, which uses it to build the breadcrumb.
export const settingsNav: NavItem[] = [
	{ name: 'General', url: '/settings/general', icon: SlidersHorizontalIcon },
	{ name: 'Clients', url: '/settings/clients', icon: ChevronsLeftRightEllipsisIcon },
	{ name: 'Users', url: '/settings/users', icon: UsersIcon },
	{ name: 'Groups', url: '/settings/groups', icon: LayersIcon }
];

// A nav entry is active for its own page and anything beneath it. The trailing
// slash stops "/devices" claiming "/devices-archive".
export function isPathActive(pathname: string, url: string): boolean {
	return pathname === url || pathname.startsWith(url + '/');
}
