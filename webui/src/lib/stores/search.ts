import { writable } from "svelte/store";

export type SearchStoreType = {
	active: boolean
	query: string
};

const { subscribe, set, update } = writable<SearchStoreType>({
	active: false,
	query: ""
});

export function setSearchActive(v: boolean) {
	console.log("set active:", v)
	update((prev) => ({
		active: v,
		query: prev.query
	}));
}

export function setSearchFilter(v: string) {
	console.log("set filter:", v)
	update((prev) => ({
		active: prev.active,
		query: v
	}));
}

export const search = {
	subscribe,
	set,
	update,
};
