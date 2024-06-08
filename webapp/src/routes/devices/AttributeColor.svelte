<script lang="ts">
	import { getStores } from '$app/stores';
	import { ColorAttribute, ColorValue, Permissions, Value, Unit } from '$lib/api/v1/clients/client_service_pb';
	import { Input } from "$lib/components/ui/input/index.js";

	export let disabled: boolean;
	export let id: string;
	export let attr: ColorAttribute;
	export let onAction: (val: Value) => Promise<void> | undefined

	$: str = getStr(attr)

	let getStr = (attr: ColorAttribute) : string => {
		var str = "";
		if (attr.hueSat !== undefined) {
			str += "Hue: " + attr.hueSat.hue + ", Sat: " + attr.hueSat.sat;
			if (attr.xy !== undefined) {
				str += ", ";
			}
		}
		if (attr.xy !== undefined) {
			str += "X: " + attr.xy.x + ", Y: " + attr.xy.y;
		}
		return str;
	}
</script>

{#if disabled || attr.perms === Permissions.PERM_READONLY || attr.perms === Permissions.PERM_UNDEFINED}
	<p>{str}</p>
{:else if attr.perms === Permissions.PERM_READWRITE}
<p>{str}</p>
{:else if attr.perms === Permissions.PERM_WRITEONLY}
	<p>WO: {str}</p>
{:else}
	<p>UNKNOWN {str}</p>
{/if}
